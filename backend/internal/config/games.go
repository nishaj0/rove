package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/models"
)

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
