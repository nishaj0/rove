package models

import "time"

type ConflictPolicy string
type ThemeMode string

const (
	ConflictLatestWins ConflictPolicy = "latest_wins"
	ConflictKeepBoth   ConflictPolicy = "keep_both"
	ThemeDark          ThemeMode      = "dark"
	ThemeLight         ThemeMode      = "light"
	ThemeSystem        ThemeMode      = "system"
)

type Game struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	ExecutablePath string `json:"executablePath"`
	CoverImageData string `json:"coverImageData,omitempty"`
}

type GameStats struct {
	PlaytimeSeconds int64
	LastPlayedAt    time.Time
}

type Settings struct {
	Port                int            `json:"port"`
	SyncIntervalSeconds int            `json:"syncIntervalSeconds"`
	ConflictPolicy      ConflictPolicy `json:"conflictPolicy"`
	ThemeMode           ThemeMode      `json:"themeMode"`
	Autostart           bool           `json:"autostart"`
	IgnorePatterns      []string       `json:"ignorePatterns"`
	Games               []Game         `json:"games"`
}

type Status struct {
	Running      bool      `json:"running"`
	LastSyncTime time.Time `json:"lastSyncTime"`
	LastError    string    `json:"lastError,omitempty"`
	Pending      int       `json:"pending"`
}

type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Event     string    `json:"event"`
	Message   string    `json:"message"`
}

type BackupExportRequest struct {
	GameID          string `json:"gameId"`
	DestinationPath string `json:"destinationPath"`
}

type BackupImportRequest struct {
	GameID     string `json:"gameId"`
	SourcePath string `json:"sourcePath"`
	Merge      bool   `json:"merge"`
}
