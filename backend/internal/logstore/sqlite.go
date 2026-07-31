package logstore

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"cloud-saver/backend/internal/models"
)

type Store struct {
	db   *sql.DB
	path string
}

const (
	busyRetryCount = 4
	busyRetryDelay = 30 * time.Millisecond
)

func Open(root string) (*Store, error) {
	dbPath := filepath.Join(root, "sync.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// keep a single sqlite connection to avoid write lock churn under bursty ui refreshes.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := configure(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db, path: dbPath}, nil
}

func configure(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return fmt.Errorf("set sqlite journal mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return fmt.Errorf("set sqlite busy timeout: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		return fmt.Errorf("set sqlite synchronous mode: %w", err)
	}
	return nil
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  level TEXT NOT NULL,
  event TEXT NOT NULL,
  message TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS conflicts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  game_id TEXT NOT NULL,
  file_path TEXT NOT NULL,
  strategy TEXT NOT NULL,
  message TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS game_stats (
	game_id TEXT PRIMARY KEY,
	total_play_seconds INTEGER NOT NULL DEFAULT 0,
	last_played_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS backup_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id TEXT NOT NULL,
	path TEXT NOT NULL,
	created_at TEXT NOT NULL,
	reason TEXT NOT NULL,
	signature TEXT NOT NULL,
	status TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_backup_history_game_created ON backup_history(game_id, created_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS sync_state (
  game_id TEXT PRIMARY KEY,
  revision_id TEXT NOT NULL DEFAULT '',
  manifest_json TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS sync_conflicts (
  id TEXT PRIMARY KEY,
  game_id TEXT NOT NULL,
  path TEXT NOT NULL,
  local_hash TEXT NOT NULL,
  remote_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  state TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_game_state ON sync_conflicts(game_id, state, created_at DESC);
CREATE TABLE IF NOT EXISTS file_state (
  game_id TEXT NOT NULL,
  relative_path TEXT NOT NULL,
  size INTEGER NOT NULL DEFAULT 0,
  modified_ns INTEGER NOT NULL DEFAULT 0,
  file_id TEXT NOT NULL DEFAULT '',
  sha256 TEXT NOT NULL DEFAULT '',
  dirty INTEGER NOT NULL DEFAULT 1,
  last_revision_id TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(game_id, relative_path)
);
CREATE INDEX IF NOT EXISTS idx_file_state_game_dirty ON file_state(game_id, dirty);
CREATE TABLE IF NOT EXISTS protection_jobs (
  id TEXT PRIMARY KEY,
  game_id TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL,
  state TEXT NOT NULL,
  stage TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  revision_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  changed_files INTEGER NOT NULL DEFAULT 0,
  bytes INTEGER NOT NULL DEFAULT 0,
  progress INTEGER NOT NULL DEFAULT 0,
  retry_count INTEGER NOT NULL DEFAULT 0,
  next_retry_at TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  conflict_ids_json TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_protection_jobs_state_retry ON protection_jobs(state, next_retry_at, priority DESC);
CREATE INDEX IF NOT EXISTS idx_protection_jobs_game_updated ON protection_jobs(game_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS protection_revisions (
  id TEXT PRIMARY KEY,
  game_id TEXT NOT NULL,
  sync_id TEXT NOT NULL,
  parent_ids_json TEXT NOT NULL DEFAULT '[]',
  kind TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  file_count INTEGER NOT NULL DEFAULT 0,
  total_bytes INTEGER NOT NULL DEFAULT 0,
  manifest_json TEXT NOT NULL,
  pinned INTEGER NOT NULL DEFAULT 0,
  uploaded INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_protection_revisions_game_created ON protection_revisions(game_id, created_at DESC);
CREATE TABLE IF NOT EXISTS recovery_points (
  id TEXT PRIMARY KEY,
  game_id TEXT NOT NULL,
  revision_id TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(revision_id) REFERENCES protection_revisions(id)
);
CREATE INDEX IF NOT EXISTS idx_recovery_points_game_created ON recovery_points(game_id, created_at DESC);
CREATE TABLE IF NOT EXISTS provider_health (
  provider TEXT PRIMARY KEY,
  state TEXT NOT NULL,
  last_success_at TEXT NOT NULL DEFAULT '',
  last_attempt_at TEXT NOT NULL DEFAULT '',
  next_retry_at TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS provider_cursors (
  provider TEXT PRIMARY KEY,
  cursor TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS service_control (
  scope_key TEXT PRIMARY KEY,
  protection_enabled INTEGER NOT NULL DEFAULT 1,
  pause_scope TEXT NOT NULL DEFAULT 'none',
  pause_until TEXT NOT NULL DEFAULT '',
  pause_until_login INTEGER NOT NULL DEFAULT 0,
  reason TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS game_sessions (
  game_id TEXT PRIMARY KEY,
  running INTEGER NOT NULL DEFAULT 0,
  dirty INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL DEFAULT '',
  ended_at TEXT NOT NULL DEFAULT '',
  needs_reconciliation INTEGER NOT NULL DEFAULT 0,
  last_audit_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS cleanup_quarantine (
  object_key TEXT PRIMARY KEY,
  game_id TEXT NOT NULL,
  marked_at TEXT NOT NULL,
  verified_at TEXT NOT NULL DEFAULT '',
  bytes INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS activity_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category TEXT NOT NULL,
  severity TEXT NOT NULL,
  game_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  ts TEXT NOT NULL,
  action_target TEXT NOT NULL DEFAULT '',
  dedupe_key TEXT NOT NULL DEFAULT '',
  resolved INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_activity_events_ts ON activity_events(ts DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_events_open_dedupe ON activity_events(dedupe_key) WHERE dedupe_key <> '' AND resolved = 0;
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	return nil
}

// SyncState is the last fully reconciled manifest for a local game.
type SyncState struct {
	RevisionID   string
	ManifestJSON string
}

func (s *Store) SyncState(gameID string) (SyncState, error) {
	if gameID == "" {
		return SyncState{}, nil
	}
	var state SyncState
	err := s.db.QueryRow(`SELECT revision_id, manifest_json FROM sync_state WHERE game_id = ?`, gameID).Scan(&state.RevisionID, &state.ManifestJSON)
	if err == sql.ErrNoRows {
		return SyncState{}, nil
	}
	if err != nil {
		return SyncState{}, fmt.Errorf("query sync state: %w", err)
	}
	return state, nil
}

func (s *Store) SaveSyncState(gameID, revisionID, manifestJSON string) error {
	if gameID == "" {
		return fmt.Errorf("sync state requires game id")
	}
	_, err := s.db.Exec(`
INSERT INTO sync_state(game_id, revision_id, manifest_json, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(game_id) DO UPDATE SET revision_id = excluded.revision_id, manifest_json = excluded.manifest_json, updated_at = excluded.updated_at`,
		gameID, revisionID, manifestJSON, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save sync state: %w", err)
	}
	return nil
}

func (s *Store) CreateSyncConflict(conflict models.SyncConflict) error {
	if conflict.ID == "" || conflict.GameID == "" || conflict.Path == "" {
		return fmt.Errorf("sync conflict is incomplete")
	}
	if conflict.CreatedAt.IsZero() {
		conflict.CreatedAt = time.Now().UTC()
	}
	if conflict.State == "" {
		conflict.State = "open"
	}
	_, err := s.db.Exec(`INSERT OR REPLACE INTO sync_conflicts(id, game_id, path, local_hash, remote_hash, created_at, state) VALUES (?, ?, ?, ?, ?, ?, ?)`, conflict.ID, conflict.GameID, conflict.Path, conflict.LocalHash, conflict.RemoteHash, conflict.CreatedAt.Format(time.RFC3339), conflict.State)
	return err
}

func (s *Store) SyncConflicts(gameID string) ([]models.SyncConflict, error) {
	rows, err := s.db.Query(`SELECT id, game_id, path, local_hash, remote_hash, created_at, state FROM sync_conflicts WHERE game_id = ? AND state = 'open' ORDER BY created_at DESC`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.SyncConflict{}
	for rows.Next() {
		var value models.SyncConflict
		var raw string
		if err := rows.Scan(&value.ID, &value.GameID, &value.Path, &value.LocalHash, &value.RemoteHash, &raw, &value.State); err != nil {
			return nil, err
		}
		value.CreatedAt, _ = time.Parse(time.RFC3339, raw)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) SyncConflict(id string) (models.SyncConflict, error) {
	var value models.SyncConflict
	var raw string
	err := s.db.QueryRow(`SELECT id, game_id, path, local_hash, remote_hash, created_at, state FROM sync_conflicts WHERE id = ?`, id).Scan(&value.ID, &value.GameID, &value.Path, &value.LocalHash, &value.RemoteHash, &raw, &value.State)
	if err != nil {
		return models.SyncConflict{}, err
	}
	value.CreatedAt, _ = time.Parse(time.RFC3339, raw)
	return value, nil
}

func (s *Store) ResolveSyncConflict(id string) error {
	_, err := s.db.Exec(`UPDATE sync_conflicts SET state = 'resolved' WHERE id = ?`, id)
	return err
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Write(level, event, message string) error {
	if level == "" {
		level = "info"
	}
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		_, err := s.db.Exec(`INSERT INTO logs(ts, level, event, message) VALUES (?, ?, ?, ?)`, time.Now().UTC().Format(time.RFC3339), level, event, message)
		if err == nil {
			return nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return fmt.Errorf("insert log: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return nil
}

func (s *Store) WriteConflict(gameID, filePath, strategy, message string) error {
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		_, err := s.db.Exec(`INSERT INTO conflicts(ts, game_id, file_path, strategy, message) VALUES (?, ?, ?, ?, ?)`, time.Now().UTC().Format(time.RFC3339), gameID, filePath, strategy, message)
		if err == nil {
			return nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return fmt.Errorf("insert conflict: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return nil
}

func (s *Store) Recent(limit int) ([]models.LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var rows *sql.Rows
	var err error
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		rows, err = s.db.Query(`SELECT id, ts, level, event, message FROM logs ORDER BY id DESC LIMIT ?`, limit)
		if err == nil {
			break
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return nil, fmt.Errorf("query logs: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	defer rows.Close()

	result := make([]models.LogEntry, 0, limit)
	for rows.Next() {
		var row models.LogEntry
		var ts string
		if err := rows.Scan(&row.ID, &ts, &row.Level, &row.Event, &row.Message); err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}
		parsed, _ := time.Parse(time.RFC3339, ts)
		row.Timestamp = parsed
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate logs: %w", err)
	}
	return result, nil
}

func (s *Store) AddPlaytime(gameID string, seconds int64, playedAt time.Time) error {
	if gameID == "" || seconds <= 0 {
		return nil
	}
	ts := playedAt.UTC().Format(time.RFC3339)
	const q = `
INSERT INTO game_stats(game_id, total_play_seconds, last_played_at)
VALUES (?, ?, ?)
ON CONFLICT(game_id) DO UPDATE SET
  total_play_seconds = total_play_seconds + excluded.total_play_seconds,
  last_played_at = excluded.last_played_at;
`
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		_, err := s.db.Exec(q, gameID, seconds, ts)
		if err == nil {
			return nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return fmt.Errorf("upsert game_stats: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return nil
}

func (s *Store) GameStats(gameIDs []string) (map[string]models.GameStats, error) {
	stats := make(map[string]models.GameStats, len(gameIDs))
	if len(gameIDs) == 0 {
		return stats, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(gameIDs)), ",")
	q := fmt.Sprintf(`SELECT game_id, total_play_seconds, last_played_at FROM game_stats WHERE game_id IN (%s)`, placeholders)
	args := make([]any, 0, len(gameIDs))
	for _, id := range gameIDs {
		args = append(args, id)
	}

	var rows *sql.Rows
	var err error
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		rows, err = s.db.Query(q, args...)
		if err == nil {
			break
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return nil, fmt.Errorf("query game_stats: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	defer rows.Close()

	for rows.Next() {
		var gameID string
		var total int64
		var lastPlayedRaw string
		if err := rows.Scan(&gameID, &total, &lastPlayedRaw); err != nil {
			return nil, fmt.Errorf("scan game_stats: %w", err)
		}
		parsed, _ := time.Parse(time.RFC3339, lastPlayedRaw)
		stats[gameID] = models.GameStats{PlaytimeSeconds: total, LastPlayedAt: parsed}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate game_stats: %w", err)
	}

	return stats, nil
}

func (s *Store) DeleteGameStats(gameID string) error {
	if gameID == "" {
		return nil
	}
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		_, err := s.db.Exec(`DELETE FROM game_stats WHERE game_id = ?`, gameID)
		if err == nil {
			return nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return fmt.Errorf("delete game_stats: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return nil
}

// stores one auto-backup record for later history and signature checks.
func (s *Store) InsertBackupHistory(entry models.BackupHistoryEntry) error {
	if entry.GameID == "" || entry.Path == "" {
		return fmt.Errorf("backup history requires game id and path")
	}
	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	reason := entry.Reason
	if reason == "" {
		reason = models.BackupReasonPeriodic
	}
	status := entry.Status
	if status == "" {
		status = models.BackupStatusOK
	}

	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		_, err := s.db.Exec(
			`INSERT INTO backup_history(game_id, path, created_at, reason, signature, status) VALUES (?, ?, ?, ?, ?, ?)`,
			entry.GameID,
			entry.Path,
			createdAt.UTC().Format(time.RFC3339),
			reason,
			entry.Signature,
			status,
		)
		if err == nil {
			return nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return fmt.Errorf("insert backup_history: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return nil
}

// returns recent local backup records for one game ordered newest first.
func (s *Store) BackupHistory(gameID string, limit int) ([]models.BackupHistoryEntry, error) {
	if gameID == "" {
		return []models.BackupHistoryEntry{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		rows, err = s.db.Query(
			`SELECT id, game_id, path, created_at, reason, signature, status FROM backup_history WHERE game_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
			gameID,
			limit,
		)
		if err == nil {
			break
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return nil, fmt.Errorf("query backup_history: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	defer rows.Close()

	result := make([]models.BackupHistoryEntry, 0, limit)
	for rows.Next() {
		var row models.BackupHistoryEntry
		var createdRaw string
		if err := rows.Scan(&row.ID, &row.GameID, &row.Path, &createdRaw, &row.Reason, &row.Signature, &row.Status); err != nil {
			return nil, fmt.Errorf("scan backup_history: %w", err)
		}
		parsed, _ := time.Parse(time.RFC3339, createdRaw)
		row.CreatedAt = parsed
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup_history: %w", err)
	}
	return result, nil
}

// returns latest successful backup signature for change detection decisions.
func (s *Store) LatestBackupSignature(gameID string) (string, error) {
	if gameID == "" {
		return "", nil
	}

	var signature string
	var err error
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		err = s.db.QueryRow(
			`SELECT signature FROM backup_history WHERE game_id = ? AND status = ? ORDER BY created_at DESC, id DESC LIMIT 1`,
			gameID,
			models.BackupStatusOK,
		).Scan(&signature)
		if err == nil {
			return signature, nil
		}
		if err == sql.ErrNoRows {
			return "", nil
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return "", fmt.Errorf("query latest backup signature: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	return "", nil
}

// trims backup history to keep only the newest records and returns removed paths.
func (s *Store) PruneBackupHistory(gameID string, keep int) ([]string, error) {
	if gameID == "" || keep < 1 {
		return []string{}, nil
	}

	var rows *sql.Rows
	var err error
	for attempt := 0; attempt <= busyRetryCount; attempt++ {
		rows, err = s.db.Query(
			`SELECT id, path FROM backup_history WHERE game_id = ? ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET ?`,
			gameID,
			keep,
		)
		if err == nil {
			break
		}
		if !isBusyError(err) || attempt == busyRetryCount {
			return nil, fmt.Errorf("query prune backup_history: %w", err)
		}
		time.Sleep(busyRetryDelay)
	}
	defer rows.Close()

	type pruneRow struct {
		id   int64
		path string
	}
	toRemove := make([]pruneRow, 0)
	for rows.Next() {
		var row pruneRow
		if err := rows.Scan(&row.id, &row.path); err != nil {
			return nil, fmt.Errorf("scan prune backup_history: %w", err)
		}
		toRemove = append(toRemove, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prune backup_history: %w", err)
	}

	removedPaths := make([]string, 0, len(toRemove))
	for _, row := range toRemove {
		for attempt := 0; attempt <= busyRetryCount; attempt++ {
			_, err := s.db.Exec(`DELETE FROM backup_history WHERE id = ?`, row.id)
			if err == nil {
				removedPaths = append(removedPaths, row.path)
				break
			}
			if !isBusyError(err) || attempt == busyRetryCount {
				return nil, fmt.Errorf("delete backup_history row: %w", err)
			}
			time.Sleep(busyRetryDelay)
		}
	}

	return removedPaths, nil
}

func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy")
}
