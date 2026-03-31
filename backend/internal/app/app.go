package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"cloud-saver/backend/internal/api"
	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/gamemon"
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
	monitor *gamemon.Monitor
	tray    *tray.Manager
	backup  *backup.Service
	debug   bool
	httpSrv *http.Server
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

type Options struct {
	ConfigPath string
	NoTray     bool
	Debug      bool
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

	engine := syncer.New(cfg, logs, opts.Debug)
	watchMgr := watcher.New(func(err error) {
		if err != nil {
			_ = logs.Write("error", "watcher", err.Error())
			if opts.Debug {
				log.Printf("[watcher:error] %v", err)
			}
		}
	})
	monitor := gamemon.New(cfg, logs, nil, func(err error) {
		if err != nil {
			_ = logs.Write("error", "gamemon", err.Error())
			if opts.Debug {
				log.Printf("[gamemon:error] %v", err)
			}
		}
	})
	trayMgr := tray.New()
	backupSvc := backup.New()

	a := &Application{
		cfg:     cfg,
		logs:    logs,
		engine:  engine,
		watcher: watchMgr,
		monitor: monitor,
		tray:    trayMgr,
		backup:  backupSvc,
		debug:   opts.Debug,
	}
	a.monitor.SetOnRunningChanged(func(runningIDs []string) {
		if err := a.reloadWatcherForRunningGames(runningIDs); err != nil {
			a.logEvent("error", "watcher", err.Error())
		}
	})
	a.logEvent("info", "app", fmt.Sprintf("application initialized (config=%s db=%s debug=%t)", cfg.Path(), logs.Path(), opts.Debug))
	return a, nil
}

func (a *Application) Run(ctx context.Context, noTray bool) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	a.engine.Start(ctx)
	a.monitor.Start(ctx)
	a.logEvent("info", "app", fmt.Sprintf("application starting (noTray=%t)", noTray))

	if err := a.reloadWatcherForRunningGames(a.monitor.RunningIDs()); err != nil {
		a.logEvent("error", "watcher", err.Error())
	}

	exePath, _ := os.Executable()
	// keep watcher roots in sync when game list changes via api.
	apiSvc := api.New(
		a.cfg,
		a.engine,
		a.logs,
		a.backup,
		a.monitor,
		exePath,
		a.debug,
		func(_ []models.Game) {
			a.monitor.ForcePoll()
			if err := a.reloadWatcherForRunningGames(a.monitor.RunningIDs()); err != nil {
				a.logEvent("error", "watcher", err.Error())
			}
		},
	)
	a.logEvent("info", "http", fmt.Sprintf("api listening on %s", a.httpSrvAddr()))
	a.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", a.cfg.Get().Port),
		Handler: apiSvc.Handler(),
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logEvent("error", "http", err.Error())
		}
	}()

	if noTray {
		// allow headless mode for local dev and testing.
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
	a.logEvent("info", "app", "application stopping")
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
	if a.httpSrv != nil {
		// stop accepting api traffic before waiting on goroutines.
		_ = a.httpSrv.Shutdown(context.Background())
	}
	if a.logs != nil {
		_ = a.logs.Close()
	}
	a.wg.Wait()
	return nil
}

func (a *Application) httpSrvAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.Get().Port)
}

func (a *Application) reloadWatcherForRunningGames(runningIDs []string) error {
	if len(runningIDs) == 0 {
		return a.watcher.Reload(nil, func(path string) {
			a.engine.NotifyFileEvent(path)
		})
	}

	runningSet := make(map[string]struct{}, len(runningIDs))
	for _, id := range runningIDs {
		runningSet[id] = struct{}{}
	}

	settings := a.cfg.Get()
	activeGames := make([]models.Game, 0, len(runningSet))
	for _, game := range settings.Games {
		if _, ok := runningSet[game.ID]; ok {
			activeGames = append(activeGames, game)
		}
	}

	return a.watcher.Reload(activeGames, func(path string) {
		a.engine.NotifyFileEvent(path)
	})
}

func (a *Application) logEvent(level, event, message string) {
	if a.logs != nil {
		_ = a.logs.Write(level, event, message)
	}
	if a.debug {
		log.Printf("[%s:%s] %s", level, event, message)
	}
}

func openUI() error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://127.0.0.1:7331/status")
	return cmd.Start()
}
