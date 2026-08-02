package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

type syncTrigger struct {
	reason, gameID, jobID string
}

type StoreFactory func(context.Context, models.Settings) (RevisionStore, error)

type dirtyGame struct {
	lastEvent   time.Time
	lastCapture time.Time
}

// Engine is the unified local-first protection coordinator. It records dirty
// paths cheaply, serializes capture/provider work, and persists every job stage
// before the next side effect.
type Engine struct {
	cfg          *config.Manager
	logs         *logstore.Store
	debug        bool
	localStore   *FolderStore
	storeFactory StoreFactory

	mu      sync.RWMutex
	status  models.Status
	health  models.ProviderHealth
	running map[string]struct{}
	dirty   map[string]dirtyGame
	exits   map[string]time.Time

	trigger chan syncTrigger
	started atomic.Bool
	paused  atomic.Bool
	ctx     context.Context

	jobsMu   sync.RWMutex
	jobs     map[string]models.SyncJob
	jobOrder []string

	statusUpdates chan struct{}
}

func New(cfg *config.Manager, logs *logstore.Store, debug bool) *Engine {
	vaultRoot := filepath.Join(filepath.Dir(cfg.Path()), "vault")
	return &Engine{
		cfg:           cfg,
		logs:          logs,
		debug:         debug,
		localStore:    NewFolderStore(vaultRoot),
		trigger:       make(chan syncTrigger, 64),
		jobs:          make(map[string]models.SyncJob),
		running:       make(map[string]struct{}),
		dirty:         make(map[string]dirtyGame),
		exits:         make(map[string]time.Time),
		statusUpdates: make(chan struct{}, 1),
		ctx:           context.Background(),
		storeFactory: func(_ context.Context, settings models.Settings) (RevisionStore, error) {
			return NewFolderStore(settings.SyncRoot), nil
		},
	}
}

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
	e.ctx = ctx
	settings := e.cfg.Get()
	control, err := e.logs.ServiceControl("global", settings.ServiceControl)
	if err == nil {
		if control.PauseUntilLogin {
			control.PauseScope = models.PauseNone
			control.PauseUntilLogin = false
			control.Reason = ""
		}
		settings.ServiceControl = normalizeControl(control)
		_ = e.logs.SaveServiceControl("global", settings.ServiceControl)
	}
	e.paused.Store(!settings.ServiceControl.ProtectionEnabled || settings.ServiceControl.PauseScope == models.PauseAll)
	e.setRunning(!e.paused.Load())
	if jobs, err := e.logs.ProtectionJobs("", 100); err == nil {
		for i := len(jobs) - 1; i >= 0; i-- {
			e.storeJobMemory(jobs[i])
		}
	}
	_ = e.logs.PruneActivity()
	e.refreshHealth()
	go e.loop(ctx)
}

// Stop is a user pause, not daemon shutdown. The state is durable.
func (e *Engine) Stop() {
	control := e.GlobalControl()
	control.ProtectionEnabled = true
	control.PauseScope = models.PauseAll
	control.PauseUntil = time.Time{}
	control.PauseUntilLogin = false
	control.Reason = "Paused until resumed"
	_ = e.SetServiceControl("", control)
}

func (e *Engine) Resume() {
	control := e.GlobalControl()
	control.ProtectionEnabled = true
	control.PauseScope = models.PauseNone
	control.PauseUntil = time.Time{}
	control.PauseUntilLogin = false
	control.Reason = ""
	_ = e.SetServiceControl("", control)
	e.retryNow(false)
}

func (e *Engine) GlobalControl() models.ServiceControlState {
	settings := e.cfg.Get()
	value, err := e.logs.ServiceControl("global", settings.ServiceControl)
	if err != nil {
		return normalizeControl(settings.ServiceControl)
	}
	return normalizeControl(value)
}

func (e *Engine) GameControl(gameID string) models.ServiceControlState {
	fallback := models.ServiceControlState{ProtectionEnabled: true, PauseScope: models.PauseNone}
	value, err := e.logs.ServiceControl("game:"+gameID, fallback)
	if err != nil {
		return fallback
	}
	if value.PauseUntilLogin {
		value.PauseScope = models.PauseNone
		value.PauseUntilLogin = false
		value.Reason = ""
		_ = e.logs.SaveServiceControl("game:"+gameID, value)
	}
	return normalizeControl(value)
}

func normalizeControl(value models.ServiceControlState) models.ServiceControlState {
	if value.PauseScope == "" {
		value.PauseScope = models.PauseNone
	}
	if !value.PauseUntil.IsZero() && time.Now().After(value.PauseUntil) {
		value.PauseScope = models.PauseNone
		value.PauseUntil = time.Time{}
		value.Reason = ""
	}
	return value
}

func (e *Engine) SetServiceControl(gameID string, value models.ServiceControlState) error {
	value = normalizeControl(value)
	switch value.PauseScope {
	case models.PauseNone, models.PauseUploads, models.PauseAll:
	default:
		return errors.New("invalid pause scope")
	}
	key := "global"
	if gameID != "" {
		if _, ok := e.cfg.FindGame(gameID); !ok {
			return errors.New("game not found")
		}
		key = "game:" + gameID
	}
	if err := e.logs.SaveServiceControl(key, value); err != nil {
		return err
	}
	if gameID == "" {
		e.paused.Store(!value.ProtectionEnabled || value.PauseScope == models.PauseAll)
		e.setRunning(!e.paused.Load())
	}
	title := "Cloud Sync resumed"
	detail := "Cloud Sync workers are active."
	if value.PauseScope != models.PauseNone || !value.ProtectionEnabled {
		title, detail = "Cloud Sync paused", value.Reason
		if detail == "" {
			detail = "Cloud Sync work is paused until you resume it."
		}
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "sync", Severity: "info", GameID: gameID, Title: title, Detail: detail})
	e.signalStatus()
	return nil
}

