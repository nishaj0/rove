package gamemon

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/process"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

const (
	// Keep the idle interval short enough to observe games which launch and
	// crash before their first long background scan.
	idlePollInterval     = 2 * time.Second
	activePollInterval   = 2 * time.Second
	defaultFlushInterval = 15 * time.Second
	launchProbeInterval  = 250 * time.Millisecond
	launchProbeWindow    = 10 * time.Second
	// guard against unrealistic jumps to avoid overcounting playtime on clock skew/sleep.
	maxClockGap = 60 * time.Second
)

// tracks transient runtime playtime state for one game.
type session struct {
	running        bool
	lastTick       time.Time
	lastFlush      time.Time
	pendingSeconds int64
}

// represents a pending playtime write to sqlite.
type flushOp struct {
	gameID  string
	seconds int64
	at      time.Time
}

// polls running processes and accumulates per-game playtime.
type Monitor struct {
	cfg              *config.Manager
	logs             *logstore.Store
	onRunningChanged func([]string)
	onError          func(error)
	snapshotFn       func() map[string]bool

	pollInterval  time.Duration
	flushInterval time.Duration

	mu         sync.RWMutex
	pollMu     sync.Mutex
	sessions   map[string]*session
	runningIDs map[string]struct{}
	started    atomic.Bool
}

// creates a monitor with optional callbacks for running-set changes and errors.
func New(cfg *config.Manager, logs *logstore.Store, onRunningChanged func([]string), onError func(error)) *Monitor {
	return &Monitor{
		cfg:              cfg,
		logs:             logs,
		onRunningChanged: onRunningChanged,
		onError:          onError,
		snapshotFn:       nil,
		pollInterval:     idlePollInterval,
		flushInterval:    defaultFlushInterval,
		sessions:         make(map[string]*session),
		runningIDs:       make(map[string]struct{}),
	}
}

// begins polling and flushes outstanding playtime when context is canceled.
func (m *Monitor) Start(ctx context.Context) {
	if !m.started.CompareAndSwap(false, true) {
		return
	}
	m.pollOnce(time.Now())

	go func() {
		for {
			m.mu.RLock()
			interval := m.pollInterval
			if len(m.runningIDs) > 0 {
				interval = activePollInterval
			}
			m.mu.RUnlock()
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				m.flushAll(time.Now())
				return
			case now := <-timer.C:
				m.pollOnce(now)
			}
		}
	}()
}

// updates the callback fired when running game ids change.
func (m *Monitor) SetOnRunningChanged(onRunningChanged func([]string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRunningChanged = onRunningChanged
}

// triggers an immediate process snapshot when monitor is active.
func (m *Monitor) ForcePoll() {
	if !m.started.Load() {
		return
	}
	m.pollOnce(time.Now())
}

// ProbeLaunch quickly samples a just-started executable until it is observed.
// This closes the gap where cmd.exe returns before Windows has created the
// game's process and the normal idle poll would otherwise miss a short crash.
func (m *Monitor) ProbeLaunch(gameID string) {
	if strings.TrimSpace(gameID) == "" || !m.started.Load() {
		return
	}
	go func() {
		deadline := time.Now().Add(launchProbeWindow)
		for {
			m.ForcePoll()
			if m.IsRunning(gameID) || time.Now().After(deadline) || !m.started.Load() {
				return
			}
			time.Sleep(launchProbeInterval)
		}
	}()
}

// reports whether a specific game is currently detected as running.
func (m *Monitor) IsRunning(gameID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.runningIDs[gameID]
	return ok
}

// returns unflushed playtime currently buffered in memory.
func (m *Monitor) PendingSeconds(gameID string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.sessions[gameID]
	if s == nil {
		return 0
	}
	return s.pendingSeconds
}

// returns a stable sorted copy of detected running game ids.
func (m *Monitor) RunningIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.runningIDs))
	for id := range m.runningIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// updates sessions from process snapshots and schedules playtime flushes.
