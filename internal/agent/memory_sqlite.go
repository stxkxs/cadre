package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stxkxs/cadre/internal/provider"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteMemoryStore persists agent memory in a SQLite database.
type SQLiteMemoryStore struct {
	db *sql.DB
}

// NewSQLiteMemoryStore opens (or creates) the SQLite database at path.
func NewSQLiteMemoryStore(path string) (*SQLiteMemoryStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory database: %w", err)
	}

	s := &SQLiteMemoryStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate memory database: %w", err)
	}

	return s, nil
}

func (s *SQLiteMemoryStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		namespace TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		tool_name TEXT,
		tool_input TEXT,
		tool_call_id TEXT,
		timestamp DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_agent_memory_namespace ON agent_memory(namespace);
	CREATE INDEX IF NOT EXISTS idx_agent_memory_timestamp ON agent_memory(namespace, timestamp);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Additive migration: add content_blocks column for tool_use/tool_result flows.
	// ALTER TABLE ADD COLUMN is a no-op if the column already exists in SQLite (we catch the error).
	_, _ = s.db.Exec(`ALTER TABLE agent_memory ADD COLUMN content_blocks TEXT`)

	return nil
}

// Save persists a message under the given namespace.
func (s *SQLiteMemoryStore) Save(namespace string, msg Message) error {
	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	var blocksJSON *string
	if len(msg.ContentBlocks) > 0 {
		data, err := json.Marshal(msg.ContentBlocks)
		if err != nil {
			return fmt.Errorf("marshal content blocks: %w", err)
		}
		s := string(data)
		blocksJSON = &s
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_memory (namespace, role, content, tool_name, tool_input, tool_call_id, timestamp, content_blocks)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, namespace, msg.Role, msg.Content, msg.ToolName, msg.ToolInput, msg.ToolCallID, ts, blocksJSON)
	return err
}

// Load returns the most recent n messages for the namespace.
func (s *SQLiteMemoryStore) Load(namespace string, n int) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT role, content, tool_name, tool_input, tool_call_id, timestamp, content_blocks
		FROM agent_memory
		WHERE namespace = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT ?
	`, namespace, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var blocksJSON sql.NullString
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolName, &m.ToolInput, &m.ToolCallID, &m.Timestamp, &blocksJSON); err != nil {
			return nil, err
		}
		if blocksJSON.Valid && blocksJSON.String != "" {
			var blocks []provider.ContentBlock
			if err := json.Unmarshal([]byte(blocksJSON.String), &blocks); err == nil {
				m.ContentBlocks = blocks
			}
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse so oldest is first (we queried DESC for LIMIT).
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}

// Search returns messages whose content matches the pattern (case-insensitive LIKE).
func (s *SQLiteMemoryStore) Search(namespace string, pattern string) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT role, content, tool_name, tool_input, tool_call_id, timestamp, content_blocks
		FROM agent_memory
		WHERE namespace = ? AND content LIKE ?
		ORDER BY timestamp ASC
	`, namespace, "%"+pattern+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var blocksJSON sql.NullString
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolName, &m.ToolInput, &m.ToolCallID, &m.Timestamp, &blocksJSON); err != nil {
			return nil, err
		}
		if blocksJSON.Valid && blocksJSON.String != "" {
			var blocks []provider.ContentBlock
			if err := json.Unmarshal([]byte(blocksJSON.String), &blocks); err == nil {
				m.ContentBlocks = blocks
			}
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// Clear deletes all messages for the namespace.
func (s *SQLiteMemoryStore) Clear(namespace string) error {
	_, err := s.db.Exec("DELETE FROM agent_memory WHERE namespace = ?", namespace)
	return err
}

// Close closes the database connection.
func (s *SQLiteMemoryStore) Close() error {
	return s.db.Close()
}
