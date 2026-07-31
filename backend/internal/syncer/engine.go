package syncer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

type syncTrigger struct{ reason, gameID, jobID string }
type StoreFactory func(context.Context, models.Settings) (RevisionStore, error)

// Engine serializes reconciliation work, coalesces watcher bursts, and retains
// a bounded in-memory job history for the local UI.
type Engine struct {
	cfg          *config.Manager
	logs         *logstore.Store
	debug        bool
	mu           sync.RWMutex
	status       models.Status
	trigger      chan syncTrigger
	pending      atomic.Int64
	started      atomic.Bool
	running      atomic.Bool
	paused       atomic.Bool
	jobsMu       sync.RWMutex
	jobs         map[string]models.SyncJob
	jobOrder     []string
	ctx          context.Context
	storeFactory StoreFactory
}

func New(cfg *config.Manager, logs *logstore.Store, debug bool) *Engine {
	return &Engine{cfg: cfg, logs: logs, debug: debug, trigger: make(chan syncTrigger, 1), jobs: make(map[string]models.SyncJob), ctx: context.Background(), storeFactory: func(_ context.Context, settings models.Settings) (RevisionStore, error) {
		return NewFolderStore(settings.SyncRoot), nil
	}}
}

// SetStoreFactory lets application composition add direct providers without
// coupling the reconciliation package to a provider implementation.
func (e *Engine) SetStoreFactory(factory StoreFactory) {
	if factory != nil {
		e.storeFactory = factory
	}
}

func (e *Engine) Start(ctx context.Context) {
	if !e.started.CompareAndSwap(false, true) {
		e.Resume()
		return
	}
	e.running.Store(true)
	e.ctx = ctx
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
	e.Trigger(TriggerReasonResume)
	_ = e.logs.Write("info", "sync", "sync engine resumed")
}

func (e *Engine) Trigger(reason string) string { return e.enqueue("", reason) }
func (e *Engine) TriggerGame(gameID, reason string) string {
	if gameID == "" {
		return ""
	}
	return e.enqueue(gameID, reason)
}
func (e *Engine) enqueue(gameID, reason string) string {
	if reason == "" {
		reason = TriggerReasonManual
	}
	job := models.SyncJob{ID: uuid.NewString(), GameID: gameID, Reason: reason, State: "queued"}
	e.storeJob(job)
	select {
	case e.trigger <- syncTrigger{reason: reason, gameID: gameID, jobID: job.ID}:
	default:
		// A later full reconciliation makes an extra watcher job redundant.
		e.updateJob(job.ID, func(j *models.SyncJob) { j.State = "coalesced"; j.FinishedAt = time.Now().UTC() })
	}
	return job.ID
}

func (e *Engine) NotifyFileEvent(_ string) { e.pending.Add(1); e.Trigger(TriggerReasonWatcher) }
func (e *Engine) Status() models.Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	value := e.status
	value.Pending = int(e.pending.Load())
	settings := e.cfg.Get()
	value.SyncConfigured = settings.CloudProvider == models.CloudProviderGoogleDrive && settings.GoogleDrive.Connected || settings.CloudProvider != models.CloudProviderGoogleDrive && settings.SyncRoot != ""
	return value
}
func (e *Engine) Jobs(gameID string) []models.SyncJob {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()
	result := make([]models.SyncJob, 0, len(e.jobOrder))
	for i := len(e.jobOrder) - 1; i >= 0; i-- {
		job := e.jobs[e.jobOrder[i]]
		if gameID == "" || job.GameID == gameID {
			result = append(result, job)
		}
	}
	return result
}

// ResolveConflict selects the active provider before applying an explicit
// user decision, keeping the choice independent of storage transport.
func (e *Engine) ResolveConflict(game models.Game, conflictID, action string) error {
	settings := e.cfg.Get()
	store, err := e.storeFactory(e.ctx, settings)
	if err != nil {
		return err
	}
	return ResolveConflictWithStore(e.ctx, store, game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, e.logs, conflictID, action)
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
		case req := <-e.trigger:
			timer.Stop()
			e.runOnce(req)
		case <-timer.C:
			e.runOnce(syncTrigger{reason: TriggerReasonInterval, jobID: e.newScheduledJob()})
		}
	}
}
func (e *Engine) newScheduledJob() string { return e.enqueueNoQueue("", TriggerReasonInterval) }
func (e *Engine) enqueueNoQueue(gameID, reason string) string {
	job := models.SyncJob{ID: uuid.NewString(), GameID: gameID, Reason: reason, State: "queued"}
	e.storeJob(job)
	return job.ID
}