func (e *Engine) Trigger(reason string) string {
	settings := e.cfg.Get()
	var first string
	for _, game := range settings.Games {
		id := e.TriggerGame(game.ID, reason)
		if first == "" {
			first = id
		}
	}
	return first
}

func (e *Engine) TriggerGame(gameID, reason string) string {
	if gameID == "" {
		return ""
	}
	return e.enqueue(gameID, reason, 50, "")
}

func (e *Engine) CreateRecoveryPoint(gameID, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Recovery Point " + time.Now().Format("2006-01-02 15:04")
	}
	return e.enqueue(gameID, "recovery-point:"+name, 100, "")
}

// WaitForLocalProtection waits only for the durable local publication stage.
// Provider replication may remain queued or offline after this returns.
func (e *Engine) WaitForLocalProtection(ctx context.Context, jobID string) (models.Revision, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		job := e.job(jobID)
		if job.RevisionID != "" {
			return e.logs.Revision(job.RevisionID)
		}
		if job.State == "failed" {
			if job.Error != "" {
				return models.Revision{}, errors.New(job.Error)
			}
			return models.Revision{}, errors.New("local sync failed")
		}
		select {
		case <-ctx.Done():
			return models.Revision{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (e *Engine) enqueue(gameID, reason string, priority int, revisionID string) string {
	if reason == "" {
		reason = TriggerReasonManual
	}
	now := time.Now().UTC()
	job := models.SyncJob{
		ID:         uuid.NewString(),
		GameID:     gameID,
		Reason:     reason,
		State:      "queued",
		Stage:      "waiting",
		Priority:   priority,
		RevisionID: revisionID,
		UpdatedAt:  now,
	}
	e.storeJob(job)
	select {
	case e.trigger <- syncTrigger{reason: reason, gameID: gameID, jobID: job.ID}:
	default:
		e.updateJob(job.ID, func(value *models.SyncJob) {
			value.State, value.Stage = "waiting", "waiting"
			value.Error = "Cloud Sync is queued behind other work."
		})
	}
	e.signalStatus()
	return job.ID
}

// NotifyFileEvent maps a path to the longest save root and marks only that
// relative path dirty. No hashing or provider work happens on this callback.
func (e *Engine) NotifyFileEvent(path string) {
	clean := filepath.Clean(path)
	var selected *models.Game
	selectedRootLength := -1
	settings := e.cfg.Get()
	for i := range settings.Games {
		root := filepath.Clean(settings.Games[i].Path)
		rel, err := filepath.Rel(root, clean)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(root) > selectedRootLength {
			selected, selectedRootLength = &settings.Games[i], len(root)
		}
	}
	if selected == nil {
		return
	}
	rel, err := filepath.Rel(selected.Path, clean)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		_ = e.logs.MarkAllFilesDirty(selected.ID)
	} else {
		_ = e.logs.MarkFileDirty(selected.ID, rel)
	}
	now := time.Now().UTC()
	e.mu.Lock()
	state := e.dirty[selected.ID]
	state.lastEvent = now
	e.dirty[selected.ID] = state
	delete(e.exits, selected.ID)
	e.mu.Unlock()
	e.signalStatus()
}

// UpdateRunning records lifecycle transitions. A stopped game receives a final
// checkpoint only after five seconds without further save activity.
func (e *Engine) UpdateRunning(ids []string) {
	next := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		next[id] = struct{}{}
	}
	now := time.Now().UTC()
	e.mu.Lock()
	for id := range e.running {
		if _, stillRunning := next[id]; !stillRunning {
			e.exits[id] = now.Add(5 * time.Second)
		}
	}
	for id := range next {
		if _, wasRunning := e.running[id]; !wasRunning {
			state := e.dirty[id]
			state.lastEvent = now
			e.dirty[id] = state
		}
	}
	e.running = next
	e.mu.Unlock()
	e.signalStatus()
}

func (e *Engine) StatusUpdates() <-chan struct{} { return e.statusUpdates }

func (e *Engine) Status() models.Status {
	e.mu.RLock()
	value := e.status
	value.Provider = e.health
	e.mu.RUnlock()
	value.ServiceControl = e.GlobalControl()
	value.Running = value.ServiceControl.ProtectionEnabled && value.ServiceControl.PauseScope != models.PauseAll
	settings := e.cfg.Get()
	value.ProtectionTarget = settings.ProtectionTarget
	value.SyncConfigured = settings.ProtectionTarget == models.ProtectionTargetLocal ||
		settings.ProtectionTarget == models.ProtectionTargetDrive && settings.GoogleDrive.Connected ||
		settings.ProtectionTarget == models.ProtectionTargetFolder && strings.TrimSpace(settings.SyncRoot) != ""
	count, bytes, _ := e.logs.PendingSummary("")
	value.Pending, value.PendingBytes = count, bytes
	value.Provider.PendingCount, value.Provider.PendingBytes = count, bytes
	conflicts := 0
	for _, game := range settings.Games {
		items, _ := e.logs.SyncConflicts(game.ID)
		conflicts += len(items)
	}
	value.ConflictCount = conflicts
	if jobs := e.Jobs(""); len(jobs) > 0 {
		for _, job := range jobs {
			if job.State == "running" || job.State == "queued" || job.State == "waiting" {
				value.ActiveJobID = job.ID
				break
			}
		}
	}
	return value
}

func (e *Engine) ProtectionStatus(gameID string) models.GameProtectionStatus {
	result := models.GameProtectionStatus{GameID: gameID, ServiceControl: e.GameControl(gameID)}
	if game, ok := e.cfg.FindGame(gameID); ok {
		result.Policy = game.ProtectionPolicy
	}
	result.DirtyFiles, _ = e.logs.DirtyFileCount(gameID)
	result.PendingCount, result.PendingBytes, _ = e.logs.PendingSummary(gameID)
	if revisions, _ := e.logs.Revisions(gameID, 100); len(revisions) > 0 {
		result.LastLocalProtection = revisions[0].CreatedAt
		for _, revision := range revisions {
			if revision.Uploaded {
				result.LastCloudUpload = revision.CreatedAt
				break
			}
		}
	}
	result.ConflictCount = lenMust(e.logs.SyncConflicts(gameID))
	for _, job := range e.Jobs(gameID) {
		if job.State == "running" || job.State == "queued" || job.State == "waiting" || job.State == "waiting_for_network" || job.State == "paused" {
			copy := job
			result.ActiveJob = &copy
			break
		}
	}
	return result
}

func lenMust[T any](value []T, _ error) int { return len(value) }

func (e *Engine) Jobs(gameID string) []models.SyncJob {
	if jobs, err := e.logs.ProtectionJobs(gameID, 100); err == nil {
		return jobs
	}
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

func (e *Engine) ResolveConflict(game models.Game, conflictID, action string) error {
	settings := e.cfg.Get()
	store, err := e.storeFactory(e.ctx, settings)
	if err != nil {
		return err
	}
	if err := ResolveConflictWithStore(e.ctx, store, game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, e.logs, conflictID, action); err != nil {
		return err
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "conflict", Severity: "info", GameID: game.ID, Title: "Conflict resolved", Detail: "The selected conflict copy was applied."})
	return nil
}

func (e *Engine) Revisions(gameID string) ([]models.Revision, error) {
	return e.logs.Revisions(gameID, 100)
}

func (e *Engine) RecoveryPoints(gameID string) ([]models.RecoveryPoint, error) {
	return e.logs.RecoveryPoints(gameID)
}

func (e *Engine) StorageUsage(gameID string) (models.StorageUsage, error) {
	revisions, err := e.logs.Revisions(gameID, 1000)
	if err != nil {
		return models.StorageUsage{}, err
	}
	points, err := e.logs.RecoveryPoints(gameID)
	if err != nil {
		return models.StorageUsage{}, err
	}
	objects := map[string]int64{}
	usage := models.StorageUsage{GameID: gameID, RecoveryPoints: len(points)}
	for _, revision := range revisions {
		if revision.Kind == "automatic" || revision.Kind == "final" {
			usage.AutomaticCount++
		}
		var value Manifest
		if json.Unmarshal([]byte(revision.ManifestJSON), &value) != nil {
			continue
		}
		for _, file := range value.Files {
			objects[file.Hash] = file.Size
		}
	}
	for _, size := range objects {
		usage.UniqueBytes += size
	}
	_, usage.PendingBytes, _ = e.logs.PendingSummary(gameID)
	// Physical quarantine cleanup is deliberately inactive; report no
	// reclaimable bytes rather than promising unsafe deletion.
	usage.ReclaimableBytes = 0
	return usage, nil
}

func (e *Engine) CloudProfiles(ctx context.Context) ([]models.GameProfile, error) {
	settings := e.cfg.Get()
	if settings.ProtectionTarget == models.ProtectionTargetLocal {
		return e.localStore.ListGameProfiles(ctx)
	}
	store, err := e.storeFactory(ctx, settings)
	if err != nil {
		return nil, err
	}
	profiles, ok := store.(ProfileStore)
	if !ok {
		return nil, errors.New("provider does not support recovery library metadata")
	}
	return profiles.ListGameProfiles(ctx)
}

// RestoreCloudProfile restores a provider profile into a new or empty
// destination. It materializes and verifies the complete revision in a sibling
// staging directory before the destination becomes visible.
func (e *Engine) RestoreCloudProfile(ctx context.Context, syncID, destination string) error {
	syncID, destination = strings.TrimSpace(syncID), filepath.Clean(strings.TrimSpace(destination))
	if syncID == "" || destination == "." || destination == "" {
		return errors.New("sync id and destination are required")
	}
	if info, err := os.Stat(destination); err == nil {
		if !info.IsDir() {
			return errors.New("recovery destination must be a folder")
		}
		entries, readErr := os.ReadDir(destination)
		if readErr != nil {
			return readErr
		}
		if len(entries) != 0 {
			return errors.New("recovery destination is not empty; add it as a game and review conflicts before restoring")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	settings := e.cfg.Get()
	store, err := e.storeFactory(ctx, settings)
	if err != nil {
		return err
	}
	headID, err := store.ReadHead(ctx, syncID)
	if err != nil {
		return err
	}
	if headID == "" {
		return errors.New("the selected cloud game has no complete revision")
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(destination)+"-recovery-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if _, err := RestoreRevision(ctx, store, staging, syncID, headID, settings.DeviceID, settings.IgnorePatterns); err != nil {
		return err
	}

	emptyPrevious := ""
	if _, err := os.Stat(destination); err == nil {
		emptyPrevious = filepath.Join(parent, "."+filepath.Base(destination)+"-empty-"+uuid.NewString())
		if err := os.Rename(destination, emptyPrevious); err != nil {
			return err
		}
	}
	if err := os.Rename(staging, destination); err != nil {
		if emptyPrevious != "" {
			_ = os.Rename(emptyPrevious, destination)
		}
		return err
	}
	if emptyPrevious != "" {
		_ = os.RemoveAll(emptyPrevious)
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{
		Category: "recovery",
		Severity: "info",
		Title:    "Cloud recovery completed",
		Detail:   "A complete, verified cloud revision was restored to the selected empty folder.",
	})
	return nil
}

func (e *Engine) DeleteRecoveryPoint(id string) error {
	point, err := e.logs.RecoveryPoint(id)
	if err != nil {
		return err
	}
	game, ok := e.cfg.FindGame(point.GameID)
	if !ok {
		return errors.New("game not found")
	}
	_ = e.localStore.DeleteRecoveryPoint(e.ctx, game.SyncID, point.ID)
	settings := e.cfg.Get()
	if settings.ProtectionTarget != models.ProtectionTargetLocal {
		if store, storeErr := e.storeFactory(e.ctx, settings); storeErr == nil {
			if points, supported := store.(RecoveryPointStore); supported {
				_ = points.DeleteRecoveryPoint(e.ctx, game.SyncID, point.ID)
			}
		}
	}
	if err := e.logs.DeleteRecoveryPoint(id); err != nil {
		return err
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "recovery", Severity: "info", Title: "Recovery Point deleted", Detail: "The pin was removed. Retention cleanup may reclaim its unreferenced data."})
	return nil
}

// PrepareLaunch performs a bounded clean remote-head restore. Provider failure
// is returned so the UI can offer explicit offline play.
func (e *Engine) PrepareLaunch(ctx context.Context, game models.Game) error {
	settings := e.cfg.Get()
	if !settings.RemoteCheckBeforeLaunch || settings.ProtectionTarget == models.ProtectionTargetLocal {
		return nil
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	store, err := e.storeFactory(checkCtx, settings)
	if err != nil {
		return err
	}
	remoteHead, err := store.ReadHead(checkCtx, game.SyncID)
	if err != nil {
		return err
	}
	if remoteHead == "" {
		return nil
	}
	localHead, err := e.localStore.ReadHead(checkCtx, game.SyncID)
	if err == nil && localHead == remoteHead {
		return nil
	}
	dirty, _ := e.logs.DirtyFileCount(game.ID)
	pending, _, _ := e.logs.PendingSummary(game.ID)
	if dirty > 0 || pending > 0 {
		return errors.New("local saves or uploads are pending; review sync before launching")
	}
	if _, _, err := e.captureLocal(checkCtx, game, "pre_restore", "Before cloud launch restore", true); err != nil {
		return err
	}
	if _, err := ReplicateRevision(checkCtx, store, e.localStore, game.SyncID, remoteHead); err != nil {
		return err
	}
	if _, err := RestoreRevision(checkCtx, store, game.Path, game.SyncID, remoteHead, settings.DeviceID, settings.IgnorePatterns); err != nil {
		return err
	}
	remoteManifest, err := store.ReadManifest(checkCtx, game.SyncID, remoteHead)
	if err != nil {
		return err
	}
	if err := saveLocalManifestTo(e.logs, game.ID, remoteManifest); err != nil {
		return err
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "recovery", Severity: "info", GameID: game.ID, Title: "Newer cloud save restored", Detail: "Rove applied the latest clean provider revision before launch."})
	return nil
}

func (e *Engine) RestoreLocalRevision(ctx context.Context, game models.Game, revisionID string) error {
	if _, _, err := e.captureLocal(ctx, game, "pre_restore", "Before restore", true); err != nil {
		return err
	}
	settings := e.cfg.Get()
	if _, err := RestoreRevision(ctx, e.localStore, game.Path, game.SyncID, revisionID, settings.DeviceID, settings.IgnorePatterns); err != nil {
		return err
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "recovery", Severity: "info", GameID: game.ID, Title: "Revision restored", Detail: "The selected revision was materialized exactly."})
	e.NotifyFileEvent(game.Path)
	return nil
}

func (e *Engine) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-e.trigger:
			e.runOnce(request)
		case now := <-ticker.C:
			e.scheduleDue(now.UTC())
			e.resumePending(now.UTC())
		}
	}
}

