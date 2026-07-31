package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/branding"
	"cloud-saver/backend/internal/models"
	runtimecfg "cloud-saver/backend/internal/runtime"
)

const (
	defaultBackupIntervalMinutes  = 10
	defaultBackupExitDelaySeconds = 10
	defaultBackupRetentionCount   = 5
	defaultSnapshotMaxPerGame     = 5
	defaultSnapshotIntervalMins   = 15
	minBackupIntervalMinutes      = 3
	maxBackupIntervalMinutes      = 120
	minBackupExitDelaySeconds     = 3
	maxBackupExitDelaySeconds     = 120
	minBackupRetentionCount       = 1
	maxBackupRetentionCount       = 30
	minSnapshotMaxPerGame         = 1
	maxSnapshotMaxPerGame         = 50
	minSnapshotIntervalMins       = 1
	maxSnapshotIntervalMins       = 1440
	defaultCheckpointMinutes      = 10
	minCheckpointMinutes          = 2
	maxCheckpointMinutes          = 1440
	currentSettingsSchemaVersion  = 1
)

func defaultBackupPolicy() models.BackupPolicy {
	return models.BackupPolicy{
		Enabled:          true,
		IntervalMinutes:  defaultBackupIntervalMinutes,
		ExitDelaySeconds: defaultBackupExitDelaySeconds,
		RetentionCount:   defaultBackupRetentionCount,
	}
}

func defaultSyncPolicy() models.SyncPolicy {
	return models.SyncPolicy{
		Enabled:                   true,
		Mode:                      models.ProtectionBalanced,
		CheckpointIntervalMinutes: defaultCheckpointMinutes,
		ExitProtection:            true,
	}
}

func normalizeSyncPolicy(policy *models.SyncPolicy) bool {
	if policy == nil {
		return false
	}
	changed := false
	defaults := defaultSyncPolicy()
	switch policy.Mode {
	case models.ProtectionBalanced, models.ProtectionExitOnly, models.ProtectionContinuous, models.ProtectionManual:
	default:
		*policy = defaults
		return true
	}
	if policy.CheckpointIntervalMinutes < minCheckpointMinutes || policy.CheckpointIntervalMinutes > maxCheckpointMinutes {
		policy.CheckpointIntervalMinutes = defaults.CheckpointIntervalMinutes
		changed = true
	}
	return changed
}

func validateSyncPolicy(policy models.SyncPolicy) error {
	switch policy.Mode {
	case models.ProtectionBalanced, models.ProtectionExitOnly, models.ProtectionContinuous, models.ProtectionManual:
	default:
		return errors.New("invalid protection mode")
	}
	if policy.CheckpointIntervalMinutes < minCheckpointMinutes || policy.CheckpointIntervalMinutes > maxCheckpointMinutes {
		return fmt.Errorf("checkpoint interval must be between %d and %d minutes", minCheckpointMinutes, maxCheckpointMinutes)
	}
	return nil
}

func normalizeBackupPolicy(policy *models.BackupPolicy) bool {
	if policy == nil {
		return false
	}
	changed := false
	defaults := defaultBackupPolicy()

	if policy.IntervalMinutes < minBackupIntervalMinutes || policy.IntervalMinutes > maxBackupIntervalMinutes {
		policy.IntervalMinutes = defaults.IntervalMinutes
		changed = true
	}
	if policy.ExitDelaySeconds < minBackupExitDelaySeconds || policy.ExitDelaySeconds > maxBackupExitDelaySeconds {
		policy.ExitDelaySeconds = defaults.ExitDelaySeconds
		changed = true
	}
	if policy.RetentionCount < minBackupRetentionCount || policy.RetentionCount > maxBackupRetentionCount {
		policy.RetentionCount = defaults.RetentionCount
		changed = true
	}

	return changed
}

func validateBackupPolicy(policy models.BackupPolicy) error {
	if policy.IntervalMinutes < minBackupIntervalMinutes || policy.IntervalMinutes > maxBackupIntervalMinutes {
		return fmt.Errorf("backup interval must be between %d and %d minutes", minBackupIntervalMinutes, maxBackupIntervalMinutes)
	}
	if policy.ExitDelaySeconds < minBackupExitDelaySeconds || policy.ExitDelaySeconds > maxBackupExitDelaySeconds {
		return fmt.Errorf("backup exit delay must be between %d and %d seconds", minBackupExitDelaySeconds, maxBackupExitDelaySeconds)
	}
	if policy.RetentionCount < minBackupRetentionCount || policy.RetentionCount > maxBackupRetentionCount {
		return fmt.Errorf("backup retention count must be between %d and %d", minBackupRetentionCount, maxBackupRetentionCount)
	}
	return nil
}

