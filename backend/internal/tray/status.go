package tray

import "time"

// holds runtime values shown in tray status rows.
type Status struct {
	PlayingGames []string
	State        string
	Provider     string
	PendingCount int
	LastLocal    time.Time
	LastCloud    time.Time
	Paused       bool
	Offline      bool
	HasConflicts bool
}

type Actions struct {
	OpenUI        func()
	ProtectNow    func()
	RecoveryPoint func()
	Retry         func()
	PauseUploads  func()
	PauseAll      func()
	PauseGame     func()
	Resume        func()
	Review        func()
	Exit          func()
}