func (e *Engine) scheduleDue(now time.Time) {
	global := normalizeControl(e.GlobalControl())
	if global != e.GlobalControl() {
		_ = e.SetServiceControl("", global)
	}
	if !global.ProtectionEnabled || global.PauseScope == models.PauseAll {
		return
	}
	settings := e.cfg.Get()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, game := range settings.Games {
		control := normalizeControl(e.GameControl(game.ID))
		if !control.ProtectionEnabled || control.PauseScope == models.PauseAll || !game.ProtectionPolicy.Enabled {
			continue
		}
		state, isDirty := e.dirty[game.ID]
		if due, exiting := e.exits[game.ID]; exiting {
			if game.ProtectionPolicy.Mode == models.ProtectionManual || !game.ProtectionPolicy.ExitProtection {
				delete(e.exits, game.ID)
				continue
			}
			if now.Before(due) || now.Sub(state.lastEvent) < 5*time.Second {
				continue
			}
			delete(e.exits, game.ID)
			delete(e.dirty, game.ID)
			go e.enqueue(game.ID, "final", 80, "")
			continue
		}
		if !isDirty {
			continue
		}
		_, isRunning := e.running[game.ID]
		if !isRunning {
			continue
		}
		var due bool
		switch game.ProtectionPolicy.Mode {
		case models.ProtectionContinuous:
			due = now.Sub(state.lastEvent) >= 5*time.Second && now.Sub(state.lastCapture) >= 2*time.Minute
		case models.ProtectionBalanced:
			interval := time.Duration(game.ProtectionPolicy.CheckpointIntervalMinutes) * time.Minute
			due = now.Sub(state.lastCapture) >= interval
		}
		if due {
			state.lastCapture = now
			e.dirty[game.ID] = state
			go e.enqueue(game.ID, "checkpoint", 40, "")
		}
	}
}

