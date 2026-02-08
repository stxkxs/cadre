package state

import (
	"fmt"
	"sort"
	"sync"
)

// MemoryStore implements an in-memory state store
type MemoryStore struct {
	mu          sync.RWMutex
	runs        map[string]*RunState
	checkpoints map[string]*Checkpoint
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs:        make(map[string]*RunState),
		checkpoints: make(map[string]*Checkpoint),
	}
}

// SaveRun saves a run state
func (s *MemoryStore) SaveRun(run *RunState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
	return nil
}

// GetRun retrieves a run state
func (s *MemoryStore) GetRun(id string) (*RunState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if run, ok := s.runs[id]; ok {
		return run, nil
	}
	return nil, fmt.Errorf("run not found: %s", id)
}

// ListRuns lists recent runs
func (s *MemoryStore) ListRuns(limit int) ([]*RunState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]*RunState, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, run)
	}

	// Sort by start time descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})

	if len(runs) > limit {
		runs = runs[:limit]
	}

	return runs, nil
}

// DeleteRun deletes a run
func (s *MemoryStore) DeleteRun(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, id)
	return nil
}

// SaveCheckpoint saves a checkpoint
func (s *MemoryStore) SaveCheckpoint(cp *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints[cp.ID] = cp
	return nil
}

// GetCheckpoint retrieves a checkpoint
func (s *MemoryStore) GetCheckpoint(id string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cp, ok := s.checkpoints[id]; ok {
		return cp, nil
	}
	return nil, fmt.Errorf("checkpoint not found: %s", id)
}

// ListCheckpoints lists checkpoints for a run
func (s *MemoryStore) ListCheckpoints(runID string) ([]*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cps := make([]*Checkpoint, 0)
	for _, cp := range s.checkpoints {
		if cp.RunID == runID {
			cps = append(cps, cp)
		}
	}

	// Sort by creation time descending
	sort.Slice(cps, func(i, j int) bool {
		return cps[i].CreatedAt.After(cps[j].CreatedAt)
	})

	return cps, nil
}

// DeleteCheckpoint deletes a checkpoint
func (s *MemoryStore) DeleteCheckpoint(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.checkpoints, id)
	return nil
}

// Close closes the store (no-op for memory)
func (s *MemoryStore) Close() error {
	return nil
}
