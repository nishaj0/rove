package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/models"
)

func (m *Manager) AddGame(name, path, executablePath, coverImageData string) (models.Game, error) {
	return m.AddGameWithPolicy(name, path, executablePath, coverImageData, nil)
}

// adds a tracked game and optionally overrides default backup policy values.
func (m *Manager) AddGameWithPolicy(name, path, executablePath, coverImageData string, policy *models.BackupPolicy) (models.Game, error) {
	if name == "" || path == "" || executablePath == "" {
		return models.Game{}, errors.New("name, path, and executablePath are required")
	}
	if _, err := os.Stat(path); err != nil {
		return models.Game{}, fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(executablePath); err != nil {
		return models.Game{}, fmt.Errorf("invalid executablePath: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.settings.Games {
		if filepath.Clean(g.Path) == filepath.Clean(path) {
			return models.Game{}, errors.New("path already tracked")
		}
	}

	backupPolicy := defaultBackupPolicy()
	if policy != nil {
		backupPolicy = *policy
		if err := validateBackupPolicy(backupPolicy); err != nil {
			return models.Game{}, err
		}
	}

	game := models.Game{
		ID:               uuid.NewString(),
		Name:             name,
		Path:             path,
		ExecutablePath:   executablePath,
		CoverImageData:   coverImageData,
		BackupPolicy:     backupPolicy,
		SyncID:           uuid.NewString(),
		ProtectionPolicy: defaultSyncPolicy(),
	}
	m.settings.Games = append(m.settings.Games, game)
	if err := m.saveLocked(); err != nil {
		return models.Game{}, err
	}
	return game, nil
}

// UpdateGameSyncPolicy updates the unified protection policy without touching
// legacy backup metadata, which remains available for read-only migration.
func (m *Manager) UpdateGameSyncPolicy(id string, policy models.SyncPolicy) (models.Game, error) {
	if err := validateSyncPolicy(policy); err != nil {
		return models.Game{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.settings.Games {
		if m.settings.Games[i].ID != id {
			continue
		}
		m.settings.Games[i].ProtectionPolicy = policy
		if err := m.saveLocked(); err != nil {
			return models.Game{}, err
		}
		return m.settings.Games[i], nil
	}
	return models.Game{}, errors.New("game not found")
}

func (m *Manager) UpdateGame(id string, name string, path string, executablePath string, coverImageData string) (models.Game, error) {
	return m.UpdateGameWithPolicy(id, name, path, executablePath, coverImageData, nil)
}

// updates game identity/path fields and can replace backup policy when provided.
func (m *Manager) UpdateGameWithPolicy(id string, name string, path string, executablePath string, coverImageData string, policy *models.BackupPolicy) (models.Game, error) {
	if id == "" || name == "" || path == "" || executablePath == "" {
		return models.Game{}, errors.New("id, name, path, and executablePath are required")
	}
	if _, err := os.Stat(path); err != nil {
		return models.Game{}, fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(executablePath); err != nil {
		return models.Game{}, fmt.Errorf("invalid executablePath: %w", err)
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
	m.settings.Games[idx].ExecutablePath = executablePath
	m.settings.Games[idx].CoverImageData = coverImageData
	if policy != nil {
		if err := validateBackupPolicy(*policy); err != nil {
			return models.Game{}, err
		}
		m.settings.Games[idx].BackupPolicy = *policy
	}
	if err := m.saveLocked(); err != nil {
		return models.Game{}, err
	}
	return m.settings.Games[idx], nil
}

// updates only per-game backup policy while preserving other game fields.
func (m *Manager) UpdateGameBackupPolicy(id string, policy models.BackupPolicy) (models.Game, error) {
	if id == "" {
		return models.Game{}, errors.New("id is required")
	}
	if err := validateBackupPolicy(policy); err != nil {
		return models.Game{}, err
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
		return models.Game{}, errors.New("game not found")
	}

	m.settings.Games[idx].BackupPolicy = policy
	if err := m.saveLocked(); err != nil {
		return models.Game{}, err
	}
	return m.settings.Games[idx], nil
}

// SetGameSyncID pairs a local game with an existing shared-folder game.
func (m *Manager) SetGameSyncID(id, syncID string) (models.Game, error) {
	if id == "" || strings.TrimSpace(syncID) == "" {
		return models.Game{}, errors.New("id and syncId are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.settings.Games {
		if m.settings.Games[i].ID != id {
			continue
		}
		m.settings.Games[i].SyncID = strings.TrimSpace(syncID)
		if err := m.saveLocked(); err != nil {
			return models.Game{}, err
		}
		return m.settings.Games[i], nil
	}
	return models.Game{}, errors.New("game not found")
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
