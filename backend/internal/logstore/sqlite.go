package logstore

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"cloud-saver/backend/internal/models"
)

type Store struct {
	db   *sql.DB
	path string
}

func Open(root string) (*Store, error) {
	dbPath := filepath.Join(root, "sync.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db, path: dbPath}, nil
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
	_, err := s.db.Exec(`INSERT INTO logs(ts, level, event, message) VALUES (?, ?, ?, ?)`, time.Now().UTC().Format(time.RFC3339), level, event, message)
	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}
	return nil
}

func (s *Store) WriteConflict(gameID, filePath, strategy, message string) error {
	_, err := s.db.Exec(`INSERT INTO conflicts(ts, game_id, file_path, strategy, message) VALUES (?, ?, ?, ?, ?)`, time.Now().UTC().Format(time.RFC3339), gameID, filePath, strategy, message)
	if err != nil {
		return fmt.Errorf("insert conflict: %w", err)
	}
	return nil
}

func (s *Store) Recent(limit int) ([]models.LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := s.db.Query(`SELECT id, ts, level, event, message FROM logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
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
