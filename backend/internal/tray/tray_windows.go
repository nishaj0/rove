//go:build windows

package tray

import (
	"fmt"
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

func (m *Manager) Run(onOpenUI, onToggleSync, onExit func()) {
	systray.Run(func() {
		systray.SetTitle(branding.DisplayName)
		systray.SetTooltip(branding.DisplayName)

		openItem := systray.AddMenuItem("Open UI", "Open settings UI")
		toggleItem := systray.AddMenuItem("Stop Sync", "Start or stop syncing")
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit", fmt.Sprintf("Exit %s", branding.DisplayName))

		go func() {
			for {
				select {
				case <-openItem.ClickedCh:
					onOpenUI()
				case <-toggleItem.ClickedCh:
					onToggleSync()
					if m.running.Load() {
						toggleItem.SetTitle("Stop Sync")
					} else {
						toggleItem.SetTitle("Start Sync")
					}
				case <-exitItem.ClickedCh:
					onExit()
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
}