// coordinates in-memory settings state with on-disk persistence.
type Manager struct {
	mu             sync.RWMutex
	path           string
	settings       models.Settings
	recoveryNotice string
}

// creates a manager and loads settings from disk, creating defaults when missing.
func New(path string) (*Manager, error) {
	if path == "" {
		resolved, err := defaultPath()
		if err != nil {
			return nil, err
		}
		path = resolved
	}

	m := &Manager{path: path}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

// resolves the per-user settings.json location under the app data directory.
func defaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, branding.InternalName, "settings.json"), nil
}

// returns baseline values used for first run and normalization fallback.
func DefaultSettings() models.Settings {
	return models.Settings{
		SchemaVersion:           currentSettingsSchemaVersion,
		Port:                    runtimecfg.DefaultAPIPort,
		SyncIntervalSeconds:     30,
		ConflictPolicy:          models.ConflictLatestWins,
		ThemeMode:               models.ThemeLight,
		Autostart:               false,
		IgnorePatterns:          []string{"*.tmp", "*.log", "cache"},
		SnapshotVaultPath:       "",
		SnapshotMaxPerGame:      defaultSnapshotMaxPerGame,
		SnapshotOnExit:          true,
		SnapshotPeriodic:        false,
		SnapshotIntervalMinutes: defaultSnapshotIntervalMins,
		DeviceID:                uuid.NewString(),
		SyncRetention:           10,
		ReviewConflicts:         true,
		CloudProvider:           models.CloudProviderFolder,
		ProtectionTarget:        models.ProtectionTargetLocal,
		ServiceControl: models.ServiceControlState{
			ProtectionEnabled: true,
			PauseScope:        models.PauseNone,
		},
		AutomaticUpload:         true,
		UploadOnMetered:         false,
		RemoteCheckBeforeLaunch: true,
		ImportantNotifications:  true,
		DebugLogging:            false,
		AutoRevisionRetention:   3,
		ProtectionDefaultsReady: true,
		Games:                   []models.Game{},
	}
}

// RecoveryNotice reports a safe startup message when incompatible or damaged
// settings were repaired. It contains field/base names only, never full paths
// or credential values.
func (m *Manager) RecoveryNotice() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recoveryNotice
}

// returns the active settings file path.
func (m *Manager) Path() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.path
}

// returns a safe cloned copy of in-memory settings.
func (m *Manager) Get() models.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cloned := m.settings
	cloned.Games = append([]models.Game{}, m.settings.Games...)
	cloned.IgnorePatterns = append([]string{}, m.settings.IgnorePatterns...)
	return cloned
}

