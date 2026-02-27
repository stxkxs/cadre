package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store defines the interface for state storage backends
type Store interface {
	SaveRun(run *RunState) error
	GetRun(id string) (*RunState, error)
	ListRuns(limit int) ([]*RunState, error)
	DeleteRun(id string) error

	SaveCheckpoint(cp *Checkpoint) error
	GetCheckpoint(id string) (*Checkpoint, error)
	ListCheckpoints(runID string) ([]*Checkpoint, error)
	DeleteCheckpoint(id string) error

	Close() error
}

// Manager manages execution state
type Manager struct {
	store     Store
	mu        sync.RWMutex
	activeRun *RunState
}

// NewManager creates a new state manager
func NewManager(driver, path string) (*Manager, error) {
	var store Store
	var err error

	switch driver {
	case "memory", "":
		store = NewMemoryStore()
	case "sqlite":
		store, err = NewSQLiteStore(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create sqlite store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported state driver: %s", driver)
	}

	return &Manager{store: store}, nil
}

// Close closes the state manager
func (m *Manager) Close() error {
	return m.store.Close()
}

// StartRun creates and returns a new run state
func (m *Manager) StartRun(crewName string, inputs map[string]interface{}) (*RunState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run := NewRunState(uuid.New().String(), crewName)
	run.Status = "running"
	run.Inputs = inputs

	if err := m.store.SaveRun(run); err != nil {
		return nil, fmt.Errorf("failed to save run: %w", err)
	}

	m.activeRun = run
	return run, nil
}

// UpdateRun updates the current run state
func (m *Manager) UpdateRun(run *RunState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.SaveRun(run); err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}

	m.activeRun = run
	return nil
}

// CompleteRun marks the run as complete
func (m *Manager) CompleteRun(outputs map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeRun == nil {
		return fmt.Errorf("no active run")
	}

	m.activeRun.Status = "completed"
	m.activeRun.CompletedAt = time.Now()
	m.activeRun.Outputs = outputs

	if err := m.store.SaveRun(m.activeRun); err != nil {
		return fmt.Errorf("failed to save run: %w", err)
	}

	return nil
}

// FailRun marks the run as failed
func (m *Manager) FailRun(err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeRun == nil {
		return fmt.Errorf("no active run")
	}

	m.activeRun.Status = "failed"
	m.activeRun.CompletedAt = time.Now()
	m.activeRun.Error = err.Error()

	if saveErr := m.store.SaveRun(m.activeRun); saveErr != nil {
		return fmt.Errorf("failed to save run: %w", saveErr)
	}

	return nil
}

// GetActiveRun returns the current active run
func (m *Manager) GetActiveRun() (*RunState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeRun != nil {
		return m.activeRun, nil
	}

	// Try to find an active run in storage
	runs, err := m.store.ListRuns(1)
	if err != nil {
		return nil, err
	}

	for _, run := range runs {
		if run.Status == "running" {
			return run, nil
		}
	}

	return nil, nil
}

// ListRuns lists recent runs
func (m *Manager) ListRuns(limit int) ([]*RunState, error) {
	return m.store.ListRuns(limit)
}

// SaveCheckpoint creates a checkpoint of the current state
func (m *Manager) SaveCheckpoint(state *RunState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:        uuid.New().String(),
		RunID:     state.ID,
		CreatedAt: time.Now(),
		State:     *state,
	}

	// Find current task
	for _, task := range state.Tasks {
		if task.Status == "running" {
			checkpoint.CurrentTask = task.Name
			break
		}
	}

	if err := m.store.SaveCheckpoint(checkpoint); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Also save to file for easy recovery
	return m.saveCheckpointFile(checkpoint)
}

// LoadCheckpoint loads a checkpoint by ID
func (m *Manager) LoadCheckpoint(id string) (*Checkpoint, error) {
	return m.store.GetCheckpoint(id)
}

// saveCheckpointFile saves checkpoint to a file for recovery
func (m *Manager) saveCheckpointFile(cp *Checkpoint) error {
	dir := ".cadre/checkpoints"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoints dir: %w", err)
	}

	filename := filepath.Join(dir, cp.ID+".json")
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// UpdateTaskState updates the state of a specific task
func (m *Manager) UpdateTaskState(taskName string, status string, outputs map[string]interface{}, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeRun == nil {
		return fmt.Errorf("no active run")
	}

	task := m.activeRun.GetTask(taskName)
	if task == nil {
		return fmt.Errorf("task not found: %s", taskName)
	}

	task.Status = status
	if status == "running" {
		task.StartedAt = time.Now()
	}
	if status == "completed" || status == "failed" {
		task.CompletedAt = time.Now()
	}
	if outputs != nil {
		task.Outputs = outputs
	}
	if err != nil {
		task.Error = err.Error()
	}

	m.activeRun.UpdateTask(*task)
	return m.store.SaveRun(m.activeRun)
}

// AddAgentMemory stores agent memory for checkpointing
func (m *Manager) AddAgentMemory(agentName string, messages []Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store in metadata for now
	if m.activeRun == nil {
		return nil
	}

	if m.activeRun.Metadata == nil {
		m.activeRun.Metadata = make(map[string]interface{})
	}

	key := "agent_memory_" + agentName
	m.activeRun.Metadata[key] = messages

	return nil
}

// SetMetadata sets a key-value pair on the active run's metadata.
func (m *Manager) SetMetadata(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeRun == nil {
		return
	}
	if m.activeRun.Metadata == nil {
		m.activeRun.Metadata = make(map[string]interface{})
	}
	m.activeRun.Metadata[key] = value
}

// GetAgentMemory retrieves stored agent memory
func (m *Manager) GetAgentMemory(agentName string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeRun == nil || m.activeRun.Metadata == nil {
		return nil, nil
	}

	key := "agent_memory_" + agentName
	if mem, ok := m.activeRun.Metadata[key]; ok {
		// Type assertion
		if messages, ok := mem.([]Message); ok {
			return messages, nil
		}
	}

	return nil, nil
}
