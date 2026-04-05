package tray

// holds runtime values shown in tray status rows.
type Status struct {
	SyncEnabled  bool
	Pending      int
	PlayingGames []string
}
