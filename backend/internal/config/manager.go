package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/branding"
	"cloud-saver/backend/internal/models"
	runtimecfg "cloud-saver/backend/internal/runtime"
)

const (
	defaultBackupIntervalMinutes  = 10
	defaultBackupExitDelaySeconds = 10
	defaultBackupRetentionCount   = 5
	defaultSnapshotMaxPerGame     = 5
	defaultSnapshotIntervalMins   = 15
	minBackupIntervalMinutes      = 3
	maxBackupIntervalMinutes      = 120
	minBackupExitDelaySeconds     = 3
	maxBackupExitDelaySeconds     = 120
	minBackupRetentionCount       = 1
	maxBackupRetentionCount       = 30
	minSnapshotMaxPerGame         = 1
	maxSnapshotMaxPerGame         = 50
	minSnapshotIntervalMins       = 1
	maxSnapshotIntervalMins       = 1440
)

func defaultBackupPolicy() models.BackupPolicy {
	return models.BackupPolicy{
		Enabled:          true,
		IntervalMinutes:  defaultBackupIntervalMinutes,
		ExitDelaySeconds: defaultBackupExitDelaySeconds,
		RetentionCount:   defaultBackupRetentionCount,
	}
}

func normalizeBackupPolicy(policy *models.BackupPolicy) bool {
	if policy == nil {
		return false
	}
	changed := false
	defaults := defaultBackupPolicy()

	if policy.IntervalMinutes < minBackupIntervalMinutes || policy.IntervalMinutes > maxBackupIntervalMinutes {
		policy.IntervalMinutes = defaults.IntervalMinutes
		changed = true
	}
	if policy.ExitDelaySeconds < minBackupExitDelaySeconds || policy.ExitDelaySeconds > maxBackupExitDelaySeconds {
		policy.ExitDelaySeconds = defaults.ExitDelaySeconds
		changed = true
	}
	if policy.RetentionCount < minBackupRetentionCount || policy.RetentionCount > maxBackupRetentionCount {
		policy.RetentionCount = defaults.RetentionCount
		changed = true
	}

	return changed
}

func validateBackupPolicy(policy models.BackupPolicy) error {
	if policy.IntervalMinutes < minBackupIntervalMinutes || policy.IntervalMinutes > maxBackupIntervalMinutes {
		return fmt.Errorf("backup interval must be between %d and %d minutes", minBackupIntervalMinutes, maxBackupIntervalMinutes)
	}
	if policy.ExitDelaySeconds < minBackupExitDelaySeconds || policy.ExitDelaySeconds > maxBackupExitDelaySeconds {
		return fmt.Errorf("backup exit delay must be between %d and %d seconds", minBackupExitDelaySeconds, maxBackupExitDelaySeconds)
	}
	if policy.RetentionCount < minBackupRetentionCount || policy.RetentionCount > maxBackupRetentionCount {
		return fmt.Errorf("backup retention count must be between %d and %d", minBackupRetentionCount, maxBackupRetentionCount)
	}
	return nil
}

// coordinates in-memory settings state with on-disk persistence.
type Manager struct {
	mu       sync.RWMutex
	path     string
	settings models.Settings
}

// creates a manager and loads settings from disk, creating defaults when missing.
func New(path string) (*Manager, error) {
	if path == "" {
		resolved, err := defaultPath()
		if err != nil {
			return nil, err
		}
		path = resolved
	}

	m := &Manager{path: path}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

// resolves the per-user settings.json location under the app data directory.
func defaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, branding.InternalName, "settings.json"), nil
}

// returns baseline values used for first run and normalization fallback.
func DefaultSettings() models.Settings {
	return models.Settings{
		Port:                    runtimecfg.DefaultAPIPort,
		SyncIntervalSeconds:     30,
		ConflictPolicy:          models.ConflictLatestWins,
		ThemeMode:               models.ThemeDark,
		Autostart:               false,
		IgnorePatterns:          []string{"*.tmp", "*.log", "cache"},
		SnapshotVaultPath:       "",
		SnapshotMaxPerGame:      defaultSnapshotMaxPerGame,
		SnapshotOnExit:          true,
		SnapshotPeriodic:        false,
		SnapshotIntervalMinutes: defaultSnapshotIntervalMins,
		DeviceID:                uuid.NewString(),
		SyncRetention:           10,
		ReviewConflicts:         true,
		Games:                   []models.Game{},
	}
}

// returns the active settings file path.
func (m *Manager) Path() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.path
}

// returns a safe cloned copy of in-memory settings.
func (m *Manager) Get() models.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cloned := m.settings
	cloned.Games = append([]models.Game{}, m.settings.Games...)
	cloned.IgnorePatterns = append([]string{}, m.settings.IgnorePatterns...)
	return cloned
}

