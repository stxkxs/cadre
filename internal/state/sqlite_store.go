package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements state storage using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// migrate creates the necessary tables
func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		crew_name TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		error TEXT,
		data JSON NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
	CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		data JSON NOT NULL,
		FOREIGN KEY (run_id) REFERENCES runs(id)
	);

	CREATE INDEX IF NOT EXISTS idx_checkpoints_run_id ON checkpoints(run_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveRun saves a run state
func (s *SQLiteStore) SaveRun(run *RunState) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO runs (id, crew_name, status, started_at, completed_at, error, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.CrewName, run.Status, run.StartedAt, run.CompletedAt, run.Error, data)

	return err
}

// GetRun retrieves a run state
func (s *SQLiteStore) GetRun(id string) (*RunState, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM runs WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	var run RunState
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal run: %w", err)
	}

	return &run, nil
}

// ListRuns lists recent runs
func (s *SQLiteStore) ListRuns(limit int) ([]*RunState, error) {
	rows, err := s.db.Query(`
		SELECT data FROM runs
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*RunState
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}

		var run RunState
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		runs = append(runs, &run)
	}

	return runs, rows.Err()
}

// DeleteRun deletes a run
func (s *SQLiteStore) DeleteRun(id string) error {
	_, err := s.db.Exec("DELETE FROM runs WHERE id = ?", id)
	return err
}

// SaveCheckpoint saves a checkpoint
func (s *SQLiteStore) SaveCheckpoint(cp *Checkpoint) error {
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO checkpoints (id, run_id, created_at, data)
		VALUES (?, ?, ?, ?)
	`, cp.ID, cp.RunID, cp.CreatedAt, data)

	return err
}

// GetCheckpoint retrieves a checkpoint
func (s *SQLiteStore) GetCheckpoint(id string) (*Checkpoint, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM checkpoints WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("checkpoint not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &cp, nil
}

// ListCheckpoints lists checkpoints for a run
func (s *SQLiteStore) ListCheckpoints(runID string) ([]*Checkpoint, error) {
	rows, err := s.db.Query(`
		SELECT data FROM checkpoints
		WHERE run_id = ?
		ORDER BY created_at DESC
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cps []*Checkpoint
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}

		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		cps = append(cps, &cp)
	}

	return cps, rows.Err()
}

// DeleteCheckpoint deletes a checkpoint
func (s *SQLiteStore) DeleteCheckpoint(id string) error {
	_, err := s.db.Exec("DELETE FROM checkpoints WHERE id = ?", id)
	return err
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