func (e *Engine) resumePending(now time.Time) {
	pending, err := e.logs.PendingProtectionJobs(now)
	if err != nil {
		return
	}
	for _, job := range pending {
		if job.State == "paused" && e.uploadsPaused(job.GameID) {
			continue
		}
		if job.State == "failed" && (job.ErrorCode == "authentication_required" || job.ErrorCode == "quota_full") {
			continue
		}
		select {
		case e.trigger <- syncTrigger{reason: job.Reason, gameID: job.GameID, jobID: job.ID}:
			return
		default:
			return
		}
	}
}

func (e *Engine) runOnce(request syncTrigger) {
	game, ok := e.cfg.FindGame(request.gameID)
	if !ok {
		e.finishJob(request.jobID, "failed", "failed", 0, 0, "game_not_found", "Cloud Sync game not found.")
		return
	}
	global, perGame := e.GlobalControl(), e.GameControl(game.ID)
	if !global.ProtectionEnabled || !perGame.ProtectionEnabled || global.PauseScope == models.PauseAll || perGame.PauseScope == models.PauseAll {
		e.updateJob(request.jobID, func(job *models.SyncJob) {
			job.State, job.Stage, job.Error = "paused", "paused", "Cloud Sync is paused."
		})
		return
	}
	job := e.job(request.jobID)
	revisionID := job.RevisionID
	changed, bytes := job.ChangedFiles, job.Bytes
	if revisionID == "" {
		kind, name, pinned := reasonDetails(request.reason)
		e.updateJob(request.jobID, func(value *models.SyncJob) {
			value.State, value.Stage, value.Progress = "running", "stabilizing", 5
			value.StartedAt = time.Now().UTC()
		})
		revision, result, err := e.captureLocal(e.ctx, game, kind, name, pinned)
		if err != nil {
			e.retryJob(request.jobID, classifyLocalError(err), err)
			return
		}
		revisionID, changed, bytes = revision.ID, result.ChangedFiles, result.Bytes
		e.updateJob(request.jobID, func(value *models.SyncJob) {
			value.RevisionID, value.ChangedFiles, value.Bytes = revisionID, changed, bytes
			value.Stage, value.Progress = "locally_protected", 55
		})
		e.mu.Lock()
		e.status.LastLocalProtection = revision.CreatedAt
		e.mu.Unlock()
	}
	settings := e.cfg.Get()
	if settings.ProtectionTarget == models.ProtectionTargetLocal {
		e.finishJob(request.jobID, "completed", "completed", changed, bytes, "", "")
		e.noteProtectionComplete(game, revisionID, false)
		return
	}
	if e.uploadsPaused(game.ID) {
		e.updateJob(request.jobID, func(value *models.SyncJob) {
			value.State, value.Stage, value.Error = "paused", "paused", "Cloud uploads are paused; the local revision is safe."
		})
		return
	}
	e.updateJob(request.jobID, func(value *models.SyncJob) {
		value.State, value.Stage, value.Progress = "running", "uploading", 65
	})
	e.noteProviderAttempt()
	store, err := e.storeFactory(e.ctx, settings)
	if err == nil {
		_, err = ReplicateRevisionGuarded(e.ctx, e.localStore, store, game.SyncID, revisionID, game.ID, e.logs)
	}
	if err == nil {
		if profiles, ok := store.(ProfileStore); ok {
			revision, revisionErr := e.logs.Revision(revisionID)
			if revisionErr == nil {
				err = profiles.WriteGameProfile(e.ctx, gameProfile(game, revision))
			}
		}
	}
	if err == nil {
		if points, supported := store.(RecoveryPointStore); supported {
			localPoints, _ := e.logs.RecoveryPoints(game.ID)
			for _, point := range localPoints {
				if point.RevisionID == revisionID {
					err = points.WriteRecoveryPoint(e.ctx, game.SyncID, point)
					break
				}
			}
		}
	}
	if err != nil {
		if errors.Is(err, ErrConcurrentRevision) {
			e.finishJob(request.jobID, "conflict", "conflict", changed, bytes, "concurrent_revision", "Another device changed this game. Review the preserved versions before applying either side.")
			_ = e.logs.WriteActivity(models.ActivityEvent{
				Category:         "conflict",
				Severity:         "warning",
				GameID:           game.ID,
				Title:            "Concurrent cloud save needs review",
				Detail:           "Both device revisions were preserved; the shared head was not overwritten.",
				DeduplicationKey: "concurrent_revision:" + game.ID,
			})
			return
		}
		code, state, action, retryable := classifyProviderError(err)
		e.noteProviderFailure(code, state, action, err)
		if retryable {
			e.retryJob(request.jobID, code, err, providerRetryAfter(err))
			e.noteProviderRetryAt(e.job(request.jobID).NextRetryAt)
		} else {
			stage := code
			if code == "quota_full" {
				stage = "quota_blocked"
			}
			e.finishJob(request.jobID, "failed", stage, changed, bytes, code, userProviderMessage(code))
		}
		return
	}
	_ = e.logs.MarkRevisionUploaded(revisionID)
	e.noteProviderSuccess()
	e.finishJob(request.jobID, "completed", "completed", changed, bytes, "", "")
	e.noteProtectionComplete(game, revisionID, true)
}

