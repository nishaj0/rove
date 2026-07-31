package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud-saver/backend/internal/api"
	googledriveauth "cloud-saver/backend/internal/auth/googledrive"
	"cloud-saver/backend/internal/autobackup"
	"cloud-saver/backend/internal/autostart"
	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/branding"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/gamemon"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
	runtimecfg "cloud-saver/backend/internal/runtime"
	"cloud-saver/backend/internal/snapshots"
	"cloud-saver/backend/internal/syncer"
	googledrivestore "cloud-saver/backend/internal/syncer/drive"
	"cloud-saver/backend/internal/tray"
	"cloud-saver/backend/internal/updater"
	"cloud-saver/backend/internal/watcher"
)

// coordinates daemon services and runtime lifecycle.
type Application struct {
	cfg             *config.Manager
	logs            *logstore.Store
	engine          *syncer.Engine
	watcher         *watcher.Manager
	monitor         *gamemon.Monitor
	tray            *tray.Manager
	backup          *backup.Service
	autoBackup      *autobackup.Service
	snapshots       *snapshots.Service
	googleDriveAuth *googledriveauth.Service
	debug           bool
	httpSrv         *http.Server
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// captures startup configuration for daemon runtime.
type Options struct {
	ConfigPath string
	NoTray     bool
	Debug      bool
}

// initializes config, storage, sync engine, monitor, watcher, api, and tray wiring.
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
	googleDriveAuth := googledriveauth.New(cfg)
	engine.SetStoreFactory(func(ctx context.Context, settings models.Settings) (syncer.RevisionStore, error) {
		if settings.CloudProvider == models.CloudProviderGoogleDrive {
			client, err := googleDriveAuth.HTTPClient(ctx)
			if err != nil {
				return nil, err
			}
			return googledrivestore.New(ctx, client)
		}
		return syncer.NewFolderStore(settings.SyncRoot), nil
	})
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
	autoBackupSvc := autobackup.New(cfg, logs, backupSvc, opts.Debug)
	snapshotSvc := snapshots.New(cfg, logs)

	a := &Application{
		cfg:             cfg,
		logs:            logs,
		engine:          engine,
		watcher:         watchMgr,
		monitor:         monitor,
		tray:            trayMgr,
		backup:          backupSvc,
		autoBackup:      autoBackupSvc,
		snapshots:       snapshotSvc,
		googleDriveAuth: googleDriveAuth,
		debug:           opts.Debug,
	}
	a.monitor.SetOnRunningChanged(func(runningIDs []string) {
		a.autoBackup.UpdateRunning(runningIDs)
		a.snapshots.UpdateRunning(runningIDs)
		if err := a.reloadWatcherForRunningGames(runningIDs); err != nil {
			a.logEvent("error", "watcher", err.Error())
		}
	})
	a.logEvent("info", "app", fmt.Sprintf("application initialized (config=%s db=%s debug=%t)", cfg.Path(), logs.Path(), opts.Debug))
	return a, nil
}

// starts background services and blocks until shutdown or tray exit.
func (a *Application) Run(ctx context.Context, noTray bool) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	a.engine.Start(ctx)
	a.monitor.Start(ctx)
	a.autoBackup.Start(ctx)
	a.snapshots.Start(ctx)
	a.logEvent("info", "app", fmt.Sprintf("application starting (noTray=%t)", noTray))

	runningIDs := a.monitor.RunningIDs()
	a.autoBackup.UpdateRunning(runningIDs)
	a.snapshots.UpdateRunning(runningIDs)
	if err := a.reloadWatcherForRunningGames(runningIDs); err != nil {
		a.logEvent("error", "watcher", err.Error())
	}

	daemonExePath, _ := os.Executable()
	startupCommand := startupCommandTarget(daemonExePath)
	if a.cfg.Get().Autostart && startupCommand == "" {
		a.logEvent("error", "autostart", "startup target not found; autostart cannot be enabled")
	} else if err := autostart.SetEnabled(branding.AutostartKeyName, startupCommand, a.cfg.Get().Autostart); err != nil {
		a.logEvent("error", "autostart", err.Error())
	}
	updaterClient := updater.New()
	// keep watcher roots in sync when game list changes via api.
	apiSvc := api.New(
		a.cfg,
		a.engine,
		a.logs,
		a.backup,
		a.monitor,
		a.snapshots,
		a.googleDriveAuth,
		daemonExePath,
		startupCommand,
		updaterClient,
		a.debug,
		func(_ []models.Game) {
			a.monitor.ForcePoll()
			running := a.monitor.RunningIDs()
			a.autoBackup.UpdateRunning(running)
			a.snapshots.UpdateRunning(running)
			if err := a.reloadWatcherForRunningGames(running); err != nil {
				a.logEvent("error", "watcher", err.Error())
			}
		},
	)
	a.logEvent("info", "http", fmt.Sprintf("api listening on %s", a.httpSrvAddr()))
	a.httpSrv = &http.Server{
		Addr:    runtimecfg.LoopbackAddr(a.cfg.Get().Port),
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

	statusUpdates := make(chan tray.Status, 1)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runningIDs := a.monitor.RunningIDs()
				update := tray.Status{
					PlayingGames: a.runningGameNames(runningIDs),
				}
				select {
				case statusUpdates <- update:
				default:
				}
			}
		}
	}()

	a.tray.Run(
		statusUpdates,
		func() {
			if err := openUI(); err != nil {
				a.logEvent("error", "tray", fmt.Sprintf("open app failed: %v", err))
			}
		},
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
			a.engine.Trigger(syncer.TriggerReasonManual)
		},
		func() {
			_ = a.Stop()
		},
	)
	return nil
}