// validates and persists mutable settings fields from api input.
func (m *Manager) Update(next models.Settings) error {
	if next.Port <= 0 || next.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if next.SyncIntervalSeconds < 5 {
		return errors.New("sync interval must be >= 5 seconds")
	}
	if next.ConflictPolicy != models.ConflictLatestWins && next.ConflictPolicy != models.ConflictKeepBoth {
		return errors.New("invalid conflict policy")
	}
	if next.ThemeMode != models.ThemeDark && next.ThemeMode != models.ThemeLight && next.ThemeMode != models.ThemeSystem {
		return errors.New("invalid theme mode")
	}
	switch next.ProtectionTarget {
	case models.ProtectionTargetLocal, models.ProtectionTargetFolder, models.ProtectionTargetDrive:
	default:
		return errors.New("invalid protection target")
	}
	if next.ProtectionTarget == models.ProtectionTargetDrive && !next.GoogleDrive.Connected {
		return errors.New("connect Google Drive before selecting it as the cloud provider")
	}
	if next.SnapshotMaxPerGame < minSnapshotMaxPerGame || next.SnapshotMaxPerGame > maxSnapshotMaxPerGame {
		return fmt.Errorf("snapshot max per game must be between %d and %d", minSnapshotMaxPerGame, maxSnapshotMaxPerGame)
	}
	if next.SnapshotIntervalMinutes < minSnapshotIntervalMins || next.SnapshotIntervalMinutes > maxSnapshotIntervalMins {
		return fmt.Errorf("snapshot interval must be between %d and %d minutes", minSnapshotIntervalMins, maxSnapshotIntervalMins)
	}
	if strings.TrimSpace(next.DeviceID) == "" {
		return errors.New("device id is required")
	}
	if next.SyncRetention < 1 || next.SyncRetention > 50 {
		return errors.New("sync retention must be between 1 and 50")
	}
	if next.AutoRevisionRetention < 1 || next.AutoRevisionRetention > 50 {
		return errors.New("automatic revision retention must be between 1 and 50")
	}
	if next.ServiceControl.PauseScope != models.PauseNone && next.ServiceControl.PauseScope != models.PauseUploads && next.ServiceControl.PauseScope != models.PauseAll {
		return errors.New("invalid protection pause scope")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.Port = next.Port
	m.settings.SyncIntervalSeconds = next.SyncIntervalSeconds
	m.settings.ConflictPolicy = next.ConflictPolicy
	m.settings.ThemeMode = next.ThemeMode
	m.settings.Autostart = next.Autostart
	m.settings.IgnorePatterns = append([]string{}, next.IgnorePatterns...)
	m.settings.SnapshotVaultPath = next.SnapshotVaultPath
	m.settings.SnapshotMaxPerGame = next.SnapshotMaxPerGame
	m.settings.SnapshotOnExit = next.SnapshotOnExit
	m.settings.SnapshotPeriodic = next.SnapshotPeriodic
	m.settings.SnapshotIntervalMinutes = next.SnapshotIntervalMinutes
	m.settings.SyncRoot = next.SyncRoot
	m.settings.DeviceID = next.DeviceID
	m.settings.SyncRetention = next.SyncRetention
	m.settings.ReviewConflicts = next.ReviewConflicts
	m.settings.ProtectionTarget = next.ProtectionTarget
	m.settings.ServiceControl = next.ServiceControl
	m.settings.AutomaticUpload = next.AutomaticUpload
	m.settings.UploadOnMetered = next.UploadOnMetered
	m.settings.RemoteCheckBeforeLaunch = next.RemoteCheckBeforeLaunch
	m.settings.ImportantNotifications = next.ImportantNotifications
	m.settings.DebugLogging = next.DebugLogging
	m.settings.AutoRevisionRetention = next.AutoRevisionRetention
	m.settings.ProtectionDefaultsReady = true
	switch next.ProtectionTarget {
	case models.ProtectionTargetDrive:
		m.settings.CloudProvider = models.CloudProviderGoogleDrive
	default:
		m.settings.CloudProvider = models.CloudProviderFolder
	}
	// The UI never receives the encrypted token. Preserve the existing value on
	// ordinary settings saves; connect/disconnect use dedicated methods below.
	m.settings.GoogleDrive.Connected = next.GoogleDrive.Connected
	m.settings.GoogleDrive.AccountEmail = next.GoogleDrive.AccountEmail
	m.settings.GoogleDrive.ReconnectRequired = next.GoogleDrive.ReconnectRequired
	if strings.TrimSpace(next.GoogleDrive.EncryptedRefreshToken) != "" {
		m.settings.GoogleDrive.EncryptedRefreshToken = next.GoogleDrive.EncryptedRefreshToken
	}
	return m.saveLocked()
}

// SetGoogleDriveConnection persists encrypted credentials after an OAuth flow.
func (m *Manager) SetGoogleDriveConnection(email, encryptedRefreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.GoogleDrive = models.GoogleDriveSettings{Connected: true, AccountEmail: strings.TrimSpace(email), EncryptedRefreshToken: encryptedRefreshToken}
	return m.saveLocked()
}

// ClearGoogleDriveConnection removes all locally stored Google credentials.
func (m *Manager) ClearGoogleDriveConnection() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.GoogleDrive = models.GoogleDriveSettings{}
	if m.settings.CloudProvider == models.CloudProviderGoogleDrive {
		m.settings.CloudProvider = models.CloudProviderFolder
	}
	if m.settings.ProtectionTarget == models.ProtectionTargetDrive {
		m.settings.ProtectionTarget = models.ProtectionTargetLocal
	}
	return m.saveLocked()
}

// MarkGoogleDriveReconnectRequired keeps the provider choice but makes sync
// failures actionable without exposing an OAuth error to the UI.
func (m *Manager) MarkGoogleDriveReconnectRequired() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings.GoogleDrive.ReconnectRequired = true
	return m.saveLocked()
}