func (e *Engine) captureLocal(ctx context.Context, game models.Game, kind, name string, pinned bool) (models.Revision, Result, error) {
	settings := e.cfg.Get()
	e.mu.RLock()
	_, playing := e.running[game.ID]
	e.mu.RUnlock()
	if playing && kind != "final" {
		ctx = WithCaptureLimit(ctx, 4*1024*1024)
	}
	result, err := ReconcileWithStore(ctx, e.localStore, game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, e.logs)
	if err != nil {
		return models.Revision{}, Result{}, err
	}
	headID, err := e.localStore.ReadHead(ctx, game.SyncID)
	if err != nil || headID == "" {
		return models.Revision{}, Result{}, errors.New("local revision publication did not produce a valid head")
	}
	manifest, err := e.localStore.ReadManifest(ctx, game.SyncID, headID)
	if err != nil {
		return models.Revision{}, Result{}, err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return models.Revision{}, Result{}, err
	}
	var total int64
	for _, file := range manifest.Files {
		total += file.Size
	}
	revision := models.Revision{
		ID:           manifest.ID,
		GameID:       game.ID,
		SyncID:       game.SyncID,
		Kind:         kind,
		Name:         name,
		CreatedAt:    manifest.CreatedAt,
		FileCount:    len(manifest.Files),
		TotalBytes:   total,
		ManifestJSON: string(payload),
		Pinned:       pinned,
	}
	if manifest.ParentID != "" {
		revision.ParentIDs = []string{manifest.ParentID}
	}
	if err := e.logs.SaveRevision(revision); err != nil {
		return models.Revision{}, Result{}, err
	}
	_ = e.localStore.WriteGameProfile(ctx, gameProfile(game, revision))
	if pinned {
		point := models.RecoveryPoint{ID: uuid.NewString(), GameID: game.ID, RevisionID: revision.ID, Name: name, CreatedAt: revision.CreatedAt, FileCount: revision.FileCount, TotalBytes: revision.TotalBytes}
		if err := e.logs.CreateRecoveryPoint(point); err != nil {
			return models.Revision{}, Result{}, err
		}
		_ = e.localStore.WriteRecoveryPoint(ctx, game.SyncID, point)
		_ = e.logs.WriteActivity(models.ActivityEvent{Category: "recovery", Severity: "info", GameID: game.ID, Title: "Recovery Point created", Detail: name})
	}
	if kind == "automatic" || kind == "final" {
		_, _ = e.logs.PruneAutomaticRevisions(game.ID, settings.AutoRevisionRetention)
	}
	return revision, result, nil
}