func (m *Monitor) pollOnce(now time.Time) {
	m.pollMu.Lock()
	defer m.pollMu.Unlock()
	games := m.cfg.Get().Games
	snapshot := m.snapshotFn
	if snapshot == nil {
		snapshot = m.snapshotRunningExecutables
	}
	runningByExecutable := snapshot()
	tracked := make(map[string]models.Game, len(games))
	for _, game := range games {
		tracked[game.ID] = game
	}

	flushes := make([]flushOp, 0)
	nextRunning := make(map[string]struct{})

	m.mu.Lock()
	for id, s := range m.sessions {
		if _, ok := tracked[id]; ok {
			continue
		}
		if s.running && s.pendingSeconds > 0 {
			flushes = append(flushes, flushOp{gameID: id, seconds: s.pendingSeconds, at: now})
		}
		delete(m.sessions, id)
	}

	for _, game := range games {
		normalizedExe := normalizePath(game.ExecutablePath)
		if normalizedExe == "" {
			continue
		}
		running := runningByExecutable[normalizedExe]
		s := m.sessions[game.ID]
		if s == nil {
			s = &session{lastTick: now, lastFlush: now}
			m.sessions[game.ID] = s
		}

		if s.running {
			// accumulate elapsed time only for sane deltas between poll ticks.
			// Count the final observed interval as well: a process may disappear
			// between polls because it crashed or exited normally.
			delta := now.Sub(s.lastTick)
			if delta > 0 && delta <= maxClockGap {
				s.pendingSeconds += int64(delta.Seconds())
			}
		}
		if s.running && !running {
			// flush immediately on transition to stopped so no session time is lost.
			if s.pendingSeconds > 0 {
				flushes = append(flushes, flushOp{gameID: game.ID, seconds: s.pendingSeconds, at: now})
				s.pendingSeconds = 0
			}
		}
		if running {
			nextRunning[game.ID] = struct{}{}
			// periodically persist while game keeps running to reduce buffered time on crash.
			if now.Sub(s.lastFlush) >= m.flushInterval && s.pendingSeconds > 0 {
				flushes = append(flushes, flushOp{gameID: game.ID, seconds: s.pendingSeconds, at: now})
				s.pendingSeconds = 0
				s.lastFlush = now
			}
		}

		s.running = running
		s.lastTick = now
	}

	runningChanged := !sameSet(m.runningIDs, nextRunning)
	if runningChanged {
		m.runningIDs = nextRunning
	}
	m.mu.Unlock()

	m.flushOps(flushes)
	if runningChanged && m.onRunningChanged != nil {
		ids := make([]string, 0, len(nextRunning))
		for id := range nextRunning {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		m.onRunningChanged(ids)
	}
}

// writes all buffered playtime and clears running state.
func (m *Monitor) flushAll(now time.Time) {
	m.mu.Lock()
	flushes := make([]flushOp, 0, len(m.sessions))
	for gameID, s := range m.sessions {
		if s.pendingSeconds > 0 {
			flushes = append(flushes, flushOp{gameID: gameID, seconds: s.pendingSeconds, at: now})
			s.pendingSeconds = 0
		}
		s.running = false
	}
	m.runningIDs = make(map[string]struct{})
	m.mu.Unlock()
	m.flushOps(flushes)
}

// persists buffered playtime and requeues on write failure.
func (m *Monitor) flushOps(flushes []flushOp) {
	for _, op := range flushes {
		if op.seconds <= 0 {
			continue
		}
		if err := m.logs.AddPlaytime(op.gameID, op.seconds, op.at); err != nil {
			if m.onError != nil {
				m.onError(fmt.Errorf("persist playtime: %w", err))
			}
			m.mu.Lock()
			if s := m.sessions[op.gameID]; s != nil {
				s.pendingSeconds += op.seconds
			}
			m.mu.Unlock()
		}
	}
}

// returns normalized executable paths for active processes.
func (m *Monitor) snapshotRunningExecutables() map[string]bool {
	running := make(map[string]bool)
	targetNames := make(map[string]struct{})
	for _, game := range m.cfg.Get().Games {
		name := strings.ToLower(filepath.Base(strings.TrimSpace(game.ExecutablePath)))
		if name != "" {
			targetNames[name] = struct{}{}
		}
	}
	if len(targetNames) == 0 {
		return running
	}
	procs, err := process.Processes()
	if err != nil {
		if m.onError != nil {
			m.onError(fmt.Errorf("list processes: %w", err))
		}
		return running
	}
	for _, proc := range procs {
		name, err := proc.Name()
		if err != nil {
			continue
		}
		if _, tracked := targetNames[strings.ToLower(name)]; !tracked {
			continue
		}
		exe, err := proc.Exe()
		if err != nil || exe == "" {
			continue
		}
		running[normalizePath(exe)] = true
	}
	return running
}

// canonicalizes executable path values for consistent map keys.
func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(trimmed))
}

// compares running-id sets to detect change notifications.
func sameSet(a map[string]struct{}, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
