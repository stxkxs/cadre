package state

import (
	"time"
)

// RunState represents the state of a crew run
type RunState struct {
	ID          string            `json:"id"`
	CrewName    string            `json:"crew_name"`
	Status      string            `json:"status"` // pending, running, completed, failed, cancelled
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt time.Time         `json:"completed_at,omitempty"`
	Error       string            `json:"error,omitempty"`
	Tasks       []TaskState       `json:"tasks"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskState represents the state of a task within a run
type TaskState struct {
	Name        string            `json:"name"`
	Agent       string            `json:"agent"`
	Status      string            `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   time.Time         `json:"started_at,omitempty"`
	CompletedAt time.Time         `json:"completed_at,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Error       string            `json:"error,omitempty"`
	Attempts    int               `json:"attempts"`
}

// Checkpoint represents a saved execution state for resumption
type Checkpoint struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	CreatedAt   time.Time `json:"created_at"`
	State       RunState  `json:"state"`

	// For resumption
	CurrentTask string            `json:"current_task"`
	Context     map[string]interface{} `json:"context,omitempty"`

	// Agent states for memory restoration
	AgentStates map[string]AgentState `json:"agent_states,omitempty"`
}

// AgentState represents saved agent state
type AgentState struct {
	Name     string            `json:"name"`
	Memory   []Message         `json:"memory,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role      string    `json:"role"` // system, user, assistant, tool
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`

	// For tool messages
	ToolName  string `json:"tool_name,omitempty"`
	ToolInput string `json:"tool_input,omitempty"`
}

// NewRunState creates a new run state
func NewRunState(id, crewName string) *RunState {
	return &RunState{
		ID:        id,
		CrewName:  crewName,
		Status:    "pending",
		StartedAt: time.Now(),
		Tasks:     []TaskState{},
		Inputs:    make(map[string]interface{}),
		Outputs:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}
}

// GetTask returns a task state by name
func (r *RunState) GetTask(name string) *TaskState {
	for i := range r.Tasks {
		if r.Tasks[i].Name == name {
			return &r.Tasks[i]
		}
	}
	return nil
}

// UpdateTask updates or adds a task state
func (r *RunState) UpdateTask(task TaskState) {
	for i := range r.Tasks {
		if r.Tasks[i].Name == task.Name {
			r.Tasks[i] = task
			return
		}
	}
	r.Tasks = append(r.Tasks, task)
}

// IsComplete returns true if all tasks are complete or failed
func (r *RunState) IsComplete() bool {
	for _, task := range r.Tasks {
		if task.Status == "pending" || task.Status == "running" {
			return false
		}
	}
	return true
}

// NextPendingTask returns the next task that can be executed
func (r *RunState) NextPendingTask(dependencies map[string][]string) *TaskState {
	for i := range r.Tasks {
		task := &r.Tasks[i]
		if task.Status != "pending" {
			continue
		}

		// Check if all dependencies are complete
		deps, ok := dependencies[task.Name]
		if !ok {
			return task
		}

		allComplete := true
		for _, dep := range deps {
			depTask := r.GetTask(dep)
			if depTask == nil || depTask.Status != "completed" {
				allComplete = false
				break
			}
		}

		if allComplete {
			return task
		}
	}
	return nil
}