func gameProfile(game models.Game, revision models.Revision) models.GameProfile {
	return models.GameProfile{
		SyncID:          game.SyncID,
		Name:            game.Name,
		ExecutableName:  filepath.Base(game.ExecutablePath),
		PathHint:        portablePathHint(game.Path),
		LatestRevision:  revision.ID,
		LatestProtected: revision.CreatedAt,
		FileCount:       revision.FileCount,
		TotalBytes:      revision.TotalBytes,
	}
}

func portablePathHint(path string) string {
	clean := filepath.Clean(path)
	candidates := []struct {
		root  string
		token string
	}{
		{os.Getenv("APPDATA"), "%APPDATA%"},
		{os.Getenv("LOCALAPPDATA"), "%LOCALAPPDATA%"},
		{filepath.Join(os.Getenv("USERPROFILE"), "Documents"), "%USERPROFILE%\\Documents"},
	}
	lower := strings.ToLower(clean)
	for _, candidate := range candidates {
		root := filepath.Clean(strings.TrimSpace(candidate.root))
		if root == "." || root == "" {
			continue
		}
		rootLower := strings.ToLower(root)
		if lower == rootLower {
			return candidate.token
		}
		if strings.HasPrefix(lower, rootLower+string(filepath.Separator)) {
			return candidate.token + clean[len(root):]
		}
	}
	return ""
}

func reasonDetails(reason string) (kind, name string, pinned bool) {
	switch {
	case strings.HasPrefix(reason, "recovery-point:"):
		return "recovery_point", strings.TrimPrefix(reason, "recovery-point:"), true
	case reason == "final":
		return "final", "", false
	case reason == "pre_restore":
		return "pre_restore", "Before restore", true
	case reason == "checkpoint":
		return "automatic", "", false
	default:
		return "manual", "", false
	}
}

func (e *Engine) uploadsPaused(gameID string) bool {
	global, game := normalizeControl(e.GlobalControl()), normalizeControl(e.GameControl(gameID))
	return !e.cfg.Get().AutomaticUpload || global.PauseScope == models.PauseUploads || game.PauseScope == models.PauseUploads
}

func (e *Engine) retryNow(force bool) {
	now := time.Now().UTC()
	jobs, _ := e.logs.ProtectionJobs("", 500)
	for _, job := range jobs {
		if job.State != "waiting_for_network" && job.State != "failed" && job.State != "paused" && job.State != "waiting" {
			continue
		}
		if !force && job.ErrorCode == "rate_limited" && job.NextRetryAt.After(now) {
			continue
		}
		job.NextRetryAt = now
		if job.ErrorCode == "authentication_required" || job.ErrorCode == "quota_full" {
			job.ErrorCode = ""
		}
		job.State, job.Stage, job.UpdatedAt = "waiting", "waiting", now
		_ = e.logs.SaveProtectionJob(job)
	}
	e.signalStatus()
}

