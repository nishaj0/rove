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
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	return nil
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

func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy")
}
