package logstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud-saver/backend/internal/models"
)

// FileState is the persistent incremental hashing cache for one save path.
type FileState struct {
	GameID         string
	RelativePath   string
	Size           int64
	ModifiedNS     int64
	FileID         string
	SHA256         string
	Dirty          bool
	LastRevisionID string
}

func (s *Store) FileStates(gameID string) (map[string]FileState, error) {
	rows, err := s.db.Query(`SELECT game_id, relative_path, size, modified_ns, file_id, sha256, dirty, last_revision_id FROM file_state WHERE game_id = ?`, gameID)
	if err != nil {
		return nil, fmt.Errorf("query file state: %w", err)
	}
	defer rows.Close()
	result := map[string]FileState{}
	for rows.Next() {
		var value FileState
		var dirty int
		if err := rows.Scan(&value.GameID, &value.RelativePath, &value.Size, &value.ModifiedNS, &value.FileID, &value.SHA256, &dirty, &value.LastRevisionID); err != nil {
			return nil, fmt.Errorf("scan file state: %w", err)
		}
		value.Dirty = dirty != 0
		result[value.RelativePath] = value
	}
	return result, rows.Err()
}

func (s *Store) MarkFileDirty(gameID, relativePath string) error {
	if gameID == "" || relativePath == "" {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO file_state(game_id, relative_path, dirty) VALUES (?, ?, 1)
ON CONFLICT(game_id, relative_path) DO UPDATE SET dirty = 1`, gameID, relativePath)
	return err
}

func (s *Store) UpsertFileState(value FileState) error {
	_, err := s.db.Exec(`
INSERT INTO file_state(game_id, relative_path, size, modified_ns, file_id, sha256, dirty, last_revision_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(game_id, relative_path) DO UPDATE SET
 size=excluded.size, modified_ns=excluded.modified_ns, file_id=excluded.file_id,
 sha256=excluded.sha256, dirty=excluded.dirty, last_revision_id=excluded.last_revision_id`,
		value.GameID, value.RelativePath, value.Size, value.ModifiedNS, value.FileID, value.SHA256, boolInt(value.Dirty), value.LastRevisionID)
	return err
}

func (s *Store) DeleteFileState(gameID, relativePath string) error {
	_, err := s.db.Exec(`DELETE FROM file_state WHERE game_id = ? AND relative_path = ?`, gameID, relativePath)
	return err
}

func (s *Store) MarkAllFilesDirty(gameID string) error {
	_, err := s.db.Exec(`UPDATE file_state SET dirty = 1 WHERE game_id = ?`, gameID)
	return err
}

func (s *Store) DirtyFileCount(gameID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM file_state WHERE game_id = ? AND dirty = 1`, gameID).Scan(&count)
	return count, err
}

func (s *Store) SaveProtectionJob(job models.SyncJob) error {
	now := time.Now().UTC()
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	conflicts, _ := json.Marshal(job.ConflictIDs)
	_, err := s.db.Exec(`
INSERT INTO protection_jobs(id, game_id, reason, state, stage, priority, revision_id, created_at, started_at, updated_at, finished_at, changed_files, bytes, progress, retry_count, next_retry_at, error_code, error_message, conflict_ids_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
 state=excluded.state, stage=excluded.stage, priority=excluded.priority, revision_id=excluded.revision_id,
 started_at=excluded.started_at, updated_at=excluded.updated_at, finished_at=excluded.finished_at,
 changed_files=excluded.changed_files, bytes=excluded.bytes, progress=excluded.progress,
 retry_count=excluded.retry_count, next_retry_at=excluded.next_retry_at,
 error_code=excluded.error_code, error_message=excluded.error_message,
 conflict_ids_json=excluded.conflict_ids_json`,
		job.ID, job.GameID, job.Reason, job.State, job.Stage, job.Priority, job.RevisionID,
		formatTime(createdTime(job)), formatTime(job.StartedAt), formatTime(job.UpdatedAt), formatTime(job.FinishedAt),
		job.ChangedFiles, job.Bytes, job.Progress, job.RetryCount, formatTime(job.NextRetryAt),
		job.ErrorCode, job.Error, string(conflicts))
	return err
}

func createdTime(job models.SyncJob) time.Time {
	if !job.StartedAt.IsZero() {
		return job.StartedAt
	}
	if !job.UpdatedAt.IsZero() {
		return job.UpdatedAt
	}
	return time.Now().UTC()
}

func (s *Store) ProtectionJobs(gameID string, limit int) ([]models.SyncJob, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, game_id, reason, state, stage, priority, revision_id, started_at, updated_at, finished_at, changed_files, bytes, progress, retry_count, next_retry_at, error_code, error_message, conflict_ids_json FROM protection_jobs`
	args := []any{}
	if gameID != "" {
		query += ` WHERE game_id = ?`
		args = append(args, gameID)
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query protection jobs: %w", err)
	}
	defer rows.Close()
	result := []models.SyncJob{}
	for rows.Next() {
		var job models.SyncJob
		var started, updated, finished, retryAt, conflictJSON string
		if err := rows.Scan(&job.ID, &job.GameID, &job.Reason, &job.State, &job.Stage, &job.Priority, &job.RevisionID, &started, &updated, &finished, &job.ChangedFiles, &job.Bytes, &job.Progress, &job.RetryCount, &retryAt, &job.ErrorCode, &job.Error, &conflictJSON); err != nil {
			return nil, err
		}
		job.StartedAt = parseTime(started)
		job.UpdatedAt = parseTime(updated)
		job.FinishedAt = parseTime(finished)
		job.NextRetryAt = parseTime(retryAt)
		_ = json.Unmarshal([]byte(conflictJSON), &job.ConflictIDs)
		result = append(result, job)
	}
	return result, rows.Err()
}

func (s *Store) PendingProtectionJobs(now time.Time) ([]models.SyncJob, error) {
	rows, err := s.db.Query(`
SELECT id, game_id, reason, state, stage, priority, revision_id, started_at, updated_at, finished_at, changed_files, bytes, progress, retry_count, next_retry_at, error_code, error_message, conflict_ids_json
FROM protection_jobs
WHERE state IN ('queued','waiting','waiting_for_network','paused','failed')
  AND (next_retry_at = '' OR next_retry_at <= ?)
ORDER BY priority DESC, updated_at ASC`, formatTime(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.SyncJob{}
	for rows.Next() {
		var job models.SyncJob
		var started, updated, finished, retryAt, conflictJSON string
		if err := rows.Scan(&job.ID, &job.GameID, &job.Reason, &job.State, &job.Stage, &job.Priority, &job.RevisionID, &started, &updated, &finished, &job.ChangedFiles, &job.Bytes, &job.Progress, &job.RetryCount, &retryAt, &job.ErrorCode, &job.Error, &conflictJSON); err != nil {
			return nil, err
		}
		job.StartedAt, job.UpdatedAt, job.FinishedAt, job.NextRetryAt = parseTime(started), parseTime(updated), parseTime(finished), parseTime(retryAt)
		_ = json.Unmarshal([]byte(conflictJSON), &job.ConflictIDs)
		result = append(result, job)
	}
	return result, rows.Err()
}

func (s *Store) PendingSummary(gameID string) (int, int64, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(bytes), 0) FROM protection_jobs WHERE state IN ('queued','waiting','waiting_for_network','paused','failed','authentication_required','quota_blocked')`
	args := []any{}
	if gameID != "" {
		query += ` AND game_id = ?`
		args = append(args, gameID)
	}
	var count int
	var bytes int64
	err := s.db.QueryRow(query, args...).Scan(&count, &bytes)
	return count, bytes, err
}

func (s *Store) SaveRevision(revision models.Revision) error {
	parents, _ := json.Marshal(revision.ParentIDs)
	_, err := s.db.Exec(`
INSERT INTO protection_revisions(id, game_id, sync_id, parent_ids_json, kind, name, created_at, file_count, total_bytes, manifest_json, pinned, uploaded)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET pinned=excluded.pinned, uploaded=excluded.uploaded, name=excluded.name`,
		revision.ID, revision.GameID, revision.SyncID, string(parents), revision.Kind, revision.Name,
		formatTime(revision.CreatedAt), revision.FileCount, revision.TotalBytes, revision.ManifestJSON,
		boolInt(revision.Pinned), boolInt(revision.Uploaded))
	return err
}

func (s *Store) Revision(id string) (models.Revision, error) {
	var value models.Revision
	var parents, created string
	var pinned, uploaded int
	err := s.db.QueryRow(`SELECT id, game_id, sync_id, parent_ids_json, kind, name, created_at, file_count, total_bytes, manifest_json, pinned, uploaded FROM protection_revisions WHERE id = ?`, id).
		Scan(&value.ID, &value.GameID, &value.SyncID, &parents, &value.Kind, &value.Name, &created, &value.FileCount, &value.TotalBytes, &value.ManifestJSON, &pinned, &uploaded)
	if err != nil {
		return value, err
	}
	value.CreatedAt, value.Pinned, value.Uploaded = parseTime(created), pinned != 0, uploaded != 0
	_ = json.Unmarshal([]byte(parents), &value.ParentIDs)
	return value, nil
}

func (s *Store) Revisions(gameID string, limit int) ([]models.Revision, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id, game_id, sync_id, parent_ids_json, kind, name, created_at, file_count, total_bytes, pinned, uploaded FROM protection_revisions WHERE game_id = ? ORDER BY created_at DESC LIMIT ?`, gameID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Revision{}
	for rows.Next() {
		var value models.Revision
		var parents, created string
		var pinned, uploaded int
		if err := rows.Scan(&value.ID, &value.GameID, &value.SyncID, &parents, &value.Kind, &value.Name, &created, &value.FileCount, &value.TotalBytes, &pinned, &uploaded); err != nil {
			return nil, err
		}
		value.CreatedAt, value.Pinned, value.Uploaded = parseTime(created), pinned != 0, uploaded != 0
		_ = json.Unmarshal([]byte(parents), &value.ParentIDs)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) MarkRevisionUploaded(id string) error {
	_, err := s.db.Exec(`UPDATE protection_revisions SET uploaded = 1 WHERE id = ?`, id)
	return err
}

// PruneAutomaticRevisions removes only unpinned automatic metadata beyond the
// configured visible retention. Immutable objects remain available to the
// quarantine sweep and are never deleted here.
func (s *Store) PruneAutomaticRevisions(gameID string, keep int) ([]string, error) {
	if keep < 1 {
		keep = 3
	}
	rows, err := s.db.Query(`
SELECT id FROM protection_revisions
WHERE game_id = ? AND pinned = 0 AND kind IN ('automatic','final')
ORDER BY created_at DESC`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) <= keep {
		return nil, rows.Err()
	}
	removed := ids[keep:]
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, id := range removed {
		if _, err := tx.Exec(`DELETE FROM protection_revisions WHERE id = ? AND pinned = 0 AND id NOT IN (SELECT revision_id FROM recovery_points)`, id); err != nil {
			return nil, err
		}
	}
	return removed, tx.Commit()
}

func (s *Store) CreateRecoveryPoint(point models.RecoveryPoint) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE protection_revisions SET pinned = 1, name = ? WHERE id = ?`, point.Name, point.RevisionID); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO recovery_points(id, game_id, revision_id, name, created_at) VALUES (?, ?, ?, ?, ?)`, point.ID, point.GameID, point.RevisionID, point.Name, formatTime(point.CreatedAt)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RecoveryPoints(gameID string) ([]models.RecoveryPoint, error) {
	rows, err := s.db.Query(`
SELECT p.id, p.game_id, p.revision_id, p.name, p.created_at, r.file_count, r.total_bytes
FROM recovery_points p JOIN protection_revisions r ON r.id = p.revision_id
WHERE p.game_id = ? ORDER BY p.created_at DESC`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.RecoveryPoint{}
	for rows.Next() {
		var value models.RecoveryPoint
		var created string
		if err := rows.Scan(&value.ID, &value.GameID, &value.RevisionID, &value.Name, &created, &value.FileCount, &value.TotalBytes); err != nil {
			return nil, err
		}
		value.CreatedAt = parseTime(created)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) RecoveryPoint(id string) (models.RecoveryPoint, error) {
	var value models.RecoveryPoint
	var created string
	err := s.db.QueryRow(`
SELECT p.id, p.game_id, p.revision_id, p.name, p.created_at, r.file_count, r.total_bytes
FROM recovery_points p JOIN protection_revisions r ON r.id = p.revision_id
WHERE p.id = ?`, id).Scan(&value.ID, &value.GameID, &value.RevisionID, &value.Name, &created, &value.FileCount, &value.TotalBytes)
	value.CreatedAt = parseTime(created)
	return value, err
}

func (s *Store) DeleteRecoveryPoint(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var revisionID string
	if err := tx.QueryRow(`SELECT revision_id FROM recovery_points WHERE id = ?`, id).Scan(&revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM recovery_points WHERE id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE protection_revisions SET pinned = 0, name = '' WHERE id = ?`, revisionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SaveProviderHealth(provider string, value models.ProviderHealth) error {
	_, err := s.db.Exec(`
INSERT INTO provider_health(provider, state, last_success_at, last_attempt_at, next_retry_at, error_code, message, action)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider) DO UPDATE SET state=excluded.state, last_success_at=excluded.last_success_at,
 last_attempt_at=excluded.last_attempt_at, next_retry_at=excluded.next_retry_at,
 error_code=excluded.error_code, message=excluded.message, action=excluded.action`,
		provider, value.State, formatTime(value.LastSuccessAt), formatTime(value.LastAttemptAt), formatTime(value.NextRetryAt), value.ErrorCode, value.Message, value.Action)
	return err
}

func (s *Store) ProviderHealth(provider string) (models.ProviderHealth, error) {
	var value models.ProviderHealth
	var success, attempt, retry string
	err := s.db.QueryRow(`SELECT state, last_success_at, last_attempt_at, next_retry_at, error_code, message, action FROM provider_health WHERE provider = ?`, provider).
		Scan(&value.State, &success, &attempt, &retry, &value.ErrorCode, &value.Message, &value.Action)
	if errors.Is(err, sql.ErrNoRows) {
		return value, nil
	}
	value.LastSuccessAt, value.LastAttemptAt, value.NextRetryAt = parseTime(success), parseTime(attempt), parseTime(retry)
	return value, err
}

func (s *Store) SaveServiceControl(key string, value models.ServiceControlState) error {
	if key == "" {
		key = "global"
	}
	_, err := s.db.Exec(`
INSERT INTO service_control(scope_key, protection_enabled, pause_scope, pause_until, pause_until_login, reason, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scope_key) DO UPDATE SET protection_enabled=excluded.protection_enabled, pause_scope=excluded.pause_scope,
 pause_until=excluded.pause_until, pause_until_login=excluded.pause_until_login, reason=excluded.reason, updated_at=excluded.updated_at`,
		key, boolInt(value.ProtectionEnabled), value.PauseScope, formatTime(value.PauseUntil), boolInt(value.PauseUntilLogin), value.Reason, formatTime(time.Now().UTC()))
	return err
}

func (s *Store) ServiceControl(key string, fallback models.ServiceControlState) (models.ServiceControlState, error) {
	if key == "" {
		key = "global"
	}
	var enabled, untilLogin int
	var pause, until, reason string
	err := s.db.QueryRow(`SELECT protection_enabled, pause_scope, pause_until, pause_until_login, reason FROM service_control WHERE scope_key = ?`, key).
		Scan(&enabled, &pause, &until, &untilLogin, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return fallback, nil
	}
	if err != nil {
		return fallback, err
	}
	return models.ServiceControlState{ProtectionEnabled: enabled != 0, PauseScope: models.PauseScope(pause), PauseUntil: parseTime(until), PauseUntilLogin: untilLogin != 0, Reason: reason}, nil
}

func (s *Store) WriteActivity(event models.ActivityEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = "info"
	}
	if event.DeduplicationKey != "" {
		_, err := s.db.Exec(`
INSERT INTO activity_events(category, severity, game_id, title, detail, ts, action_target, dedupe_key, resolved)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)
ON CONFLICT(dedupe_key) WHERE dedupe_key <> '' AND resolved = 0 DO UPDATE SET
 category=excluded.category, severity=excluded.severity, game_id=excluded.game_id,
 title=excluded.title, detail=excluded.detail, ts=excluded.ts, action_target=excluded.action_target`,
			event.Category, event.Severity, event.GameID, event.Title, event.Detail, formatTime(event.Timestamp), event.ActionTarget, event.DeduplicationKey)
		return err
	}
	_, err := s.db.Exec(`INSERT INTO activity_events(category, severity, game_id, title, detail, ts, action_target, dedupe_key, resolved) VALUES (?, ?, ?, ?, ?, ?, ?, '', ?)`,
		event.Category, event.Severity, event.GameID, event.Title, event.Detail, formatTime(event.Timestamp), event.ActionTarget, boolInt(event.Resolved))
	return err
}

func (s *Store) Activity(limit int) ([]models.ActivityEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	cutoff := formatTime(time.Now().UTC().Add(-90 * 24 * time.Hour))
	rows, err := s.db.Query(`SELECT id, category, severity, game_id, title, detail, ts, action_target, resolved FROM activity_events WHERE ts >= ? ORDER BY ts DESC, id DESC LIMIT ?`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ActivityEvent{}
	for rows.Next() {
		var value models.ActivityEvent
		var ts string
		var resolved int
		if err := rows.Scan(&value.ID, &value.Category, &value.Severity, &value.GameID, &value.Title, &value.Detail, &ts, &value.ActionTarget, &resolved); err != nil {
			return nil, err
		}
		value.Timestamp, value.Resolved = parseTime(ts), resolved != 0
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) PruneActivity() error {
	cutoff := formatTime(time.Now().UTC().Add(-90 * 24 * time.Hour))
	_, err := s.db.Exec(`
DELETE FROM activity_events
WHERE ts < ? OR id NOT IN (SELECT id FROM activity_events ORDER BY ts DESC, id DESC LIMIT 500)`, cutoff)
	return err
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	if parsed.IsZero() {
		parsed, _ = time.Parse(time.RFC3339, value)
	}
	return parsed
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
