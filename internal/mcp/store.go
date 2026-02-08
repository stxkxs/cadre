package mcp

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Store provides SQLite-backed message queue and agent registry for IPC.
// All instances sharing the same DB file communicate through WAL mode.
type Store struct {
	db *sql.DB
}

// Message represents an inter-agent message.
type Message struct {
	ID        string    `json:"id"`
	FromAgent string    `json:"from_agent"`
	ToAgent   string    `json:"to_agent"`
	Message   string    `json:"message"`
	Priority  string    `json:"priority"`
	Status    string    `json:"status"`
	Response  string    `json:"response,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AgentRecord represents an agent's registration in the session.
type AgentRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	PID           int       `json:"pid,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	CurrentTask   string    `json:"current_task,omitempty"`
	Summary       string    `json:"summary,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// NewStore creates a new IPC store backed by the given SQLite file.
// The directory is created if it doesn't exist. WAL mode is enabled.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		from_agent TEXT NOT NULL,
		to_agent TEXT NOT NULL,
		message TEXT NOT NULL,
		priority TEXT DEFAULT 'normal',
		status TEXT DEFAULT 'pending',
		response TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_to_agent ON messages(to_agent);
	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);

	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		status TEXT DEFAULT 'starting',
		pid INTEGER,
		session_id TEXT,
		current_task TEXT,
		summary TEXT,
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SendMessage inserts a new message from one agent to another.
func (s *Store) SendMessage(from, to, message, priority string) (string, error) {
	if priority == "" {
		priority = "normal"
	}
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT INTO messages (id, from_agent, to_agent, message, priority) VALUES (?, ?, ?, ?, ?)`,
		id, from, to, message, priority,
	)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	return id, nil
}

// CheckMessages returns pending messages for the given agent.
func (s *Store) CheckMessages(agentID string) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, from_agent, to_agent, message, priority, status, COALESCE(response, ''), created_at
		 FROM messages WHERE to_agent = ? AND status = 'pending'
		 ORDER BY created_at ASC`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("check messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.FromAgent, &m.ToAgent, &m.Message, &m.Priority, &m.Status, &m.Response, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// RespondToMessage sets a response on a message and marks it as responded.
func (s *Store) RespondToMessage(messageID, response, status string) error {
	if status == "" {
		status = "responded"
	}
	result, err := s.db.Exec(
		`UPDATE messages SET response = ?, status = ? WHERE id = ?`,
		response, status, messageID,
	)
	if err != nil {
		return fmt.Errorf("respond to message: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("message not found: %s", messageID)
	}
	return nil
}

// RegisterAgent registers or updates an agent in the session.
func (s *Store) RegisterAgent(id, name string, pid int, sessionID string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO agents (id, name, status, pid, session_id, last_heartbeat)
		 VALUES (?, ?, 'starting', ?, ?, CURRENT_TIMESTAMP)`,
		id, name, pid, sessionID,
	)
	if err != nil {
		return fmt.Errorf("register agent: %w", err)
	}
	return nil
}

// ListAgents returns all agents in the session.
func (s *Store) ListAgents() ([]AgentRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, COALESCE(pid, 0), COALESCE(session_id, ''),
		        COALESCE(current_task, ''), COALESCE(summary, ''), last_heartbeat
		 FROM agents ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var agents []AgentRecord
	for rows.Next() {
		var a AgentRecord
		if err := rows.Scan(&a.ID, &a.Name, &a.Status, &a.PID, &a.SessionID, &a.CurrentTask, &a.Summary, &a.LastHeartbeat); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// GetAgent returns a single agent by ID.
func (s *Store) GetAgent(agentID string) (*AgentRecord, error) {
	var a AgentRecord
	err := s.db.QueryRow(
		`SELECT id, name, status, COALESCE(pid, 0), COALESCE(session_id, ''),
		        COALESCE(current_task, ''), COALESCE(summary, ''), last_heartbeat
		 FROM agents WHERE id = ?`,
		agentID,
	).Scan(&a.ID, &a.Name, &a.Status, &a.PID, &a.SessionID, &a.CurrentTask, &a.Summary, &a.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return &a, nil
}

// UpdateAgentStatus updates an agent's status and optional detail.
func (s *Store) UpdateAgentStatus(agentID, status, detail string) error {
	result, err := s.db.Exec(
		`UPDATE agents SET status = ?, current_task = ?, last_heartbeat = CURRENT_TIMESTAMP WHERE id = ?`,
		status, detail, agentID,
	)
	if err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	return nil
}

// MarkAgentComplete marks an agent as complete with a summary.
func (s *Store) MarkAgentComplete(agentID, summary string) error {
	result, err := s.db.Exec(
		`UPDATE agents SET status = 'complete', summary = ?, last_heartbeat = CURRENT_TIMESTAMP WHERE id = ?`,
		summary, agentID,
	)
	if err != nil {
		return fmt.Errorf("mark agent complete: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	return nil
}

// Heartbeat updates the agent's last heartbeat timestamp.
func (s *Store) Heartbeat(agentID string) error {
	_, err := s.db.Exec(
		`UPDATE agents SET last_heartbeat = CURRENT_TIMESTAMP WHERE id = ?`,
		agentID,
	)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
