package models

import (
	"encoding/json"
	"time"
)

// defines strategies for resolving save conflicts during sync/import.
type ConflictPolicy string

// defines ui theme behavior as dark, light, or system-driven.
type ThemeMode string

// CloudProvider selects the optional backing store used for sync revisions.
type CloudProvider string

// ProtectionTarget is the destination for durable save revisions. Local is
// always the first commit; remote targets replicate that committed revision.
type ProtectionTarget string

// ProtectionMode controls how save activity is checkpointed for one game.
type ProtectionMode string

// PauseScope distinguishes upload-only pauses from suspending capture work.
type PauseScope string

// ProviderState is a user-facing, actionable provider health state.
type ProviderState string

const (
	ConflictLatestWins       ConflictPolicy   = "latest_wins"
	ConflictKeepBoth         ConflictPolicy   = "keep_both"
	ThemeDark                ThemeMode        = "dark"
	ThemeLight               ThemeMode        = "light"
	ThemeSystem              ThemeMode        = "system"
	CloudProviderFolder      CloudProvider    = "folder"
	CloudProviderGoogleDrive CloudProvider    = "google_drive"
	ProtectionTargetLocal    ProtectionTarget = "local"
	ProtectionTargetFolder   ProtectionTarget = "folder"
	ProtectionTargetDrive    ProtectionTarget = "google_drive"
	ProtectionBalanced       ProtectionMode   = "balanced"
	ProtectionExitOnly       ProtectionMode   = "exit_only"
	ProtectionContinuous     ProtectionMode   = "continuous"
	ProtectionManual         ProtectionMode   = "manual"
	PauseNone                PauseScope       = "none"
	PauseUploads             PauseScope       = "uploads"
	PauseAll                 PauseScope       = "all"
	ProviderLocalOnly        ProviderState    = "local_only"
	ProviderOnline           ProviderState    = "online"
	ProviderOffline          ProviderState    = "offline"
	ProviderReconnecting     ProviderState    = "reconnecting"
	ProviderAuthRequired     ProviderState    = "authentication_required"
	ProviderRateLimited      ProviderState    = "rate_limited"
	ProviderQuotaFull        ProviderState    = "quota_full"
	ProviderUnavailable      ProviderState    = "provider_unavailable"
	ProviderRemoteProblem    ProviderState    = "remote_data_problem"
)

// SyncPolicy replaces the overlapping automatic backup and snapshot policies.
type SyncPolicy struct {
	Enabled                   bool           `json:"enabled"`
	Mode                      ProtectionMode `json:"mode"`
	CheckpointIntervalMinutes int            `json:"checkpointIntervalMinutes"`
	ExitProtection            bool           `json:"exitProtection"`
}

// ServiceControlState is persisted so a pause survives a daemon restart.
type ServiceControlState struct {
	ProtectionEnabled bool       `json:"protectionEnabled"`
	PauseScope        PauseScope `json:"pauseScope"`
	PauseUntil        time.Time  `json:"pauseUntil,omitempty"`
	PauseUntilLogin   bool       `json:"pauseUntilLogin,omitempty"`
	Reason            string     `json:"reason,omitempty"`
}

// ProviderHealth is the canonical state rendered by Home, Game, and tray UI.
type ProviderHealth struct {
	State         ProviderState `json:"state"`
	LastSuccessAt time.Time     `json:"lastSuccessAt,omitempty"`
	LastAttemptAt time.Time     `json:"lastAttemptAt,omitempty"`
	NextRetryAt   time.Time     `json:"nextRetryAt,omitempty"`
	ErrorCode     string        `json:"errorCode,omitempty"`
	Message       string        `json:"message,omitempty"`
	Action        string        `json:"action,omitempty"`
	PendingCount  int           `json:"pendingCount"`
	PendingBytes  int64         `json:"pendingBytes"`
}

