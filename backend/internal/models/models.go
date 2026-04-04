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
}

// stores persisted runtime totals used by status views.
type GameStats struct {
	PlaytimeSeconds int64
	LastPlayedAt    time.Time
}

// stores persisted daemon configuration from settings.json.
type Settings struct {
	Port                int            `json:"port"`
	SyncIntervalSeconds int            `json:"syncIntervalSeconds"`
	ConflictPolicy      ConflictPolicy `json:"conflictPolicy"`
	ThemeMode           ThemeMode      `json:"themeMode"`
	Autostart           bool           `json:"autostart"`
	IgnorePatterns      []string       `json:"ignorePatterns"`
	Games               []Game         `json:"games"`
}

// reports current engine state for api status responses.
type Status struct {
	Running      bool      `json:"running"`
	LastSyncTime time.Time `json:"lastSyncTime"`
	LastError    string    `json:"lastError,omitempty"`
	Pending      int       `json:"pending"`
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
