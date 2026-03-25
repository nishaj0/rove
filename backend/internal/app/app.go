package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"cloud-saver/backend/internal/api"
	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
	"cloud-saver/backend/internal/tray"
	"cloud-saver/backend/internal/watcher"
)

type Application struct {
	cfg     *config.Manager
	logs    *logstore.Store
	engine  *syncer.Engine
	watcher *watcher.Manager
	tray    *tray.Manager
	backup  *backup.Service
	httpSrv *http.Server
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

type Options struct {
	ConfigPath string
	NoTray     bool
}

func New(opts Options) (*Application, error) {
	cfg, err := config.New(opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	dbRoot := filepath.Dir(cfg.Path())
	logs, err := logstore.Open(dbRoot)
	if err != nil {
		return nil, err
	}

	engine := syncer.New(cfg, logs)
	watchMgr := watcher.New()
	trayMgr := tray.New()
	backupSvc := backup.New()

	a := &Application{
		cfg:     cfg,
		logs:    logs,
		engine:  engine,
		watcher: watchMgr,
		tray:    trayMgr,
		backup:  backupSvc,
	}
	return a, nil
}

func (a *Application) Run(ctx context.Context, noTray bool) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	a.engine.Start(ctx)

	if err := a.watcher.Reload(a.cfg.Get().Games, func(path string) {
		a.engine.NotifyFileEvent(path)
	}); err != nil {
		_ = a.logs.Write("error", "watcher", err.Error())
	}

	exePath, _ := os.Executable()
	apiSvc := api.New(
		a.cfg,
		a.engine,
		a.logs,
		a.backup,
		exePath,
		func(games []models.Game) {
			if err := a.watcher.Reload(games, func(path string) {
				a.engine.NotifyFileEvent(path)
			}); err != nil {
				_ = a.logs.Write("error", "watcher", err.Error())
			}
		},
	)
	a.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", a.cfg.Get().Port),
		Handler: apiSvc.Handler(),
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			_ = a.logs.Write("error", "http", err.Error())
		}
	}()

	if noTray {
		<-ctx.Done()
		return a.Stop()
	}

	a.tray.Run(
		func() { _ = openUI() },
		func() {
			if a.engine.Status().Running {
				a.engine.Stop()
				a.tray.SetRunning(false)
			} else {
				a.engine.Resume()
				a.tray.SetRunning(true)
			}
		},
		func() {
			_ = a.Stop()
		},
	)
	return nil
}

func (a *Application) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
	if a.httpSrv != nil {
		_ = a.httpSrv.Shutdown(context.Background())
	}
	if a.logs != nil {
		_ = a.logs.Close()
	}
	a.wg.Wait()
	return nil
}

func openUI() error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://127.0.0.1:7331/status")
	return cmd.Start()
}
