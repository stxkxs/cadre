package task

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stxkxs/cadre/internal/config"
)

// Task represents a unit of work
type Task struct {
	config *config.TaskConfig

	mu sync.RWMutex // protects all runtime state below

	// Runtime state
	Status    string
	StartedAt time.Time
	EndedAt   time.Time
	Inputs    map[string]interface{}
	Outputs   map[string]interface{}
	Error     error
	Attempts  int
}

// NewTask creates a new task from configuration
func NewTask(cfg *config.TaskConfig) *Task {
	return &Task{
		config:  cfg,
		Status:  "pending",
		Inputs:  make(map[string]interface{}),
		Outputs: make(map[string]interface{}),
	}
}

// Name returns the task name
func (t *Task) Name() string {
	return t.config.Name
}

// Description returns the task description
func (t *Task) Description() string {
	return t.config.Description
}

// Agent returns the agent name assigned to this task
func (t *Task) Agent() string {
	return t.config.Agent
}

// Dependencies returns the list of tasks this depends on
func (t *Task) Dependencies() []string {
	return t.config.Dependencies
}

// Timeout returns the task timeout
func (t *Task) Timeout() (time.Duration, error) {
	return t.config.ParsedTimeout()
}

// MaxAttempts returns the maximum retry attempts
func (t *Task) MaxAttempts() int {
	if t.config.Retry.MaxAttempts == 0 {
		return 1
	}
	return t.config.Retry.MaxAttempts
}

// InputDefinitions returns the input definitions
func (t *Task) InputDefinitions() []config.InputConfig {
	return t.config.Inputs
}

// OutputDefinitions returns the output definitions
func (t *Task) OutputDefinitions() []config.OutputConfig {
	return t.config.Outputs
}

// SetInput sets an input value
func (t *Task) SetInput(name string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Inputs[name] = value
}

// GetInput gets an input value
func (t *Task) GetInput(name string) (interface{}, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	val, ok := t.Inputs[name]
	return val, ok
}

// SetOutput sets an output value
func (t *Task) SetOutput(name string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Outputs[name] = value
}

// GetOutput gets an output value
func (t *Task) GetOutput(name string) (interface{}, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	val, ok := t.Outputs[name]
	return val, ok
}

// GetStatus returns the current status (thread-safe).
func (t *Task) GetStatus() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// GetOutputs returns a copy of the outputs map (thread-safe).
func (t *Task) GetOutputs() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cp := make(map[string]interface{}, len(t.Outputs))
	for k, v := range t.Outputs {
		cp[k] = v
	}
	return cp
}

// Start marks the task as started
func (t *Task) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = "running"
	t.StartedAt = time.Now()
	t.Attempts++
}

// Complete marks the task as completed
func (t *Task) Complete(outputs map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = "completed"
	t.EndedAt = time.Now()
	if outputs != nil {
		t.Outputs = outputs
	}
}

// Fail marks the task as failed
func (t *Task) Fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = "failed"
	t.EndedAt = time.Now()
	t.Error = err
}

// Skip marks the task as skipped
func (t *Task) Skip(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = "skipped"
	t.EndedAt = time.Now()
	t.Error = nil
}

// CanRetry returns true if the task can be retried
func (t *Task) CanRetry() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Attempts < t.MaxAttempts()
}

// Duration returns how long the task ran
func (t *Task) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.StartedAt.IsZero() {
		return 0
	}
	if t.EndedAt.IsZero() {
		return time.Since(t.StartedAt)
	}
	return t.EndedAt.Sub(t.StartedAt)
}

// BuildPrompt builds the execution prompt for the agent
func (t *Task) BuildPrompt() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	prompt := "## Task: " + t.Name() + "\n\n"
	prompt += t.Description() + "\n\n"

	if len(t.Inputs) > 0 {
		prompt += "### Inputs:\n"
		for name, value := range t.Inputs {
			prompt += "- " + name + ": " + formatValue(value) + "\n"
		}
		prompt += "\n"
	}

	// Include manager instructions if present (for hierarchical process).
	if instr, ok := t.Inputs["_manager_instructions"]; ok {
		prompt += "### Manager Instructions:\n" + formatValue(instr) + "\n\n"
	}
	if fb, ok := t.Inputs["_manager_feedback"]; ok {
		prompt += "### Manager Feedback:\n" + formatValue(fb) + "\n\n"
	}

	if len(t.OutputDefinitions()) > 0 {
		prompt += "### Expected Outputs:\n"
		prompt += "You MUST include a JSON code block at the end of your response with these fields:\n\n"
		prompt += "```json\n{\n"
		fields := make([]string, 0, len(t.OutputDefinitions()))
		for _, out := range t.OutputDefinitions() {
			fields = append(fields, fmt.Sprintf("  %q: <%s>", out.Name, outputTypePlaceholder(out.Type)))
		}
		prompt += strings.Join(fields, ",\n") + "\n}\n```\n\n"
		prompt += "Provide your reasoning and work above the JSON block.\n"
	} else {
		prompt += "Please complete this task and provide the outputs in your response.\n"
	}

	return prompt
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []string:
		return "[" + joinStrings(val, ", ") + "]"
	default:
		return toString(v)
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// outputTypePlaceholder returns a human-readable placeholder for a given type.
func outputTypePlaceholder(typ string) string {
	switch typ {
	case "string":
		return "string"
	case "string[]":
		return `["string", ...]`
	case "int":
		return "integer"
	case "bool":
		return "true/false"
	default:
		return typ
	}
}
