package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/models"
)

type Manager struct {
	mu       sync.RWMutex
	path     string
	settings models.Settings
}

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

func defaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "CloudSaver", "settings.json"), nil
}

func DefaultSettings() models.Settings {
	return models.Settings{
		Port:                7331,
		SyncIntervalSeconds: 30,
		ConflictPolicy:      models.ConflictLatestWins,
		Autostart:           false,
		IgnorePatterns:      []string{"*.tmp", "*.log", "cache"},
		Games:               []models.Game{},
	}
}

func (m *Manager) Path() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.path
}

func (m *Manager) Get() models.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cloned := m.settings
	cloned.Games = append([]models.Game(nil), m.settings.Games...)
	cloned.IgnorePatterns = append([]string(nil), m.settings.IgnorePatterns...)
	return cloned
}

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

	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.Port = next.Port
	m.settings.SyncIntervalSeconds = next.SyncIntervalSeconds
	m.settings.ConflictPolicy = next.ConflictPolicy
	m.settings.Autostart = next.Autostart
	m.settings.IgnorePatterns = append([]string(nil), next.IgnorePatterns...)
	return m.saveLocked()
}

func (m *Manager) AddGame(name, path string) (models.Game, error) {
	if name == "" || path == "" {
		return models.Game{}, errors.New("name and path are required")
	}
	if _, err := os.Stat(path); err != nil {
		return models.Game{}, fmt.Errorf("invalid path: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.settings.Games {
		if filepath.Clean(g.Path) == filepath.Clean(path) {
			return models.Game{}, errors.New("path already tracked")
		}
	}

	game := models.Game{
		ID:   uuid.NewString(),
		Name: name,
		Path: path,
	}
	m.settings.Games = append(m.settings.Games, game)
	if err := m.saveLocked(); err != nil {
		return models.Game{}, err
	}
	return game, nil
}

func (m *Manager) UpdateGame(id string, name string, path string) (models.Game, error) {
	if id == "" || name == "" || path == "" {
		return models.Game{}, errors.New("id, name, and path are required")
	}
	if _, err := os.Stat(path); err != nil {
		return models.Game{}, fmt.Errorf("invalid path: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, g := range m.settings.Games {
		if g.ID == id {
			idx = i
			continue
		}
		if filepath.Clean(g.Path) == filepath.Clean(path) {
			return models.Game{}, errors.New("path already tracked by another game")
		}
	}
	if idx < 0 {
		return models.Game{}, errors.New("game not found")
	}

	m.settings.Games[idx].Name = name
	m.settings.Games[idx].Path = path
	if err := m.saveLocked(); err != nil {
		return models.Game{}, err
	}
	return m.settings.Games[idx], nil
}

func (m *Manager) DeleteGame(id string) error {
	if id == "" {
		return errors.New("id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, g := range m.settings.Games {
		if g.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("game not found")
	}

	m.settings.Games = append(m.settings.Games[:idx], m.settings.Games[idx+1:]...)
	return m.saveLocked()
}

func (m *Manager) FindGame(id string) (models.Game, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, g := range m.settings.Games {
		if g.ID == id {
			return g, true
		}
	}
	return models.Game{}, false
}

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

	if err := json.Unmarshal(payload, &m.settings); err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}
	return nil
}

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