// validates and persists mutable settings fields from api input.
func (m *Manager) Update(next models.Settings) error {
	if next.Port <= 0 || next.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if next.SyncIntervalSeconds < 5 {
		return errors.New("sync interval must be >= 5 seconds")
	}
	if next.ConflictPolicy != models.ConflictLatestWins && next.ConflictPolicy != models.ConflictKeepBoth {
		return errors.New("invalid conflict policy")
	}
	if next.ThemeMode != models.ThemeDark && next.ThemeMode != models.ThemeLight && next.ThemeMode != models.ThemeSystem {
		return errors.New("invalid theme mode")
	}
	if next.SnapshotMaxPerGame < minSnapshotMaxPerGame || next.SnapshotMaxPerGame > maxSnapshotMaxPerGame {
		return fmt.Errorf("snapshot max per game must be between %d and %d", minSnapshotMaxPerGame, maxSnapshotMaxPerGame)
	}
	if next.SnapshotIntervalMinutes < minSnapshotIntervalMins || next.SnapshotIntervalMinutes > maxSnapshotIntervalMins {
		return fmt.Errorf("snapshot interval must be between %d and %d minutes", minSnapshotIntervalMins, maxSnapshotIntervalMins)
	}
	if strings.TrimSpace(next.DeviceID) == "" {
		return errors.New("device id is required")
	}
	if next.SyncRetention < 1 || next.SyncRetention > 50 {
		return errors.New("sync retention must be between 1 and 50")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.Port = next.Port
	m.settings.SyncIntervalSeconds = next.SyncIntervalSeconds
	m.settings.ConflictPolicy = next.ConflictPolicy
	m.settings.ThemeMode = next.ThemeMode
	m.settings.Autostart = next.Autostart
	m.settings.IgnorePatterns = append([]string{}, next.IgnorePatterns...)
	m.settings.SnapshotVaultPath = next.SnapshotVaultPath
	m.settings.SnapshotMaxPerGame = next.SnapshotMaxPerGame
	m.settings.SnapshotOnExit = next.SnapshotOnExit
	m.settings.SnapshotPeriodic = next.SnapshotPeriodic
	m.settings.SnapshotIntervalMinutes = next.SnapshotIntervalMinutes
	m.settings.SyncRoot = next.SyncRoot
	m.settings.DeviceID = next.DeviceID
	m.settings.SyncRetention = next.SyncRetention
	m.settings.ReviewConflicts = next.ReviewConflicts
	return m.saveLocked()
}

// reads settings from disk and normalizes missing or legacy fields.
func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	payload, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		m.settings = DefaultSettings()
		return m.saveLocked()
	}
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	if len(payload) == 0 {
		m.settings = DefaultSettings()
		return m.saveLocked()
	}

	// trim utf-8 bom so settings written by some windows tools still decode.
	if bytes.HasPrefix(payload, []byte{0xEF, 0xBB, 0xBF}) {
		payload = payload[3:]
	}

	if err := json.Unmarshal(payload, &m.settings); err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}
	if normalizeSettings(&m.settings) {
		return m.saveLocked()
	}
	return nil
}

// backfills defaults for zero, nil, or invalid fields from older files.
func normalizeSettings(settings *models.Settings) bool {
	if settings == nil {
		return false
	}
	changed := false
	defaults := DefaultSettings()

	if settings.Port == 0 {
		settings.Port = defaults.Port
		changed = true
	}
	if settings.SyncIntervalSeconds == 0 {
		settings.SyncIntervalSeconds = defaults.SyncIntervalSeconds
		changed = true
	}
	if settings.ConflictPolicy != models.ConflictLatestWins && settings.ConflictPolicy != models.ConflictKeepBoth {
		settings.ConflictPolicy = defaults.ConflictPolicy
		changed = true
	}
	if settings.ThemeMode != models.ThemeDark && settings.ThemeMode != models.ThemeLight && settings.ThemeMode != models.ThemeSystem {
		settings.ThemeMode = defaults.ThemeMode
		changed = true
	}
	if settings.IgnorePatterns == nil {
		settings.IgnorePatterns = append([]string{}, defaults.IgnorePatterns...)
		changed = true
	}
	if settings.Games == nil {
		settings.Games = []models.Game{}
		changed = true
	}
	if settings.SnapshotMaxPerGame < minSnapshotMaxPerGame || settings.SnapshotMaxPerGame > maxSnapshotMaxPerGame {
		settings.SnapshotMaxPerGame = defaults.SnapshotMaxPerGame
		changed = true
	}
	if settings.SnapshotIntervalMinutes < minSnapshotIntervalMins || settings.SnapshotIntervalMinutes > maxSnapshotIntervalMins {
		settings.SnapshotIntervalMinutes = defaults.SnapshotIntervalMinutes
		changed = true
	}
	if strings.TrimSpace(settings.DeviceID) == "" {
		settings.DeviceID = uuid.NewString()
		changed = true
	}
	if settings.SyncRetention < 1 || settings.SyncRetention > 50 {
		settings.SyncRetention = 10
		changed = true
	}
	if !settings.ReviewConflicts {
		// This migration intentionally favours data safety over legacy automatic
		// policies. New sync installations always require review.
		settings.ReviewConflicts = true
		changed = true
	}
	for i := range settings.Games {
		if normalizeBackupPolicy(&settings.Games[i].BackupPolicy) {
			changed = true
		}
		if strings.TrimSpace(settings.Games[i].SyncID) == "" {
			settings.Games[i].SyncID = uuid.NewString()
			changed = true
		}
	}

	return changed
}

// writes in-memory settings to disk and assumes caller holds manager lock.
func (m *Manager) saveLocked() error {
	payload, err := json.MarshalIndent(m.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(m.path, payload, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}