func (e *Engine) runOnce(req syncTrigger) {
	if e.paused.Load() {
		return
	}
	settings := e.cfg.Get()
	e.updateJob(req.jobID, func(job *models.SyncJob) { job.State = "running"; job.StartedAt = time.Now().UTC() })
	if settings.CloudProvider == models.CloudProviderFolder && settings.SyncRoot == "" {
		e.updateJob(req.jobID, func(job *models.SyncJob) {
			job.State = "skipped"
			job.FinishedAt = time.Now().UTC()
			job.Error = "Choose a sync folder in Settings to enable cloud sync."
		})
		return
	}
	if settings.CloudProvider == models.CloudProviderGoogleDrive && !settings.GoogleDrive.Connected {
		e.updateJob(req.jobID, func(job *models.SyncJob) {
			job.State = "skipped"
			job.FinishedAt = time.Now().UTC()
			job.Error = "Connect Google Drive in Settings to enable cloud sync."
		})
		return
	}
	store, err := e.storeFactory(e.ctx, settings)
	if err != nil {
		e.finishJob(req.jobID, "failed", 0, 0, nil, err.Error())
		return
	}
	games := settings.Games
	if req.gameID != "" {
		games = nil
		for _, game := range settings.Games {
			if game.ID == req.gameID {
				games = append(games, game)
				break
			}
		}
		if len(games) == 0 {
			e.finishJob(req.jobID, "failed", 0, 0, nil, "sync target game not found")
			return
		}
	}
	changed := 0
	var bytes int64
	conflictIDs := []string{}
	for _, game := range games {
		result, err := ReconcileWithStore(e.ctx, store, game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, e.logs)
		if err != nil {
			e.noteError("sync", fmt.Sprintf("sync %s failed: %v", game.Name, err))
			e.finishJob(req.jobID, "failed", changed, bytes, conflictIDs, err.Error())
			return
		}
		changed += result.ChangedFiles
		bytes += result.Bytes
		conflictIDs = append(conflictIDs, result.ConflictIDs...)
	}
	state := "completed"
	if len(conflictIDs) > 0 {
		state = "conflict"
	}
	e.finishJob(req.jobID, state, changed, bytes, conflictIDs, "")
	e.pending.Store(0)
	e.mu.Lock()
	e.status.LastSyncTime = time.Now().UTC()
	e.status.LastError = ""
	e.mu.Unlock()
	if e.debug {
		_ = e.logs.Write("debug", "sync", fmt.Sprintf("sync cycle %s: %d files, %d bytes", req.reason, changed, bytes))
	}
}

func (e *Engine) finishJob(id, state string, changed int, bytes int64, conflicts []string, message string) {
	e.updateJob(id, func(job *models.SyncJob) {
		job.State = state
		job.FinishedAt = time.Now().UTC()
		job.ChangedFiles = changed
		job.Bytes = bytes
		job.ConflictIDs = conflicts
		job.Error = message
	})
}
func (e *Engine) storeJob(job models.SyncJob) {
	e.jobsMu.Lock()
	defer e.jobsMu.Unlock()
	e.jobs[job.ID] = job
	e.jobOrder = append(e.jobOrder, job.ID)
	if len(e.jobOrder) > 100 {
		delete(e.jobs, e.jobOrder[0])
		e.jobOrder = e.jobOrder[1:]
	}
}
func (e *Engine) updateJob(id string, edit func(*models.SyncJob)) {
	if id == "" {
		return
	}
	e.jobsMu.Lock()
	defer e.jobsMu.Unlock()
	job, ok := e.jobs[id]
	if !ok {
		return
	}
	edit(&job)
	e.jobs[id] = job
}
func (e *Engine) setRunning(value bool) { e.mu.Lock(); defer e.mu.Unlock(); e.status.Running = value }
func (e *Engine) noteError(event, msg string) {
	e.mu.Lock()
	e.status.LastError = msg
	e.mu.Unlock()
	_ = e.logs.Write("error", event, msg)
}
