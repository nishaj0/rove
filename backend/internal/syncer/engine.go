package syncer

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

// stores lightweight metadata used for change detection.
type fileFingerprint struct {
	ModTime int64
	Size    int64
}

// carries queue metadata for manual, watcher, or interval requests.
type syncTrigger struct {
	reason string
	gameID string
}

// schedules sync cycles and tracks runtime status.
type Engine struct {
	cfg     *config.Manager
	logs    *logstore.Store
	debug   bool
	mu      sync.RWMutex
	status  models.Status
	trigger chan syncTrigger
	tracked map[string]fileFingerprint
	pending atomic.Int64
	started atomic.Bool
	running atomic.Bool
	paused  atomic.Bool
}

// constructs an engine bound to config and sqlite log storage.
func New(cfg *config.Manager, logs *logstore.Store, debug bool) *Engine {
	return &Engine{
		cfg:     cfg,
		logs:    logs,
		debug:   debug,
		trigger: make(chan syncTrigger, 1),
		tracked: make(map[string]fileFingerprint),
	}
}

// launches the background loop and marks engine as running.
func (e *Engine) Start(ctx context.Context) {
	if !e.started.CompareAndSwap(false, true) {
		e.Resume()
		return
	}
	e.running.Store(true)
	e.paused.Store(false)
	e.setRunning(true)
	_ = e.logs.Write("info", "sync", "sync engine started")
	go e.loop(ctx)
}

// pauses sync execution without tearing down the loop goroutine.
func (e *Engine) Stop() {
	e.paused.Store(true)
	e.running.Store(false)
	e.setRunning(false)
	_ = e.logs.Write("info", "sync", "sync engine paused")
}

// re-enables sync processing and enqueues a catch-up cycle.
func (e *Engine) Resume() {
	e.paused.Store(false)
	e.running.Store(true)
	e.setRunning(true)
	e.Trigger("resume")
	_ = e.logs.Write("info", "sync", "sync engine resumed")
}

// requests a full sync cycle and coalesces duplicate queue entries.
func (e *Engine) Trigger(reason string) {
	if reason == "" {
		reason = "manual"
	}
	// keep trigger queue bounded; a single queued run is enough.
	select {
	case e.trigger <- syncTrigger{reason: reason}:
	default:
	}
}

// requests a game-scoped sync cycle when a specific game changes.
func (e *Engine) TriggerGame(gameID, reason string) {
	if gameID == "" {
		return
	}
	if reason == "" {
		reason = "manual-game"
	}
	select {
	case e.trigger <- syncTrigger{reason: reason, gameID: gameID}:
	default:
	}
}

// increments pending count and nudges the engine via watcher reason.
func (e *Engine) NotifyFileEvent(_ string) {
	e.pending.Add(1)
	e.Trigger("watcher")
}

// returns an immutable snapshot for /status responses.
func (e *Engine) Status() models.Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copy := e.status
	copy.Pending = int(e.pending.Load())
	return copy
}

// waits for interval ticks or trigger events and runs one cycle per request.
func (e *Engine) loop(ctx context.Context) {
	for {
		settings := e.cfg.Get()
		interval := time.Duration(settings.SyncIntervalSeconds) * time.Second
		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			e.Stop()
			return
		case req := <-e.trigger:
			// consume queued trigger immediately and skip waiting for timer tick.
			timer.Stop()
			e.runOnce(req.reason, req.gameID)
		case <-timer.C:
			e.runOnce("interval", "")
		}
	}
}

// performs one sync pass, optionally scoped to a single game id.
func (e *Engine) runOnce(reason, targetGameID string) {
	if e.paused.Load() {
		return
	}
	start := time.Now()
	settings := e.cfg.Get()
	games := settings.Games
	if targetGameID != "" {
		games = nil
		for _, game := range settings.Games {
			if game.ID == targetGameID {
				games = append(games, game)
				break
			}
		}
		if len(games) == 0 {
			e.noteError("sync", fmt.Sprintf("sync target game not found: %s", targetGameID))
			return
		}
	}
	changed := 0
	e.setLastError("")
	if e.debug {
		scope := "all"
		if targetGameID != "" {
			scope = "game=" + targetGameID
		}
		_ = e.logs.Write("debug", "sync", fmt.Sprintf("sync cycle start: reason=%s scope=%s games=%d", reason, scope, len(games)))
	}

	// detect changes by comparing mtime+size fingerprints so edits and rewrites are both captured.
	for _, game := range games {
		root := game.Path
		_ = filepath.WalkDir(game.Path, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				e.noteError("sync", fmt.Sprintf("scan failed for %s: %v", root, walkErr))
				return nil
			}
			if d.IsDir() {
				if shouldIgnore(path, settings.IgnorePatterns) {
					return filepath.SkipDir
				}
				return nil
			}
			if shouldIgnore(path, settings.IgnorePatterns) {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				e.noteError("sync", fmt.Sprintf("file info failed for %s: %v", path, err))
				return nil
			}
			key := game.ID + "::" + path
			fp := fileFingerprint{ModTime: info.ModTime().UnixNano(), Size: info.Size()}

			e.mu.Lock()
			// track fingerprints per game to avoid collisions for same filename across libraries.
			prev, ok := e.tracked[key]
			if !ok || prev != fp {
				e.tracked[key] = fp
				changed++
			}
			e.mu.Unlock()
			return nil
		})
	}

	if changed > 0 {
		if targetGameID != "" {
			_ = e.logs.Write("info", "sync", fmt.Sprintf("sync cycle (%s, game=%s): %d file(s) changed", reason, targetGameID, changed))
		} else {
			_ = e.logs.Write("info", "sync", fmt.Sprintf("sync cycle (%s): %d file(s) changed", reason, changed))
		}
	} else if e.debug {
		if targetGameID != "" {
			_ = e.logs.Write("debug", "sync", fmt.Sprintf("sync cycle (%s, game=%s): no file changes", reason, targetGameID))
		} else {
			_ = e.logs.Write("debug", "sync", fmt.Sprintf("sync cycle (%s): no file changes", reason))
		}
	}

	if targetGameID == "" {
		e.pending.Store(0)
	}
	e.mu.Lock()
	e.status.LastSyncTime = start.UTC()
	e.mu.Unlock()
}

// updates status atomically under engine mutex.
func (e *Engine) setRunning(value bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Running = value
}

// stores latest sync error in status snapshot.
func (e *Engine) setLastError(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.LastError = msg
}

// records a sync error and updates status state.
func (e *Engine) noteError(event, msg string) {
	e.setLastError(msg)
	_ = e.logs.Write("error", event, msg)
}

// evaluates filename and path patterns from configured ignore rules.
func shouldIgnore(path string, patterns []string) bool {
	name := filepath.Base(path)
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		if strings.EqualFold(name, p) {
			return true
		}
		if matched, _ := filepath.Match(p, name); matched {
			return true
		}
		if strings.Contains(strings.ToLower(path), strings.ToLower(string(os.PathSeparator)+p+string(os.PathSeparator))) {
			return true
		}
	}
	return false
}
