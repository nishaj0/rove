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

type fileFingerprint struct {
	ModTime int64
	Size    int64
}

type Engine struct {
	cfg     *config.Manager
	logs    *logstore.Store
	mu      sync.RWMutex
	status  models.Status
	trigger chan string
	tracked map[string]fileFingerprint
	pending atomic.Int64
	started atomic.Bool
	running atomic.Bool
	paused  atomic.Bool
}

func New(cfg *config.Manager, logs *logstore.Store) *Engine {
	return &Engine{
		cfg:     cfg,
		logs:    logs,
		trigger: make(chan string, 1),
		tracked: make(map[string]fileFingerprint),
	}
}

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

func (e *Engine) Stop() {
	e.paused.Store(true)
	e.running.Store(false)
	e.setRunning(false)
	_ = e.logs.Write("info", "sync", "sync engine paused")
}

func (e *Engine) Resume() {
	e.paused.Store(false)
	e.running.Store(true)
	e.setRunning(true)
	e.Trigger("resume")
	_ = e.logs.Write("info", "sync", "sync engine resumed")
}

func (e *Engine) Trigger(reason string) {
	if reason == "" {
		reason = "manual"
	}
	// keep trigger queue bounded; a single queued run is enough.
	select {
	case e.trigger <- reason:
	default:
	}
}

func (e *Engine) NotifyFileEvent(_ string) {
	e.pending.Add(1)
	e.Trigger("watcher")
}

func (e *Engine) Status() models.Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copy := e.status
	copy.Pending = int(e.pending.Load())
	return copy
}

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
		case reason := <-e.trigger:
			timer.Stop()
			e.runOnce(reason)
		case <-timer.C:
			e.runOnce("interval")
		}
	}
}

func (e *Engine) runOnce(reason string) {
	if e.paused.Load() {
		return
	}
	start := time.Now()
	settings := e.cfg.Get()
	changed := 0

	// detect changes by comparing mtime+size fingerprints per file.
	for _, game := range settings.Games {
		_ = filepath.WalkDir(game.Path, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
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
				return nil
			}
			key := game.ID + "::" + path
			fp := fileFingerprint{ModTime: info.ModTime().UnixNano(), Size: info.Size()}

			e.mu.Lock()
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
		_ = e.logs.Write("info", "sync", fmt.Sprintf("sync cycle (%s): %d file(s) changed", reason, changed))
	}

	e.pending.Store(0)
	e.mu.Lock()
	e.status.LastSyncTime = start.UTC()
	e.status.LastError = ""
	e.mu.Unlock()
}

func (e *Engine) setRunning(value bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Running = value
}

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
