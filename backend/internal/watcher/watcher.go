package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
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
			if m.onError != nil {
				m.onError(fmt.Errorf("watch setup warning for %s: %w", game.Path, err))
			}
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
					if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
						if err := addRecursive(w, event.Name); err != nil && m.onError != nil {
							m.onError(fmt.Errorf("watch recursive add failed for %s: %w", event.Name, err))
						}
					} else if statErr != nil && !os.IsNotExist(statErr) && m.onError != nil {
						m.onError(fmt.Errorf("watch stat failed for %s: %w", event.Name, statErr))
					}
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
	var firstErr error
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("walk dir %s: %w", path, err)
			}
			return nil
		}
		if d.IsDir() {
			if err := w.Add(path); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("add watch %s: %w", path, err)
				}
				return nil
			}
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	return firstErr
}
