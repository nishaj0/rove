package autobackup

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/branding"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

const periodicCheckCadence = 30 * time.Second

// coordinates periodic and exit-triggered local backups for running games.
type Service struct {
	cfg    *config.Manager
	logs   *logstore.Store
	zipper *backup.Service
	debug  bool

	mu          sync.Mutex
	running     map[string]struct{}
	pendingExit map[string]*time.Timer
	inFlight    map[string]struct{}
	lastSuccess map[string]time.Time
	lastLoaded  map[string]bool

	started  atomic.Bool
	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// builds a local auto-backup coordinator tied to config and sqlite metadata.
func New(cfg *config.Manager, logs *logstore.Store, zipper *backup.Service, debug bool) *Service {
	return &Service{
		cfg:         cfg,
		logs:        logs,
		zipper:      zipper,
		debug:       debug,
		running:     make(map[string]struct{}),
		pendingExit: make(map[string]*time.Timer),
		inFlight:    make(map[string]struct{}),
		lastSuccess: make(map[string]time.Time),
		lastLoaded:  make(map[string]bool),
		stopCh:      make(chan struct{}),
	}
}

// starts periodic checks and reacts to context cancellation for shutdown.
func (s *Service) Start(ctx context.Context) {
	if !s.started.CompareAndSwap(false, true) {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(periodicCheckCadence)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.stopTimers()
				return
			case <-s.stopCh:
				s.stopTimers()
				return
			case <-ticker.C:
				s.runPeriodicChecks()
			}
		}
	}()
}

// applies running-game snapshot updates and schedules or cancels final backups.
func (s *Service) UpdateRunning(runningIDs []string) {
	next := make(map[string]struct{}, len(runningIDs))
	for _, id := range runningIDs {
		if strings.TrimSpace(id) != "" {
			next[id] = struct{}{}
		}
	}

	settings := s.cfg.Get()
	gamesByID := make(map[string]models.Game, len(settings.Games))
	for _, game := range settings.Games {
		gamesByID[game.ID] = game
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range next {
		if _, ok := s.running[id]; ok {
			continue
		}
		if timer, ok := s.pendingExit[id]; ok {
			timer.Stop()
			delete(s.pendingExit, id)
		}
	}

	for id := range s.running {
		if _, ok := next[id]; ok {
			continue
		}
		game, ok := gamesByID[id]
		if !ok || !game.BackupPolicy.Enabled {
			continue
		}
		delay := time.Duration(game.BackupPolicy.ExitDelaySeconds) * time.Second
		if timer, ok := s.pendingExit[id]; ok {
			timer.Stop()
		}
		s.pendingExit[id] = time.AfterFunc(delay, func() {
			s.onExitDelayElapsed(id)
		})
	}

	s.running = next
}

// stops timers and goroutines for clean shutdown.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.stopTimers()
	s.wg.Wait()
}

func (s *Service) stopTimers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, timer := range s.pendingExit {
		timer.Stop()
		delete(s.pendingExit, id)
	}
}

func (s *Service) onExitDelayElapsed(gameID string) {
	s.mu.Lock()
	delete(s.pendingExit, gameID)
	_, stillRunning := s.running[gameID]
	s.mu.Unlock()
	if stillRunning {
		return
	}
	s.enqueueBackup(gameID, models.BackupReasonFinal)
}

func (s *Service) runPeriodicChecks() {
	now := time.Now()
	settings := s.cfg.Get()
	gamesByID := make(map[string]models.Game, len(settings.Games))
	for _, game := range settings.Games {
		gamesByID[game.ID] = game
	}

	s.mu.Lock()
	runningIDs := make([]string, 0, len(s.running))
	for id := range s.running {
		runningIDs = append(runningIDs, id)
	}
	s.mu.Unlock()

	for _, gameID := range runningIDs {
		game, ok := gamesByID[gameID]
		if !ok || !game.BackupPolicy.Enabled {
			continue
		}
		if !s.periodicDue(gameID, now, game.BackupPolicy.IntervalMinutes) {
			continue
		}
		s.enqueueBackup(gameID, models.BackupReasonPeriodic)
	}
}