func (e *Engine) RetryNow() { e.retryNow(false) }

// RetryConnectivity retries only failures that can plausibly be resolved by a
// Windows network-change event. It deliberately preserves provider-enforced
// rate-limit deadlines.
func (e *Engine) RetryConnectivity() {
	now := time.Now().UTC()
	jobs, _ := e.logs.ProtectionJobs("", 500)
	for _, job := range jobs {
		if job.ErrorCode != "offline" && job.ErrorCode != "provider_unavailable" {
			continue
		}
		job.NextRetryAt = now
		job.State, job.Stage, job.UpdatedAt = "waiting", "waiting", now
		_ = e.logs.SaveProtectionJob(job)
	}
	e.signalStatus()
}

// ConfigurationChanged refreshes provider health immediately after settings
// select a different protection target. Provider work remains asynchronous;
// this only prevents UI/tray state from reporting the previous destination.
func (e *Engine) ConfigurationChanged() {
	e.refreshHealth()
	e.retryNow(true)
	e.signalStatus()
}

func (e *Engine) retryJob(id, code string, err error, retryAfter ...time.Duration) {
	e.updateJob(id, func(job *models.SyncJob) {
		job.RetryCount++
		job.ErrorCode = code
		job.Error = userProviderMessage(code)
		job.State, job.Stage = "waiting_for_network", "waiting_for_network"
		delay := retryDelay(job.RetryCount, job.ID)
		if len(retryAfter) > 0 && retryAfter[0] > delay {
			delay = retryAfter[0]
		}
		job.NextRetryAt = time.Now().UTC().Add(delay)
	})
	_ = e.logs.WriteActivity(models.ActivityEvent{
		Category: "provider", Severity: "warning", GameID: e.job(id).GameID,
		Title: "Cloud upload pending", Detail: userProviderMessage(code),
		ActionTarget: "settings/provider", DeduplicationKey: "provider:" + code,
	})
	if e.debug {
		_ = e.logs.Write("warning", "protection", err.Error())
	}
}

func retryDelay(attempt int, seed string) time.Duration {
	steps := []time.Duration{15 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute, 5 * time.Minute, 15 * time.Minute}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(steps) {
		index = len(steps) - 1
	}
	var jitter time.Duration
	for _, value := range []byte(seed) {
		jitter += time.Duration(value)
	}
	return steps[index] + jitter%5*time.Second
}

func classifyLocalError(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "permission"), strings.Contains(message, "access is denied"):
		return "permission_denied"
	case strings.Contains(message, "space"), strings.Contains(message, "disk full"):
		return "local_disk_full"
	case strings.Contains(message, "still changing"), strings.Contains(message, "changed during"):
		return "file_unstable"
	default:
		return "capture_failed"
	}
}

func classifyProviderError(err error) (code string, state models.ProviderState, action string, retryable bool) {
	var providerErr *ProviderOperationError
	if errors.As(err, &providerErr) {
		switch providerErr.Code {
		case "authentication_required":
			return providerErr.Code, models.ProviderAuthRequired, "Reconnect provider", providerErr.Retryable
		case "quota_full":
			return providerErr.Code, models.ProviderQuotaFull, "Manage provider storage", providerErr.Retryable
		case "rate_limited":
			return providerErr.Code, models.ProviderRateLimited, "Retry later", providerErr.Retryable
		case "permission_denied":
			return providerErr.Code, models.ProviderRemoteProblem, "Review provider access", providerErr.Retryable
		case "offline":
			return providerErr.Code, models.ProviderOffline, "Retry connection", providerErr.Retryable
		default:
			return "provider_unavailable", models.ProviderUnavailable, "Retry connection", providerErr.Retryable
		}
	}
	message := strings.ToLower(err.Error())
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr), strings.Contains(message, "network"), strings.Contains(message, "connection"), strings.Contains(message, "offline"):
		return "offline", models.ProviderOffline, "Retry connection", true
	case strings.Contains(message, "401"), strings.Contains(message, "unauthorized"), strings.Contains(message, "invalid_grant"), strings.Contains(message, "authentication"):
		return "authentication_required", models.ProviderAuthRequired, "Reconnect provider", false
	case strings.Contains(message, "quota"), strings.Contains(message, "storagequota"):
		return "quota_full", models.ProviderQuotaFull, "Manage provider storage", false
	case strings.Contains(message, "429"), strings.Contains(message, "rate limit"):
		return "rate_limited", models.ProviderRateLimited, "Retry later", true
	case strings.Contains(message, "corrupt"), strings.Contains(message, "invalid"), strings.Contains(message, "incomplete"), strings.Contains(message, "hash"):
		return "remote_data_problem", models.ProviderRemoteProblem, "Review provider data", false
	default:
		return "provider_unavailable", models.ProviderUnavailable, "Retry connection", true
	}
}

func providerRetryAfter(err error) time.Duration {
	var providerErr *ProviderOperationError
	if errors.As(err, &providerErr) && providerErr.RetryAfter > 0 {
		return providerErr.RetryAfter
	}
	return 0
}

func userProviderMessage(code string) string {
	switch code {
	case "offline":
		return "This device is offline. The local revision is safe and will upload automatically after reconnection."
	case "authentication_required":
		return "Provider sign-in is required. Local sync will continue."
	case "quota_full":
		return "Provider storage is full. Local sync will continue until space is available."
	case "rate_limited":
		return "The provider asked Rove to slow down. Upload will retry automatically."
	case "remote_data_problem":
		return "Remote revision data is incomplete or invalid. Nothing was applied."
	case "file_unstable":
		return "A save file is still changing. Cloud Sync will retry after it settles."
	case "local_disk_full":
		return "The local disk does not have enough free space for a safe revision."
	case "permission_denied":
		return "Rove cannot read or write one of the required save paths."
	default:
		return "Cloud upload is temporarily unavailable. The local revision is safe and retry is scheduled."
	}
}

