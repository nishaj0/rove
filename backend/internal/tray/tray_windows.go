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

func (m *Manager) Quit() { systray.Quit() }

func (m *Manager) Run(statusUpdates <-chan Status, actions Actions) {
	systray.Run(func() {
		systray.SetTitle(branding.DisplayName)
		if iconBytes := loadTrayIcon(); len(iconBytes) > 0 {
			systray.SetIcon(iconBytes)
		}
		systray.SetTooltip(branding.DisplayName)

		openItem := systray.AddMenuItem("Open Rove", "Open desktop app")
		stateItem := systray.AddMenuItem("State: starting", "Current protection state")
		stateItem.Disable()
		providerItem := systray.AddMenuItem("Provider: checking", "Cloud provider and pending uploads")
		providerItem.Disable()
		protectNowItem := systray.AddMenuItem("Protect now", "Protect pending save changes")
		recoveryPointItem := systray.AddMenuItem("Create Recovery Point", "Pin the current save for a running game")
		retryItem := systray.AddMenuItem("Retry connection", "Retry pending provider work")
		reviewItem := systray.AddMenuItem("Review conflicts", "Open conflicts that need a decision")
		systray.AddSeparator()
		playingItem := systray.AddMenuItem("Playing: none", "Tracked games currently running")
		playingItem.Disable()
		systray.AddSeparator()
		pauseUploadsItem := systray.AddMenuItem("Pause cloud uploads", "Keep local protection active")
		pauseAllItem := systray.AddMenuItem("Pause all protection", "Suspend capture and provider work")
		pauseGameItem := systray.AddMenuItem("Pause this game", "Pause the only tracked game currently running")
		resumeItem := systray.AddMenuItem("Resume protection", "Resume paused protection work")
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Quit Rove", fmt.Sprintf("Exit %s", branding.DisplayName))

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
			if len(s.PlayingGames) == 0 {
				recoveryPointItem.Disable()
				pauseGameItem.Disable()
			} else {
				recoveryPointItem.Enable()
				if len(s.PlayingGames) == 1 {
					pauseGameItem.Enable()
				} else {
					pauseGameItem.Disable()
				}
			}
			state := s.State
			if state == "" {
				state = "Protected"
			}
			stateItem.SetTitle("State: " + state)
			provider := s.Provider
			if provider == "" {
				provider = "Local only"
			}
			providerItem.SetTitle(fmt.Sprintf("Provider: %s | pending: %d", provider, s.PendingCount))
			if s.Offline {
				retryItem.Enable()
			} else {
				retryItem.Disable()
			}
			if s.HasConflicts {
				reviewItem.Enable()
			} else {
				reviewItem.Disable()
			}
			if s.Paused {
				resumeItem.Enable()
				pauseUploadsItem.Disable()
				pauseAllItem.Disable()
			} else {
				resumeItem.Disable()
				pauseUploadsItem.Enable()
				pauseAllItem.Enable()
			}
			protected := "never"
			if !s.LastCloud.IsZero() {
				protected = s.LastCloud.Local().Format("15:04")
			} else if !s.LastLocal.IsZero() {
				protected = s.LastLocal.Local().Format("15:04") + " local"
			}
			systray.SetTooltip(fmt.Sprintf("%s | %s | playing: %s | last protected: %s | pending: %d", branding.DisplayName, state, playingSummary, protected, s.PendingCount))
		}
		renderStatus(currentStatus)

		go func() {
			for {
				select {
				case <-openItem.ClickedCh:
					actions.OpenUI()
				case <-systray.TrayDoubleClickedCh:
					actions.OpenUI()
				case <-protectNowItem.ClickedCh:
					actions.ProtectNow()
				case <-recoveryPointItem.ClickedCh:
					actions.RecoveryPoint()
				case <-retryItem.ClickedCh:
					actions.Retry()
				case <-reviewItem.ClickedCh:
					actions.Review()
				case <-pauseUploadsItem.ClickedCh:
					actions.PauseUploads()
				case <-pauseAllItem.ClickedCh:
					actions.PauseAll()
				case <-pauseGameItem.ClickedCh:
					actions.PauseGame()
				case <-resumeItem.ClickedCh:
					actions.Resume()
				case update, ok := <-statusUpdates:
					if !ok {
						continue
					}
					currentStatus = update
					renderStatus(currentStatus)
				case <-exitItem.ClickedCh:
					actions.Exit()
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