func (s *Service) periodicDue(gameID string, now time.Time, intervalMinutes int) bool {
	last, ok := s.loadLastSuccess(gameID)
	if !ok {
		return true
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	return now.Sub(last) >= interval
}

func (s *Service) loadLastSuccess(gameID string) (time.Time, bool) {
	s.mu.Lock()
	if s.lastLoaded[gameID] {
		value := s.lastSuccess[gameID]
		s.mu.Unlock()
		if value.IsZero() {
			return time.Time{}, false
		}
		return value, true
	}
	s.mu.Unlock()

	entries, err := s.logs.BackupHistory(gameID, 1)
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("read backup history failed for %s: %v", gameID, err))
		s.mu.Lock()
		s.lastLoaded[gameID] = true
		s.lastSuccess[gameID] = time.Time{}
		s.mu.Unlock()
		return time.Time{}, false
	}

	var last time.Time
	if len(entries) > 0 {
		last = entries[0].CreatedAt
	}

	s.mu.Lock()
	s.lastLoaded[gameID] = true
	s.lastSuccess[gameID] = last
	s.mu.Unlock()

	if last.IsZero() {
		return time.Time{}, false
	}
	return last, true
}

func (s *Service) enqueueBackup(gameID, reason string) {
	s.mu.Lock()
	if _, running := s.inFlight[gameID]; running {
		s.mu.Unlock()
		return
	}
	s.inFlight[gameID] = struct{}{}
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.inFlight, gameID)
			s.mu.Unlock()
		}()
		s.runBackup(gameID, reason)
	}()
}

func (s *Service) runBackup(gameID, reason string) {
	game, ok := s.cfg.FindGame(gameID)
	if !ok || !game.BackupPolicy.Enabled {
		return
	}
	settings := s.cfg.Get()
	signature, err := computeSaveSignature(game.Path, settings.IgnorePatterns)
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("compute signature failed for %s: %v", game.Name, err))
		return
	}

	lastSignature, err := s.logs.LatestBackupSignature(game.ID)
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("read signature failed for %s: %v", game.Name, err))
		return
	}
	if signature == lastSignature {
		if s.debug {
			_ = s.logs.Write("debug", "backup", fmt.Sprintf("skip %s backup for %s: unchanged", reason, game.Name))
		}
		return
	}

	backupsRoot := filepath.Join(filepath.Dir(s.cfg.Path()), branding.BackupsDirName)
	outputPath, err := s.zipper.ExportAutoGame(game, backupsRoot, reason)
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("create %s backup failed for %s: %v", reason, game.Name, err))
		return
	}

	now := time.Now().UTC()
	err = s.logs.InsertBackupHistory(models.BackupHistoryEntry{
		GameID:    game.ID,
		Path:      outputPath,
		CreatedAt: now,
		Reason:    reason,
		Signature: signature,
		Status:    models.BackupStatusOK,
	})
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("record backup history failed for %s: %v", game.Name, err))
	}

	removedPaths, err := s.logs.PruneBackupHistory(game.ID, game.BackupPolicy.RetentionCount)
	if err != nil {
		_ = s.logs.Write("error", "backup", fmt.Sprintf("prune backup history failed for %s: %v", game.Name, err))
	} else {
		for _, path := range removedPaths {
			if strings.TrimSpace(path) == "" || path == outputPath {
				continue
			}
			if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
				_ = s.logs.Write("error", "backup", fmt.Sprintf("remove old backup failed for %s: %v", game.Name, removeErr))
			}
		}
	}

	s.mu.Lock()
	s.lastLoaded[gameID] = true
	s.lastSuccess[gameID] = now
	s.mu.Unlock()

	_ = s.logs.Write("info", "backup", fmt.Sprintf("created %s auto-backup for %s", reason, game.Name))
}

func computeSaveSignature(root string, ignorePatterns []string) (string, error) {
	h := sha1.New() // #nosec G401 -- fast non-cryptographic fingerprinting for change detection.
	fileCount := 0
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldIgnore(path, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnore(path, ignorePatterns) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		normalized := strings.ToLower(filepath.ToSlash(rel))
		_, _ = h.Write([]byte(normalized))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(strconv.FormatInt(info.Size(), 10)))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
		_, _ = h.Write([]byte("\n"))
		fileCount++
		return nil
	})
	if walkErr != nil {
		return "", walkErr
	}

	sum := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%d", sum, fileCount), nil
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
		needle := strings.ToLower(string(os.PathSeparator) + p + string(os.PathSeparator))
		if strings.Contains(strings.ToLower(path), needle) {
			return true
		}
	}
	return false
}
