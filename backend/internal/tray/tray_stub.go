//go:build !windows

package tray

import "context"

type Manager struct{}

func New() *Manager { return &Manager{} }

func (m *Manager) SetRunning(_ bool) {}

func (m *Manager) Run(_ func(), _ func(), _ func()) {
	<-context.Background().Done()
}