func (e *Engine) noteProviderAttempt() {
	e.mu.Lock()
	e.health.State = models.ProviderReconnecting
	e.health.LastAttemptAt = time.Now().UTC()
	e.mu.Unlock()
	e.persistHealth()
}

func (e *Engine) noteProviderFailure(code string, state models.ProviderState, action string, err error) {
	e.mu.Lock()
	e.health.State, e.health.ErrorCode, e.health.Message, e.health.Action = state, code, userProviderMessage(code), action
	e.mu.Unlock()
	e.persistHealth()
	if e.debug {
		_ = e.logs.Write("warning", "provider", err.Error())
	}
}

func (e *Engine) noteProviderRetryAt(next time.Time) {
	if next.IsZero() {
		return
	}
	e.mu.Lock()
	e.health.NextRetryAt = next
	e.mu.Unlock()
	e.persistHealth()
}

func (e *Engine) noteProviderSuccess() {
	now := time.Now().UTC()
	e.mu.Lock()
	e.health = models.ProviderHealth{State: models.ProviderOnline, LastSuccessAt: now, LastAttemptAt: now}
	e.status.LastCloudUpload = now
	e.status.LastSyncTime = now
	e.status.LastError = ""
	e.mu.Unlock()
	e.persistHealth()
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "provider", Severity: "info", Title: "Cloud connection available", Detail: "Pending Cloud Sync work can be uploaded.", DeduplicationKey: ""})
}

func (e *Engine) refreshHealth() {
	settings := e.cfg.Get()
	if settings.ProtectionTarget == models.ProtectionTargetLocal {
		e.mu.Lock()
		e.health = models.ProviderHealth{State: models.ProviderLocalOnly, Message: "Revisions are stored on this computer only."}
		e.mu.Unlock()
		return
	}
	if health, err := e.logs.ProviderHealth(string(settings.ProtectionTarget)); err == nil && health.State != "" {
		e.mu.Lock()
		e.health = health
		e.mu.Unlock()
		return
	}
	e.mu.Lock()
	e.health = models.ProviderHealth{State: models.ProviderReconnecting, Message: "Provider connection has not been checked yet."}
	e.mu.Unlock()
}

func (e *Engine) persistHealth() {
	e.mu.RLock()
	value := e.health
	e.mu.RUnlock()
	settings := e.cfg.Get()
	_ = e.logs.SaveProviderHealth(string(settings.ProtectionTarget), value)
	e.signalStatus()
}

func (e *Engine) noteProtectionComplete(game models.Game, revisionID string, uploaded bool) {
	detail := "A durable local revision was created."
	if uploaded {
		detail = "A durable local revision was created and uploaded."
	}
	_ = e.logs.WriteActivity(models.ActivityEvent{Category: "sync", Severity: "info", GameID: game.ID, Title: "Cloud Sync completed", Detail: detail})
	e.mu.Lock()
	delete(e.dirty, game.ID)
	e.mu.Unlock()
	e.signalStatus()
}

func (e *Engine) finishJob(id, state, stage string, changed int, bytes int64, code, message string) {
	e.updateJob(id, func(job *models.SyncJob) {
		job.State, job.Stage, job.Progress = state, stage, 100
		job.FinishedAt = time.Now().UTC()
		job.ChangedFiles, job.Bytes = changed, bytes
		job.ErrorCode, job.Error = code, message
	})
}

func (e *Engine) storeJob(job models.SyncJob) {
	e.storeJobMemory(job)
	_ = e.logs.SaveProtectionJob(job)
}

func (e *Engine) storeJobMemory(job models.SyncJob) {
	e.jobsMu.Lock()
	defer e.jobsMu.Unlock()
	if _, exists := e.jobs[job.ID]; !exists {
		e.jobOrder = append(e.jobOrder, job.ID)
	}
	e.jobs[job.ID] = job
	if len(e.jobOrder) > 200 {
		delete(e.jobs, e.jobOrder[0])
		e.jobOrder = e.jobOrder[1:]
	}
}

func (e *Engine) job(id string) models.SyncJob {
	e.jobsMu.RLock()
	job := e.jobs[id]
	e.jobsMu.RUnlock()
	if job.ID != "" {
		return job
	}
	for _, value := range e.Jobs("") {
		if value.ID == id {
			return value
		}
	}
	return models.SyncJob{ID: id}
}

func (e *Engine) updateJob(id string, edit func(*models.SyncJob)) {
	if id == "" {
		return
	}
	job := e.job(id)
	if job.ID == "" {
		return
	}
	edit(&job)
	job.UpdatedAt = time.Now().UTC()
	e.storeJobMemory(job)
	_ = e.logs.SaveProtectionJob(job)
	e.signalStatus()
}

func (e *Engine) setRunning(value bool) {
	e.mu.Lock()
	e.status.Running = value
	e.mu.Unlock()
	e.signalStatus()
}

func (e *Engine) signalStatus() {
	select {
	case e.statusUpdates <- struct{}{}:
	default:
	}
}

func (e *Engine) SortedRunningGames() []string {
	e.mu.RLock()
	ids := make([]string, 0, len(e.running))
	for id := range e.running {
		ids = append(ids, id)
	}
	e.mu.RUnlock()
	sort.Strings(ids)
	return ids
}