// GoogleDriveSettings deliberately stores the refresh token only in encrypted
// form. API handlers must remove it before returning settings to the UI.
type GoogleDriveSettings struct {
	Connected             bool   `json:"connected"`
	AccountEmail          string `json:"accountEmail,omitempty"`
	EncryptedRefreshToken string `json:"encryptedRefreshToken,omitempty"`
	ReconnectRequired     bool   `json:"reconnectRequired,omitempty"`
	// FolderID is the Google Drive folder ID chosen by the user during setup.
	// When empty, Rove stores files directly under My Drive.
	FolderID   string `json:"folderId,omitempty"`
	FolderName string `json:"folderName,omitempty"`
}

// DriveFolder is a user-visible Google Drive folder returned by the folder
// picker API so the user can choose where Rove stores its data.
type DriveFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GameConfig is a writable per-game metadata file stored in the user's chosen
// Drive folder alongside the sync revision data. It carries human-readable
// display names, notes, and aggregated playtime totals.
type GameConfig struct {
	SchemaVersion    int       `json:"schemaVersion,omitempty"`
	SyncID           string    `json:"syncId"`
	DisplayName      string    `json:"displayName"`
	TotalPlaytimeSec int64     `json:"totalPlaytimeSec,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// RoveSettings is a portable snapshot of application-level preferences that
// can be exported from one installation and imported on another.
type RoveSettings struct {
	SchemaVersion   int      `json:"schemaVersion"`
	DeviceName      string   `json:"deviceName,omitempty"`
	IgnorePatterns  []string `json:"ignorePatterns,omitempty"`
	SyncIntervalSec int      `json:"syncIntervalSec,omitempty"`
	ExportedAt      string   `json:"exportedAt,omitempty"`
}

// stores per-game rules for automatic local backups.
type BackupPolicy struct {
	Enabled          bool `json:"enabled"`
	IntervalMinutes  int  `json:"intervalMinutes"`
	ExitDelaySeconds int  `json:"exitDelaySeconds"`
	RetentionCount   int  `json:"retentionCount"`
}

// stores tracked title metadata and local filesystem paths.
type Game struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	ExecutablePath string `json:"executablePath"`
	CoverImageData string `json:"coverImageData,omitempty"`
	// Version is optional user-facing metadata kept with the portable profile.
	Version string `json:"version,omitempty"`
	// CoverSource describes whether the image was chosen by the user, read from
	// the executable, or is the deterministic generated fallback.
	CoverSource  string       `json:"coverSource,omitempty"`
	BackupPolicy BackupPolicy `json:"backupPolicy"`
	// SyncID identifies this title in a shared sync folder. It deliberately differs
	// from the local ID so the same game can be paired on another computer.
	SyncID           string     `json:"syncId,omitempty"`
	ProtectionPolicy SyncPolicy `json:"protectionPolicy"`
}

// stores persisted runtime totals used by status views.
type GameStats struct {
	PlaytimeSeconds int64
	LastPlayedAt    time.Time
}

// stores persisted daemon configuration from settings.json.
type Settings struct {
	SchemaVersion           int                 `json:"schemaVersion"`
	Port                    int                 `json:"port"`
	SyncIntervalSeconds     int                 `json:"syncIntervalSeconds"`
	ConflictPolicy          ConflictPolicy      `json:"conflictPolicy"`
	ThemeMode               ThemeMode           `json:"themeMode"`
	Autostart               bool                `json:"autostart"`
	IgnorePatterns          []string            `json:"ignorePatterns"`
	SnapshotVaultPath       string              `json:"snapshotVaultPath"`
	SnapshotMaxPerGame      int                 `json:"snapshotMaxPerGame"`
	SnapshotOnExit          bool                `json:"snapshotOnExit"`
	SnapshotPeriodic        bool                `json:"snapshotPeriodic"`
	SnapshotIntervalMinutes int                 `json:"snapshotIntervalMinutes"`
	SyncRoot                string              `json:"syncRoot"`
	DeviceID                string              `json:"deviceId"`
	SyncRetention           int                 `json:"syncRetention"`
	ReviewConflicts         bool                `json:"reviewConflicts"`
	CloudProvider           CloudProvider       `json:"cloudProvider"`
	GoogleDrive             GoogleDriveSettings `json:"googleDrive"`
	ProtectionTarget        ProtectionTarget    `json:"protectionTarget"`
	ServiceControl          ServiceControlState `json:"serviceControl"`
	AutomaticUpload         bool                `json:"automaticUpload"`
	UploadOnMetered         bool                `json:"uploadOnMetered"`
	RemoteCheckBeforeLaunch bool                `json:"remoteCheckBeforeLaunch"`
	ImportantNotifications  bool                `json:"importantNotifications"`
	DebugLogging            bool                `json:"debugLogging"`
	AutoRevisionRetention   int                 `json:"autoRevisionRetention"`
	ProtectionDefaultsReady bool                `json:"protectionDefaultsReady"`
	Games                   []Game              `json:"games"`
}

// SettingsUpdate is the user-editable settings contract. Runtime/provider
// status and credentials are intentionally excluded from mutation. The raw
// legacy fields keep a briefly version-skewed desktop client compatible while
// ensuring those values are ignored instead of being persisted back.
type SettingsUpdate struct {
	Port                    *int                 `json:"port,omitempty"`
	SyncIntervalSeconds     *int                 `json:"syncIntervalSeconds,omitempty"`
	ConflictPolicy          *ConflictPolicy      `json:"conflictPolicy,omitempty"`
	ThemeMode               *ThemeMode           `json:"themeMode,omitempty"`
	Autostart               *bool                `json:"autostart,omitempty"`
	IgnorePatterns          *[]string            `json:"ignorePatterns,omitempty"`
	SnapshotVaultPath       *string              `json:"snapshotVaultPath,omitempty"`
	SnapshotMaxPerGame      *int                 `json:"snapshotMaxPerGame,omitempty"`
	SnapshotOnExit          *bool                `json:"snapshotOnExit,omitempty"`
	SnapshotPeriodic        *bool                `json:"snapshotPeriodic,omitempty"`
	SnapshotIntervalMinutes *int                 `json:"snapshotIntervalMinutes,omitempty"`
	SyncRoot                *string              `json:"syncRoot,omitempty"`
	SyncRetention           *int                 `json:"syncRetention,omitempty"`
	ReviewConflicts         *bool                `json:"reviewConflicts,omitempty"`
	ProtectionTarget        *ProtectionTarget    `json:"protectionTarget,omitempty"`
	ServiceControl          *ServiceControlState `json:"serviceControl,omitempty"`
	AutomaticUpload         *bool                `json:"automaticUpload,omitempty"`
	UploadOnMetered         *bool                `json:"uploadOnMetered,omitempty"`
	RemoteCheckBeforeLaunch *bool                `json:"remoteCheckBeforeLaunch,omitempty"`
	ImportantNotifications  *bool                `json:"importantNotifications,omitempty"`
	DebugLogging            *bool                `json:"debugLogging,omitempty"`
	AutoRevisionRetention   *int                 `json:"autoRevisionRetention,omitempty"`

	LegacySchemaVersion           json.RawMessage `json:"schemaVersion,omitempty"`
	LegacyDeviceID                json.RawMessage `json:"deviceId,omitempty"`
	LegacyCloudProvider           json.RawMessage `json:"cloudProvider,omitempty"`
	LegacyGoogleDrive             json.RawMessage `json:"googleDrive,omitempty"`
	LegacyProtectionDefaultsReady json.RawMessage `json:"protectionDefaultsReady,omitempty"`
	LegacyGames                   json.RawMessage `json:"games,omitempty"`
}

// Apply overlays editable values onto trusted persisted state. In particular,
// OAuth connection state, device identity, games, and schema metadata always
// come from the daemon rather than the desktop payload.
func (update SettingsUpdate) Apply(current Settings) Settings {
	if update.Port != nil {
		current.Port = *update.Port
	}
	if update.SyncIntervalSeconds != nil {
		current.SyncIntervalSeconds = *update.SyncIntervalSeconds
	}
	if update.ConflictPolicy != nil {
		current.ConflictPolicy = *update.ConflictPolicy
	}
	if update.ThemeMode != nil {
		current.ThemeMode = *update.ThemeMode
	}
	if update.Autostart != nil {
		current.Autostart = *update.Autostart
	}
	if update.IgnorePatterns != nil {
		current.IgnorePatterns = append([]string{}, (*update.IgnorePatterns)...)
	}
	if update.SnapshotVaultPath != nil {
		current.SnapshotVaultPath = *update.SnapshotVaultPath
	}
	if update.SnapshotMaxPerGame != nil {
		current.SnapshotMaxPerGame = *update.SnapshotMaxPerGame
	}
	if update.SnapshotOnExit != nil {
		current.SnapshotOnExit = *update.SnapshotOnExit
	}
	if update.SnapshotPeriodic != nil {
		current.SnapshotPeriodic = *update.SnapshotPeriodic
	}
	if update.SnapshotIntervalMinutes != nil {
		current.SnapshotIntervalMinutes = *update.SnapshotIntervalMinutes
	}
	if update.SyncRoot != nil {
		current.SyncRoot = *update.SyncRoot
	}
	if update.SyncRetention != nil {
		current.SyncRetention = *update.SyncRetention
	}
	if update.ReviewConflicts != nil {
		current.ReviewConflicts = *update.ReviewConflicts
	}
	if update.ProtectionTarget != nil {
		current.ProtectionTarget = *update.ProtectionTarget
	}
	if update.ServiceControl != nil {
		current.ServiceControl = *update.ServiceControl
	}
	if update.AutomaticUpload != nil {
		current.AutomaticUpload = *update.AutomaticUpload
	}
	if update.UploadOnMetered != nil {
		current.UploadOnMetered = *update.UploadOnMetered
	}
	if update.RemoteCheckBeforeLaunch != nil {
		current.RemoteCheckBeforeLaunch = *update.RemoteCheckBeforeLaunch
	}
	if update.ImportantNotifications != nil {
		current.ImportantNotifications = *update.ImportantNotifications
	}
	if update.DebugLogging != nil {
		current.DebugLogging = *update.DebugLogging
	}
	if update.AutoRevisionRetention != nil {
		current.AutoRevisionRetention = *update.AutoRevisionRetention
	}
	return current
}

// reports current engine state for api status responses.
type Status struct {
	Running             bool                `json:"running"`
	ProtectionTarget    ProtectionTarget    `json:"protectionTarget"`
	LastSyncTime        time.Time           `json:"lastSyncTime"`
	LastError           string              `json:"lastError,omitempty"`
	Pending             int                 `json:"pending"`
	SyncConfigured      bool                `json:"syncConfigured"`
	Provider            ProviderHealth      `json:"provider"`
	ServiceControl      ServiceControlState `json:"serviceControl"`
	LastLocalProtection time.Time           `json:"lastLocalProtection,omitempty"`
	LastCloudUpload     time.Time           `json:"lastCloudUpload,omitempty"`
	PendingBytes        int64               `json:"pendingBytes"`
	ConflictCount       int                 `json:"conflictCount"`
	ActiveJobID         string              `json:"activeJobId,omitempty"`
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
	Stage        string    `json:"stage,omitempty"`
	Progress     int       `json:"progress"`
	RetryCount   int       `json:"retryCount"`
	NextRetryAt  time.Time `json:"nextRetryAt,omitempty"`
	ErrorCode    string    `json:"errorCode,omitempty"`
	Priority     int       `json:"priority"`
	RevisionID   string    `json:"revisionId,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

// Revision is a durable local protection point.
type Revision struct {
	ID           string    `json:"id"`
	GameID       string    `json:"gameId"`
	SyncID       string    `json:"syncId"`
	ParentIDs    []string  `json:"parentIds,omitempty"`
	Kind         string    `json:"kind"`
	Name         string    `json:"name,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	FileCount    int       `json:"fileCount"`
	TotalBytes   int64     `json:"totalBytes"`
	ManifestJSON string    `json:"-"`
	Pinned       bool      `json:"pinned"`
	Uploaded     bool      `json:"uploaded"`
}

// RecoveryPoint is a named, pinned revision retained until explicit deletion.
type RecoveryPoint struct {
	ID         string    `json:"id"`
	GameID     string    `json:"gameId"`
	RevisionID string    `json:"revisionId"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	FileCount  int       `json:"fileCount"`
	TotalBytes int64     `json:"totalBytes"`
}

// StorageUsage separates unique local storage from reclaimable history.
type StorageUsage struct {
	GameID           string `json:"gameId,omitempty"`
	UniqueBytes      int64  `json:"uniqueBytes"`
	AutomaticCount   int    `json:"automaticCount"`
	RecoveryPoints   int    `json:"recoveryPoints"`
	PendingBytes     int64  `json:"pendingBytes"`
	ReclaimableBytes int64  `json:"reclaimableBytes"`
}

// StorageSummary is the whole-library storage view. CloudBytes represents only
// Rove-managed provider content, never the user's unrelated account files.
type StorageSummary struct {
	LocalBytes       int64 `json:"localBytes"`
	CloudBytes       int64 `json:"cloudBytes"`
	PendingBytes     int64 `json:"pendingBytes"`
	ReclaimableBytes int64 `json:"reclaimableBytes"`
	GameCount        int   `json:"gameCount"`
	CloudAvailable   bool  `json:"cloudAvailable"`
}

// ActivityEvent is the useful, sanitized history shown to end users.
type ActivityEvent struct {
	ID               int64     `json:"id"`
	Category         string    `json:"category"`
	Severity         string    `json:"severity"`
	GameID           string    `json:"gameId,omitempty"`
	Title            string    `json:"title"`
	Detail           string    `json:"detail"`
	Timestamp        time.Time `json:"timestamp"`
	ActionTarget     string    `json:"actionTarget,omitempty"`
	DeduplicationKey string    `json:"-"`
	Resolved         bool      `json:"resolved"`
}

// GameProtectionStatus is the per-game sync-health view.
type GameProtectionStatus struct {
	GameID              string              `json:"gameId"`
	Policy              SyncPolicy          `json:"policy"`
	ServiceControl      ServiceControlState `json:"serviceControl"`
	DirtyFiles          int                 `json:"dirtyFiles"`
	LastLocalProtection time.Time           `json:"lastLocalProtection,omitempty"`
	LastCloudUpload     time.Time           `json:"lastCloudUpload,omitempty"`
	PendingCount        int                 `json:"pendingCount"`
	PendingBytes        int64               `json:"pendingBytes"`
	ConflictCount       int                 `json:"conflictCount"`
	ActiveJob           *SyncJob            `json:"activeJob,omitempty"`
}

// GameProfile is portable provider metadata used by fresh-install recovery.
type GameProfile struct {
	SchemaVersion   int       `json:"schemaVersion,omitempty"`
	SyncID          string    `json:"syncId"`
	Name            string    `json:"name"`
	Version         string    `json:"version,omitempty"`
	CoverSource     string    `json:"coverSource,omitempty"`
	CoverAvailable  bool      `json:"coverAvailable,omitempty"`
	PlaytimeSeconds int64     `json:"playtimeSeconds,omitempty"`
	ExecutableName  string    `json:"executableName,omitempty"`
	PathHint        string    `json:"pathHint,omitempty"`
	LatestRevision  string    `json:"latestRevision,omitempty"`
	LatestProtected time.Time `json:"latestProtected,omitempty"`
	FileCount       int       `json:"fileCount"`
	TotalBytes      int64     `json:"totalBytes"`
}

// PlaytimeLedger records the cumulative time contributed by one installation.
// Files are independent per device so concurrent play sessions cannot overwrite
// or double-count another device's total.
type PlaytimeLedger struct {
	SchemaVersion int       `json:"schemaVersion"`
	DeviceID      string    `json:"deviceId"`
	Seconds       int64     `json:"seconds"`
	UpdatedAt     time.Time `json:"updatedAt"`
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
	Confirmed  bool   `json:"confirmed"`
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
