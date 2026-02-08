package event

import "time"

// EventType identifies the kind of lifecycle event.
type EventType string

const (
	// Crew lifecycle
	CrewStarted   EventType = "crew.started"
	CrewCompleted EventType = "crew.completed"
	CrewFailed    EventType = "crew.failed"

	// Task lifecycle
	TaskStarted   EventType = "task.started"
	TaskCompleted EventType = "task.completed"
	TaskFailed    EventType = "task.failed"
	TaskRetrying  EventType = "task.retrying"

	// Agent lifecycle
	AgentToolCall   EventType = "agent.tool.call"
	AgentToolResult EventType = "agent.tool.result"

	// Manager lifecycle (hierarchical process)
	ManagerDelegated EventType = "manager.delegated"
	ManagerReviewed  EventType = "manager.reviewed"
	ManagerFeedback  EventType = "manager.feedback"

	// State
	StateCheckpoint EventType = "state.checkpoint"
)

// Event carries data about a lifecycle occurrence.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewEvent creates an event with the current timestamp.
func NewEvent(t EventType, data map[string]interface{}) Event {
	return Event{
		Type:      t,
		Timestamp: time.Now(),
		Data:      data,
	}
}
