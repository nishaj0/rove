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
	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/gamemon"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/networkwatch"
	runtimecfg "cloud-saver/backend/internal/runtime"
	"cloud-saver/backend/internal/runtimepriority"
	"cloud-saver/backend/internal/snapshots"
	"cloud-saver/backend/internal/syncer"
	googledrivestore "cloud-saver/backend/internal/syncer/drive"
	"cloud-saver/backend/internal/tray"
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
	snapshots       *snapshots.Service
	googleDriveAuth *googledriveauth.Service
	debug           bool
	httpSrv         *http.Server
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stopOnce        sync.Once
	stopErr         error
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
	if notice := cfg.RecoveryNotice(); notice != "" {
		_ = logs.Write("warning", "settings_recovered", notice)
		_ = logs.WriteActivity(models.ActivityEvent{
			Category: "settings",
			Severity: "warning",
			Title:    "Settings repaired after update",
			Detail:   notice,
		})
		if opts.Debug {
			log.Printf("[settings:recovered] %s", notice)
		}
	}

	engine := syncer.New(cfg, logs, opts.Debug)
	googleDriveAuth := googledriveauth.New(cfg, logs, opts.Debug)
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
	snapshotSvc := snapshots.New(cfg, logs)

	a := &Application{
		cfg:             cfg,
		logs:            logs,
		engine:          engine,
		watcher:         watchMgr,
		monitor:         monitor,
		tray:            trayMgr,
		backup:          backupSvc,
		snapshots:       snapshotSvc,
		googleDriveAuth: googleDriveAuth,
		debug:           opts.Debug,
	}
	a.monitor.SetOnRunningChanged(func(runningIDs []string) {
		if err := runtimepriority.SetGameplayMode(len(runningIDs) > 0); err != nil {
			a.logEvent("warning", "performance", err.Error())
		}
		a.engine.UpdateRunning(runningIDs)
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
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := networkwatch.Watch(ctx, a.engine.RetryConnectivity); err != nil {
			a.logEvent("warning", "network", err.Error())
		}
	}()
	a.logEvent("info", "app", fmt.Sprintf("application starting (noTray=%t)", noTray))

	runningIDs := a.monitor.RunningIDs()
	_ = runtimepriority.SetGameplayMode(len(runningIDs) > 0)
	a.engine.UpdateRunning(runningIDs)
	if err := a.reloadWatcherForRunningGames(runningIDs); err != nil {
		a.logEvent("error", "watcher", err.Error())
	}

	// keep watcher roots in sync when game list changes via api.
	apiSvc := api.New(
		a.cfg,
		a.engine,
		a.logs,
		a.backup,
		a.monitor,
		a.snapshots,
		a.googleDriveAuth,
		a.debug,
		func(_ []models.Game) {
			a.monitor.ForcePoll()
			running := a.monitor.RunningIDs()
			a.engine.UpdateRunning(running)
			if err := a.reloadWatcherForRunningGames(running); err != nil {
				a.logEvent("error", "watcher", err.Error())
			}
		},
		func() {
			if a.cancel != nil {
				a.cancel()
			}
			a.tray.Quit()
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
	sendTrayStatus := func() {
		engineStatus := a.engine.Status()
		state := "Protected"
		if !engineStatus.ServiceControl.ProtectionEnabled || engineStatus.ServiceControl.PauseScope != models.PauseNone {
			state = "Paused"
		} else if engineStatus.Provider.State == models.ProviderOffline {
			state = "Offline"
		} else if engineStatus.Provider.State == models.ProviderReconnecting {
			state = "Reconnecting"
		} else if engineStatus.ConflictCount > 0 || engineStatus.LastError != "" {
			state = "Needs attention"
		} else if engineStatus.ActiveJobID != "" {
			state = "Protecting"
		} else if len(a.monitor.RunningIDs()) > 0 {
			state = "Playing"
		}
		update := tray.Status{
			PlayingGames: a.runningGameNames(a.monitor.RunningIDs()),
			State:        state,
			Provider:     string(engineStatus.Provider.State),
			PendingCount: engineStatus.Pending,
			LastLocal:    engineStatus.LastLocalProtection,
			LastCloud:    engineStatus.LastCloudUpload,
			Paused:       state == "Paused",
			Offline:      engineStatus.Provider.State == models.ProviderOffline || engineStatus.Provider.State == models.ProviderUnavailable,
			HasConflicts: engineStatus.ConflictCount > 0,
		}
		select {
		case statusUpdates <- update:
		default:
		}
	}
	sendTrayStatus()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.engine.StatusUpdates():
				sendTrayStatus()
			}
		}
	}()

	a.tray.Run(
		statusUpdates,
		tray.Actions{
			OpenUI: func() {
				if err := openUI(); err != nil {
					a.logEvent("error", "tray", fmt.Sprintf("open app failed: %v", err))
				}
			},
			ProtectNow: func() {
				a.engine.Trigger(syncer.TriggerReasonManual)
			},
			RecoveryPoint: func() {
				for _, id := range a.monitor.RunningIDs() {
					if game, ok := a.cfg.FindGame(id); ok {
						a.engine.CreateRecoveryPoint(id, game.Name+" tray Recovery Point")
					}
				}
			},
			Retry: func() {
				a.engine.RetryNow()
			},
			PauseUploads: func() {
				control := a.engine.GlobalControl()
				control.PauseScope = models.PauseUploads
				control.Reason = "Cloud uploads paused from the tray"
				_ = a.engine.SetServiceControl("", control)
			},
			PauseAll: func() {
				a.engine.Stop()
			},
			PauseGame: func() {
				ids := a.monitor.RunningIDs()
				if len(ids) != 1 {
					return
				}
				control := a.engine.GameControl(ids[0])
				control.PauseScope = models.PauseAll
				control.Reason = "Game Cloud Sync paused from the tray"
				_ = a.engine.SetServiceControl(ids[0], control)
			},
			Resume: func() {
				a.engine.Resume()
			},
			Review: func() {
				_ = openUI()
			},
			Exit: func() {
				if a.cancel != nil {
					a.cancel()
				}
			},
		},
	)
	return a.Stop()
}

