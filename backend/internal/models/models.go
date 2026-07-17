package models

import "time"

// defines strategies for resolving save conflicts during sync/import.
type ConflictPolicy string

// defines ui theme behavior as dark, light, or system-driven.
type ThemeMode string

const (
	ConflictLatestWins ConflictPolicy = "latest_wins"
	ConflictKeepBoth   ConflictPolicy = "keep_both"
	ThemeDark          ThemeMode      = "dark"
	ThemeLight         ThemeMode      = "light"
	ThemeSystem        ThemeMode      = "system"
)

// stores per-game rules for automatic local backups.
type BackupPolicy struct {
	Enabled          bool `json:"enabled"`
	IntervalMinutes  int  `json:"intervalMinutes"`
	ExitDelaySeconds int  `json:"exitDelaySeconds"`
	RetentionCount   int  `json:"retentionCount"`
}

// stores tracked title metadata and local filesystem paths.
type Game struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Path           string       `json:"path"`
	ExecutablePath string       `json:"executablePath"`
	CoverImageData string       `json:"coverImageData,omitempty"`
	BackupPolicy   BackupPolicy `json:"backupPolicy"`
	// SyncID identifies this title in a shared sync folder. It deliberately differs
	// from the local ID so the same game can be paired on another computer.
	SyncID string `json:"syncId,omitempty"`
}

// stores persisted runtime totals used by status views.
type GameStats struct {
	PlaytimeSeconds int64
	LastPlayedAt    time.Time
}

// stores persisted daemon configuration from settings.json.
type Settings struct {
	Port                    int            `json:"port"`
	SyncIntervalSeconds     int            `json:"syncIntervalSeconds"`
	ConflictPolicy          ConflictPolicy `json:"conflictPolicy"`
	ThemeMode               ThemeMode      `json:"themeMode"`
	Autostart               bool           `json:"autostart"`
	IgnorePatterns          []string       `json:"ignorePatterns"`
	SnapshotVaultPath       string         `json:"snapshotVaultPath"`
	SnapshotMaxPerGame      int            `json:"snapshotMaxPerGame"`
	SnapshotOnExit          bool           `json:"snapshotOnExit"`
	SnapshotPeriodic        bool           `json:"snapshotPeriodic"`
	SnapshotIntervalMinutes int            `json:"snapshotIntervalMinutes"`
	SyncRoot                string         `json:"syncRoot"`
	DeviceID                string         `json:"deviceId"`
	SyncRetention           int            `json:"syncRetention"`
	ReviewConflicts         bool           `json:"reviewConflicts"`
	Games                   []Game         `json:"games"`
}

// reports current engine state for api status responses.
type Status struct {
	Running        bool      `json:"running"`
	LastSyncTime   time.Time `json:"lastSyncTime"`
	LastError      string    `json:"lastError,omitempty"`
	Pending        int       `json:"pending"`
	SyncConfigured bool      `json:"syncConfigured"`
}

// SyncJob reports the lifecycle of one explicit or scheduled sync request.
type SyncJob struct {
	ID           string    `json:"id"`
	GameID       string    `json:"gameId,omitempty"`
	Reason       string    `json:"reason"`
	State        string    `json:"state"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	FinishedAt   time.Time `json:"finishedAt,omitempty"`
	ChangedFiles int       `json:"changedFiles"`
	Bytes        int64     `json:"bytes"`
	Error        string    `json:"error,omitempty"`
	ConflictIDs  []string  `json:"conflictIds,omitempty"`
}

// SyncConflict is a non-destructive collision requiring an explicit decision.
type SyncConflict struct {
	ID         string    `json:"id"`
	GameID     string    `json:"gameId"`
	Path       string    `json:"path"`
	LocalHash  string    `json:"localHash"`
	RemoteHash string    `json:"remoteHash"`
	CreatedAt  time.Time `json:"createdAt"`
	State      string    `json:"state"`
}

// stores a single sqlite-backed event returned by log endpoints.
type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Event     string    `json:"event"`
	Message   string    `json:"message"`
}

// carries game selection and destination for backup export.
type BackupExportRequest struct {
	GameID          string `json:"gameId"`
	DestinationPath string `json:"destinationPath"`
}

// carries game selection and source archive for backup import.
type BackupImportRequest struct {
	GameID     string `json:"gameId"`
	SourcePath string `json:"sourcePath"`
	Merge      bool   `json:"merge"`
}

// stores one local auto-backup record used by history endpoints.
type BackupHistoryEntry struct {
	ID        int64     `json:"id"`
	GameID    string    `json:"gameId"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
	Reason    string    `json:"reason"`
	Signature string    `json:"signature"`
	Status    string    `json:"status"`
}

// summarizes one versioned save snapshot stored in the local vault.
type SnapshotSummary struct {
	ID           string    `json:"id"`
	Seq          int       `json:"seq"`
	CreatedAt    time.Time `json:"createdAt"`
	Trigger      string    `json:"trigger"`
	ChangedFiles int       `json:"changedFiles"`
	TotalSize    int64     `json:"totalSize"`
}

// requests an immediate snapshot for a single tracked game.
type SnapshotCreateRequest struct {
	GameID string `json:"gameId"`
}

// requests restoring one previously stored snapshot for a game.
type SnapshotRestoreRequest struct {
	GameID     string `json:"gameId"`
	SnapshotID string `json:"snapshotId"`
}