// gracefully tears down watcher, http server, logging, and worker goroutines.
func (a *Application) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}
	a.logEvent("info", "app", "application stopping")
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
	if a.autoBackup != nil {
		a.autoBackup.Stop()
	}
	if a.snapshots != nil {
		a.snapshots.Stop()
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

// builds the loopback bind address from current config port.
func (a *Application) httpSrvAddr() string {
	return runtimecfg.LoopbackAddr(a.cfg.Get().Port)
}

// limits watcher roots to currently running tracked games.
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
	// filter tracked games against running ids so watcher load stays minimal.
	for _, game := range settings.Games {
		if _, ok := runningSet[game.ID]; ok {
			activeGames = append(activeGames, game)
		}
	}

	return a.watcher.Reload(activeGames, func(path string) {
		a.engine.NotifyFileEvent(path)
	})
}

// writes structured logs and mirrors to stdout when debug is enabled.
func (a *Application) logEvent(level, event, message string) {
	if a.logs != nil {
		_ = a.logs.Write(level, event, message)
	}
	if a.debug {
		log.Printf("[%s:%s] %s", level, event, message)
	}
}

func (a *Application) runningGameNames(runningIDs []string) []string {
	if len(runningIDs) == 0 {
		return nil
	}
	runningSet := make(map[string]struct{}, len(runningIDs))
	for _, id := range runningIDs {
		runningSet[id] = struct{}{}
	}

	settings := a.cfg.Get()
	names := make([]string, 0, len(runningIDs))
	for _, game := range settings.Games {
		if _, ok := runningSet[game.ID]; ok {
			names = append(names, game.Name)
		}
	}
	return names
}

// opens desktop app executable path without falling back to browser endpoints.
func openUI() error {
	if uiPath, ok := desktopAppExecutable(); ok {
		cmd := exec.Command(uiPath)
		return cmd.Start()
	}

	return fmt.Errorf("desktop app executable not found")
}

func desktopAppExecutable() (string, bool) {
	candidates := []string{
		"cloud-saver-app.exe",
		"Cloud Saver.exe",
		filepath.Join("..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "release", "Cloud Saver.exe"),
		filepath.Join("app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
		filepath.Join("app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
		filepath.Join("app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
		filepath.Join("app", "src-tauri", "target", "release", "Cloud Saver.exe"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "cloud-saver-app.exe"),
			filepath.Join(exeDir, "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "release", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "release", "Cloud Saver.exe"),
		)
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, path := range candidates {
		normalized := filepath.Clean(path)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		if resolved, ok := resolveExecutableCandidate(normalized); ok {
			return resolved, true
		}
	}
	return "", false
}

// prefers desktop app startup target and falls back to daemon executable path.
func startupCommandTarget(daemonExePath string) string {
	if uiPath, ok := desktopAppExecutable(); ok {
		return uiPath
	}
	if resolved, ok := resolveExecutableCandidate(daemonExePath); ok {
		return resolved
	}
	return ""
}

// returns an absolute executable path when the candidate exists on disk.
func resolveExecutableCandidate(candidate string) (string, bool) {
	if strings.TrimSpace(candidate) == "" {
		return "", false
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(absolute)
	if err != nil || info.IsDir() {
		return "", false
	}
	return absolute, true
}
