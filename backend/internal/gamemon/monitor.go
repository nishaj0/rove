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
	defaultPollInterval  = 3 * time.Second
	defaultFlushInterval = 15 * time.Second
	maxClockGap          = 60 * time.Second
)

type session struct {
	running        bool
	lastTick       time.Time
	lastFlush      time.Time
	pendingSeconds int64
}

type flushOp struct {
	gameID  string
	seconds int64
	at      time.Time
}

type Monitor struct {
	cfg              *config.Manager
	logs             *logstore.Store
	onRunningChanged func([]string)
	onError          func(error)

	pollInterval  time.Duration
	flushInterval time.Duration

	mu         sync.RWMutex
	sessions   map[string]*session
	runningIDs map[string]struct{}
	started    atomic.Bool
}

func New(cfg *config.Manager, logs *logstore.Store, onRunningChanged func([]string), onError func(error)) *Monitor {
	return &Monitor{
		cfg:              cfg,
		logs:             logs,
		onRunningChanged: onRunningChanged,
		onError:          onError,
		pollInterval:     defaultPollInterval,
		flushInterval:    defaultFlushInterval,
		sessions:         make(map[string]*session),
		runningIDs:       make(map[string]struct{}),
	}
}

func (m *Monitor) Start(ctx context.Context) {
	if !m.started.CompareAndSwap(false, true) {
		return
	}
	m.pollOnce(time.Now())

	ticker := time.NewTicker(m.pollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				m.flushAll(time.Now())
				return
			case now := <-ticker.C:
				m.pollOnce(now)
			}
		}
	}()
}

func (m *Monitor) SetOnRunningChanged(onRunningChanged func([]string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRunningChanged = onRunningChanged
}

func (m *Monitor) ForcePoll() {
	if !m.started.Load() {
		return
	}
	m.pollOnce(time.Now())
}

func (m *Monitor) IsRunning(gameID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.runningIDs[gameID]
	return ok
}

func (m *Monitor) PendingSeconds(gameID string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.sessions[gameID]
	if s == nil {
		return 0
	}
	return s.pendingSeconds
}

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

func (m *Monitor) pollOnce(now time.Time) {
	games := m.cfg.Get().Games
	runningByExecutable := m.snapshotRunningExecutables()
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

		if s.running && running {
			delta := now.Sub(s.lastTick)
			if delta > 0 && delta <= maxClockGap {
				s.pendingSeconds += int64(delta.Seconds())
			}
		}
		if s.running && !running {
			if s.pendingSeconds > 0 {
				flushes = append(flushes, flushOp{gameID: game.ID, seconds: s.pendingSeconds, at: now})
				s.pendingSeconds = 0
			}
		}
		if running {
			nextRunning[game.ID] = struct{}{}
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

func (m *Monitor) snapshotRunningExecutables() map[string]bool {
	running := make(map[string]bool)
	procs, err := process.Processes()
	if err != nil {
		if m.onError != nil {
			m.onError(fmt.Errorf("list processes: %w", err))
		}
		return running
	}
	for _, proc := range procs {
		exe, err := proc.Exe()
		if err != nil || exe == "" {
			continue
		}
		running[normalizePath(exe)] = true
	}
	return running
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(trimmed))
}

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