// reads settings from disk and normalizes missing or legacy fields.
func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	payload, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		m.settings = DefaultSettings()
		return m.saveLocked()
	}
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	if len(payload) == 0 {
		m.settings = DefaultSettings()
		return m.saveLocked()
	}

	// trim utf-8 bom so settings written by some windows tools still decode.
	if bytes.HasPrefix(payload, []byte{0xEF, 0xBB, 0xBF}) {
		payload = payload[3:]
	}

	settings, incompatible, err := decodeSettingsFields(payload)
	if err != nil {
		return m.recoverUnreadableSettingsLocked(payload, err)
	}
	if settings.SchemaVersion > currentSettingsSchemaVersion {
		return fmt.Errorf("settings schema %d is newer than this application supports (%d)", settings.SchemaVersion, currentSettingsSchemaVersion)
	}
	m.settings = settings
	if len(incompatible) > 0 {
		m.recoveryNotice = "Reset incompatible settings fields to safe defaults: " + strings.Join(incompatible, ", ")
	}
	if normalizeSettings(&m.settings) || len(incompatible) > 0 {
		return m.saveLocked()
	}
	return nil
}

// backfills defaults for zero, nil, or invalid fields from older files.
func normalizeSettings(settings *models.Settings) bool {
	if settings == nil {
		return false
	}
	changed := false
	defaults := DefaultSettings()
	if settings.SchemaVersion != currentSettingsSchemaVersion {
		settings.SchemaVersion = currentSettingsSchemaVersion
		changed = true
	}

	if settings.Port == 0 {
		settings.Port = defaults.Port
		changed = true
	}
	if settings.SyncIntervalSeconds == 0 {
		settings.SyncIntervalSeconds = defaults.SyncIntervalSeconds
		changed = true
	}
	if settings.ConflictPolicy != models.ConflictLatestWins && settings.ConflictPolicy != models.ConflictKeepBoth {
		settings.ConflictPolicy = defaults.ConflictPolicy
		changed = true
	}
	if settings.ThemeMode != models.ThemeDark && settings.ThemeMode != models.ThemeLight && settings.ThemeMode != models.ThemeSystem {
		settings.ThemeMode = defaults.ThemeMode
		changed = true
	}
	if settings.IgnorePatterns == nil {
		settings.IgnorePatterns = append([]string{}, defaults.IgnorePatterns...)
		changed = true
	}
	if settings.Games == nil {
		settings.Games = []models.Game{}
		changed = true
	}
	if settings.SnapshotMaxPerGame < minSnapshotMaxPerGame || settings.SnapshotMaxPerGame > maxSnapshotMaxPerGame {
		settings.SnapshotMaxPerGame = defaults.SnapshotMaxPerGame
		changed = true
	}
	if settings.SnapshotIntervalMinutes < minSnapshotIntervalMins || settings.SnapshotIntervalMinutes > maxSnapshotIntervalMins {
		settings.SnapshotIntervalMinutes = defaults.SnapshotIntervalMinutes
		changed = true
	}
	if strings.TrimSpace(settings.DeviceID) == "" {
		settings.DeviceID = uuid.NewString()
		changed = true
	}
	if settings.SyncRetention < 1 || settings.SyncRetention > 50 {
		settings.SyncRetention = 10
		changed = true
	}
	if !settings.ReviewConflicts {
		// This migration intentionally favours data safety over legacy automatic
		// policies. New sync installations always require review.
		settings.ReviewConflicts = true
		changed = true
	}
	if settings.CloudProvider == "" {
		settings.CloudProvider = models.CloudProviderFolder
		changed = true
	}
	if !settings.ProtectionDefaultsReady {
		settings.ServiceControl = defaults.ServiceControl
		settings.AutomaticUpload = defaults.AutomaticUpload
		settings.UploadOnMetered = defaults.UploadOnMetered
		settings.RemoteCheckBeforeLaunch = defaults.RemoteCheckBeforeLaunch
		settings.ImportantNotifications = defaults.ImportantNotifications
		settings.DebugLogging = defaults.DebugLogging
		settings.AutoRevisionRetention = defaults.AutoRevisionRetention
		settings.ProtectionDefaultsReady = true
		switch settings.CloudProvider {
		case models.CloudProviderGoogleDrive:
			settings.ProtectionTarget = models.ProtectionTargetDrive
		default:
			if strings.TrimSpace(settings.SyncRoot) != "" {
				settings.ProtectionTarget = models.ProtectionTargetFolder
			} else {
				settings.ProtectionTarget = models.ProtectionTargetLocal
			}
		}
		changed = true
	}
	if settings.ProtectionTarget != models.ProtectionTargetLocal && settings.ProtectionTarget != models.ProtectionTargetFolder && settings.ProtectionTarget != models.ProtectionTargetDrive {
		settings.ProtectionTarget = defaults.ProtectionTarget
		changed = true
	}
	if settings.ServiceControl.PauseScope != models.PauseNone && settings.ServiceControl.PauseScope != models.PauseUploads && settings.ServiceControl.PauseScope != models.PauseAll {
		settings.ServiceControl.PauseScope = models.PauseNone
		changed = true
	}
	if settings.AutoRevisionRetention < 1 || settings.AutoRevisionRetention > 50 {
		settings.AutoRevisionRetention = defaults.AutoRevisionRetention
		changed = true
	}
	if settings.CloudProvider != models.CloudProviderFolder && settings.CloudProvider != models.CloudProviderGoogleDrive {
		settings.CloudProvider = models.CloudProviderFolder
		changed = true
	}
	for i := range settings.Games {
		if normalizeBackupPolicy(&settings.Games[i].BackupPolicy) {
			changed = true
		}
		if strings.TrimSpace(settings.Games[i].SyncID) == "" {
			settings.Games[i].SyncID = uuid.NewString()
			changed = true
		}
		if normalizeSyncPolicy(&settings.Games[i].ProtectionPolicy) {
			changed = true
		}
	}

	return changed
}

