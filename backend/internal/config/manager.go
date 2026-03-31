package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cloud-saver/backend/internal/models"
)

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

// resolves the per-user settings.json location under CloudSaver.
func defaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "CloudSaver", "settings.json"), nil
}

// returns baseline values used for first run and normalization fallback.
func DefaultSettings() models.Settings {
	return models.Settings{
		Port:                7331,
		SyncIntervalSeconds: 30,
		ConflictPolicy:      models.ConflictLatestWins,
		ThemeMode:           models.ThemeDark,
		Autostart:           false,
		IgnorePatterns:      []string{"*.tmp", "*.log", "cache"},
		Games:               []models.Game{},
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

	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.Port = next.Port
	m.settings.SyncIntervalSeconds = next.SyncIntervalSeconds
	m.settings.ConflictPolicy = next.ConflictPolicy
	m.settings.ThemeMode = next.ThemeMode
	m.settings.Autostart = next.Autostart
	m.settings.IgnorePatterns = append([]string{}, next.IgnorePatterns...)
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