// gracefully tears down watcher, http server, logging, and worker goroutines.
func (a *Application) Stop() error {
	a.stopOnce.Do(func() {
		if a.cancel != nil {
			a.cancel()
		}
		a.logEvent("info", "app", "application stopping")
		if a.watcher != nil {
			if err := a.watcher.Close(); err != nil && a.stopErr == nil {
				a.stopErr = err
			}
		}
		if a.httpSrv != nil {
			// Stop accepting API traffic before waiting on workers, but never
			// let a stuck loopback client block an application update forever.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := a.httpSrv.Shutdown(shutdownCtx); err != nil && a.stopErr == nil {
				a.stopErr = err
			}
			cancel()
		}
		// Workers may still write their final durable state and diagnostics.
		// Keep SQLite open until every goroutine has observed cancellation.
		a.wg.Wait()
		if a.logs != nil {
			if err := a.logs.Close(); err != nil && a.stopErr == nil {
				a.stopErr = err
			}
		}
	})
	return a.stopErr
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
		"Rove.exe",
		"Cloud Saver.exe",
		filepath.Join("..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "debug", "Rove.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "release", "Rove.exe"),
		filepath.Join("..", "app", "src-tauri", "target", "release", "Cloud Saver.exe"),
		filepath.Join("app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
		filepath.Join("app", "src-tauri", "target", "debug", "Rove.exe"),
		filepath.Join("app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
		filepath.Join("app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
		filepath.Join("app", "src-tauri", "target", "release", "Rove.exe"),
		filepath.Join("app", "src-tauri", "target", "release", "Cloud Saver.exe"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "cloud-saver-app.exe"),
			filepath.Join(exeDir, "Rove.exe"),
			filepath.Join(exeDir, "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "Rove.exe"),
			filepath.Join(exeDir, "..", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "Rove.exe"),
			filepath.Join(exeDir, "..", "..", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "debug", "Rove.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "release", "Rove.exe"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "target", "release", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "debug", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "debug", "Rove.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "debug", "Cloud Saver.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "release", "cloud-saver-app.exe"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "target", "release", "Rove.exe"),
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