// writes in-memory settings to disk and assumes caller holds manager lock.
func (m *Manager) saveLocked() error {
	payload, err := json.MarshalIndent(m.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	// Preserve the last syntactically valid file before replacing it. A corrupt
	// current file never overwrites the recovery copy.
	if current, readErr := os.ReadFile(m.path); readErr == nil && json.Valid(current) && !bytes.Equal(current, payload) {
		if err := writeFileAtomically(m.path+".bak", current, 0o600); err != nil {
			return fmt.Errorf("write settings backup: %w", err)
		}
	}
	if err := writeFileAtomically(m.path, payload, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// decodeSettingsFields migrates valid JSON one known field at a time. Unknown
// fields are ignored for forward compatibility, while a type-changed field is
// reset without discarding unrelated games, provider credentials, or settings.
func decodeSettingsFields(payload []byte) (models.Settings, []string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return models.Settings{}, nil, fmt.Errorf("decode settings JSON: %w", err)
	}

	var settings models.Settings
	value := reflect.ValueOf(&settings).Elem()
	typeInfo := value.Type()
	incompatible := make([]string, 0)
	for i := 0; i < typeInfo.NumField(); i++ {
		fieldInfo := typeInfo.Field(i)
		name := strings.Split(fieldInfo.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		encoded, ok := raw[name]
		if !ok {
			continue
		}
		field := value.Field(i)
		if !field.CanAddr() {
			continue
		}
		if err := json.Unmarshal(encoded, field.Addr().Interface()); err != nil {
			incompatible = append(incompatible, name)
		}
	}
	sort.Strings(incompatible)
	return settings, incompatible, nil
}

func (m *Manager) recoverUnreadableSettingsLocked(payload []byte, decodeErr error) error {
	quarantineName, quarantineErr := preserveInvalidSettings(m.path, payload)
	if quarantineErr != nil {
		return fmt.Errorf("decode settings: %w; preserve invalid settings: %v", decodeErr, quarantineErr)
	}

	backupPayload, backupErr := os.ReadFile(m.path + ".bak")
	if backupErr == nil {
		backup, incompatible, err := decodeSettingsFields(backupPayload)
		if err == nil && backup.SchemaVersion <= currentSettingsSchemaVersion {
			m.settings = backup
			normalizeSettings(&m.settings)
			m.recoveryNotice = "Recovered settings from settings.json.bak; unreadable original preserved as " + quarantineName
			if len(incompatible) > 0 {
				m.recoveryNotice += "; reset fields: " + strings.Join(incompatible, ", ")
			}
			return m.saveLocked()
		}
	}

	m.settings = DefaultSettings()
	m.recoveryNotice = "Created safe default settings; unreadable original preserved as " + quarantineName
	return m.saveLocked()
}

func preserveInvalidSettings(path string, payload []byte) (string, error) {
	name := filepath.Base(path) + ".invalid-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	target := filepath.Join(filepath.Dir(path), name)
	if err := writeFileAtomically(target, payload, 0o600); err != nil {
		return "", err
	}
	return name, nil
}

func writeFileAtomically(path string, payload []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".cloud-saver-settings-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temp.Close()
		}
		_ = os.Remove(tempPath)
	}()
	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
}
