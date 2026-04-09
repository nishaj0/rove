//go:build windows

package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"cloud-saver/backend/internal/branding"

	"github.com/getlantern/systray"
)

type Manager struct {
	running atomic.Bool
}

func New() *Manager {
	m := &Manager{}
	m.running.Store(true)
	return m
}

func (m *Manager) SetRunning(v bool) {
	m.running.Store(v)
}

func (m *Manager) Run(statusUpdates <-chan Status, onOpenUI, onToggleSync, onSyncNow, onExit func()) {
	systray.Run(func() {
		systray.SetTitle(branding.DisplayName)
		if iconBytes := loadTrayIcon(); len(iconBytes) > 0 {
			systray.SetIcon(iconBytes)
		}
		systray.SetTooltip(branding.DisplayName)

		openItem := systray.AddMenuItem("Open App", "Open desktop app")
		toggleItem := systray.AddMenuItem("Stop Sync", "Start or stop syncing")
		syncNowItem := systray.AddMenuItem("Sync Now", "Trigger manual sync")
		systray.AddSeparator()
		playingItem := systray.AddMenuItem("Playing: none", "Tracked games currently running")
		playingItem.Disable()
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit", fmt.Sprintf("Exit %s", branding.DisplayName))

		currentStatus := Status{PlayingGames: nil}
		renderStatus := func(s Status) {
			playingSummary := "none"
			if len(s.PlayingGames) > 0 {
				if len(s.PlayingGames) <= 2 {
					playingSummary = strings.Join(s.PlayingGames, ", ")
				} else {
					playingSummary = fmt.Sprintf("%s +%d", strings.Join(s.PlayingGames[:2], ", "), len(s.PlayingGames)-2)
				}
			}

			playingItem.SetTitle(fmt.Sprintf("Playing: %s", playingSummary))
			systray.SetTooltip(fmt.Sprintf("%s | playing: %s", branding.DisplayName, playingSummary))
		}
		renderStatus(currentStatus)

		go func() {
			for {
				select {
				case <-openItem.ClickedCh:
					onOpenUI()
				case <-systray.TrayDoubleClickedCh:
					onOpenUI()
				case <-toggleItem.ClickedCh:
					onToggleSync()
					if m.running.Load() {
						toggleItem.SetTitle("Stop Sync")
					} else {
						toggleItem.SetTitle("Start Sync")
					}
					renderStatus(currentStatus)
				case <-syncNowItem.ClickedCh:
					onSyncNow()
				case update, ok := <-statusUpdates:
					if !ok {
						continue
					}
					currentStatus = update
					renderStatus(currentStatus)
				case <-exitItem.ClickedCh:
					onExit()
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
}

func loadTrayIcon() []byte {
	candidates := []string{
		filepath.Join("..", "app", "src-tauri", "icons", "icon.ico"),
		filepath.Join("app", "src-tauri", "icons", "icon.ico"),
		"icon.ico",
		filepath.Join("icons", "icon.ico"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "icon.ico"),
			filepath.Join(exeDir, "icons", "icon.ico"),
			filepath.Join(exeDir, "..", "app", "src-tauri", "icons", "icon.ico"),
			filepath.Join(exeDir, "..", "..", "app", "src-tauri", "icons", "icon.ico"),
		)
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, path := range candidates {
		normalized := filepath.Clean(path)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		if iconBytes, err := os.ReadFile(normalized); err == nil && len(iconBytes) > 0 {
			return iconBytes
		}
	}

	return nil
}
