package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	"cloud-saver/backend/internal/models"
)

type Manager struct {
	mu      sync.Mutex
	watcher *fsnotify.Watcher
	cancel  context.CancelFunc
	onError func(error)
}

func New(onError func(error)) *Manager {
	return &Manager{onError: onError}
}

func (m *Manager) Reload(games []models.Game, onEvent func(path string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.watcher != nil {
		_ = m.watcher.Close()
		m.watcher = nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	m.watcher = w

	for _, game := range games {
		if err := addRecursive(w, game.Path); err != nil {
			_ = w.Close()
			m.watcher = nil
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) {
					_ = addRecursive(w, event.Name)
				}
				onEvent(event.Name)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				if err != nil && m.onError != nil {
					m.onError(fmt.Errorf("watcher event error: %w", err))
				}
			}
		}
	}()

	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if err := w.Add(path); err != nil {
				return nil
			}
		}
		return nil
	})
}
