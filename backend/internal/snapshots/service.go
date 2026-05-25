package snapshots

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

const periodicSnapshotCadence = 30 * time.Second

// Service manages background and user-triggered versioned snapshots.
type Service struct {
	cfg  *config.Manager
	logs *logstore.Store

	opMu sync.Mutex

	mu             sync.Mutex
	running        map[string]struct{}
	lastSnapshotAt map[string]time.Time
	started        atomic.Bool
	stopOnce       sync.Once
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

func New(cfg *config.Manager, logs *logstore.Store) *Service {
	return &Service{
		cfg:            cfg,
		logs:           logs,
		running:        make(map[string]struct{}),
		lastSnapshotAt: make(map[string]time.Time),
		stopCh:         make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) {
	if !s.started.CompareAndSwap(false, true) {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(periodicSnapshotCadence)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.runPeriodicChecks()
			}
		}
	}()
}

func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *Service) UpdateRunning(runningIDs []string) {
	next := make(map[string]struct{}, len(runningIDs))
	for _, id := range runningIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			next[id] = struct{}{}
		}
	}

	settings := s.cfg.Get()
	now := time.Now()
	exitIDs := make([]string, 0)

	s.mu.Lock()
	for id := range next {
		if _, ok := s.running[id]; !ok {
			s.lastSnapshotAt[id] = now
		}
	}
	for id := range s.running {
		if _, ok := next[id]; ok {
			continue
		}
		delete(s.lastSnapshotAt, id)
		if settings.SnapshotOnExit {
			exitIDs = append(exitIDs, id)
		}
	}
	s.running = next
	s.mu.Unlock()

	for _, gameID := range exitIDs {
		go s.createSnapshot(gameID, models.SnapshotTriggerExit)
	}
}

func (s *Service) List(gameID string) ([]models.SnapshotSummary, error) {
	game, settings, err := s.resolveGame(gameID)
	if err != nil {
		return nil, err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()
	return listSnapshotsForGame(game, settings)
}

func (s *Service) CreateManual(gameID string) (CreateResult, error) {
	return s.createSnapshot(gameID, models.SnapshotTriggerManual)
}

func (s *Service) Restore(gameID, snapshotID string) error {
	if strings.TrimSpace(snapshotID) == "" {
		return errors.New("snapshotId is required")
	}

	game, settings, err := s.resolveGame(gameID)
	if err != nil {
		return err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := restoreSnapshotForGame(game, settings, snapshotID); err != nil {
		return err
	}
	s.logInfo("snapshot", fmt.Sprintf("restored snapshot %s for %s", snapshotID, game.Name))
	return nil
}

func (s *Service) Delete(gameID, snapshotID string) error {
	if strings.TrimSpace(snapshotID) == "" {
		return errors.New("snapshotId is required")
	}

	game, settings, err := s.resolveGame(gameID)
	if err != nil {
		return err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := deleteSnapshotForGame(game, settings, snapshotID); err != nil {
		return err
	}
	s.logInfo("snapshot", fmt.Sprintf("deleted snapshot %s for %s", snapshotID, game.Name))
	return nil
}

func (s *Service) createSnapshot(gameID, trigger string) (CreateResult, error) {
	game, settings, err := s.resolveGame(gameID)
	if err != nil {
		return CreateResult{}, err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	if strings.TrimSpace(settings.SnapshotVaultPath) == "" {
		s.logWarn("snapshot", fmt.Sprintf("snapshot vault path not configured; skipping %s snapshot for %s", trigger, game.Name))
		return CreateResult{Status: "skipped"}, nil
	}

	result, err := createSnapshotForGame(game, settings, trigger)
	if err != nil {
		return CreateResult{}, err
	}
	if result.Status == "created" {
		s.logInfo("snapshot", fmt.Sprintf("created %s snapshot for %s", trigger, game.Name))
		s.noteSnapshotAt(game.ID, time.Now())
	}
	return result, nil
}

func (s *Service) runPeriodicChecks() {
	settings := s.cfg.Get()
	if !settings.SnapshotPeriodic {
		return
	}

	interval := time.Duration(settings.SnapshotIntervalMinutes) * time.Minute
	now := time.Now()

	due := make([]string, 0)
	s.mu.Lock()
	for gameID := range s.running {
		last := s.lastSnapshotAt[gameID]
		if last.IsZero() {
			s.lastSnapshotAt[gameID] = now
			last = now
		}
		if now.Sub(last) >= interval {
			s.lastSnapshotAt[gameID] = now
			due = append(due, gameID)
		}
	}
	s.mu.Unlock()

	for _, gameID := range due {
		go func(id string) {
			if _, err := s.createSnapshot(id, models.SnapshotTriggerPeriodic); err != nil {
				s.logError("snapshot", fmt.Sprintf("periodic snapshot failed for %s: %v", id, err))
			}
		}(gameID)
	}
}

func (s *Service) resolveGame(gameID string) (models.Game, models.Settings, error) {
	gameID = strings.TrimSpace(gameID)
	if gameID == "" {
		return models.Game{}, models.Settings{}, errors.New("gameId is required")
	}

	game, ok := s.cfg.FindGame(gameID)
	if !ok {
		return models.Game{}, models.Settings{}, errors.New("game not found")
	}
	return game, s.cfg.Get(), nil
}

func (s *Service) noteSnapshotAt(gameID string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.running[gameID]; ok {
		s.lastSnapshotAt[gameID] = at
	}
}

func (s *Service) logInfo(event, message string) {
	if s.logs != nil {
		_ = s.logs.Write("info", event, message)
	}
}

func (s *Service) logWarn(event, message string) {
	if s.logs != nil {
		_ = s.logs.Write("warn", event, message)
	}
}

func (s *Service) logError(event, message string) {
	if s.logs != nil {
		_ = s.logs.Write("error", event, message)
	}
}
