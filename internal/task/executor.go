package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// TaskExecutor defines the interface for task execution.
// This enables mock executors in integration tests.
type TaskExecutor interface {
	Execute(ctx context.Context, task *Task, runtime *agent.Runtime) error
}

// Executor handles task execution
type Executor struct {
	config  *config.Config
	logger  *telemetry.Logger
	metrics *telemetry.Metrics
}

// NewExecutor creates a new task executor
func NewExecutor(cfg *config.Config, logger *telemetry.Logger) *Executor {
	return &Executor{
		config:  cfg,
		logger:  logger,
		metrics: telemetry.NewMetrics(),
	}
}

// Execute runs a task with the given agent runtime
func (e *Executor) Execute(ctx context.Context, task *Task, runtime *agent.Runtime) error {
	e.logger.Info("Executing task", "task", task.Name(), "agent", task.Agent())
	e.metrics.IncTasksStarted()

	task.Start()

	// Get timeout
	timeout, err := task.Timeout()
	if err != nil {
		timeout = 30 * time.Minute
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build prompt
	prompt := task.BuildPrompt()

	// Execute with retries
	var lastErr error
	for task.Attempts <= task.MaxAttempts() {
		e.logger.Debug("Task attempt", "task", task.Name(), "attempt", task.Attempts)

		result, err := runtime.Execute(ctx, prompt)
		if err != nil {
			lastErr = err
			e.logger.Warn("Task attempt failed", "task", task.Name(), "error", err)

			if !task.CanRetry() {
				break
			}

			// Backoff before retry
			e.backoff(task.Attempts)
			task.Attempts++
			continue
		}

		// Parse outputs from result
		outputs := e.parseOutputs(result, task.OutputDefinitions())
		task.Complete(outputs)

		e.metrics.IncTasksCompleted()
		e.metrics.RecordTaskDuration(task.Duration())

		e.logger.Info("Task completed", "task", task.Name(), "duration", task.Duration())
		return nil
	}

	// All retries exhausted
	task.Fail(lastErr)
	e.metrics.IncTasksFailed()

	return fmt.Errorf("task %s failed after %d attempts: %w", task.Name(), task.Attempts, lastErr)
}

// parseOutputs extracts structured output values from the agent's response.
// When output definitions exist, it looks for a JSON code block and maps fields.
// Falls back to storing the raw response if no JSON block is found or parsing fails.
func (e *Executor) parseOutputs(response string, definitions []config.OutputConfig) map[string]interface{} {
	outputs := make(map[string]interface{})

	// Always store the raw response.
	outputs["_response"] = response

	if len(definitions) == 0 {
		return outputs
	}

	// Try to extract a JSON block from the response.
	jsonStr := extractLastJSONBlock(response)
	if jsonStr == "" {
		// No fenced JSON block found — try to extract raw JSON object.
		jsonStr = extractRawJSON(response)
	}

	if jsonStr != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			for _, def := range definitions {
				if val, ok := parsed[def.Name]; ok {
					outputs[def.Name] = val
				}
			}
			return outputs
		}
		e.logger.Warn("Failed to parse JSON output block, using raw response", "error", "invalid JSON")
	} else {
		e.logger.Warn("No JSON output block found in response, using raw response")
	}

	// Fallback: store raw response as each output field.
	for _, def := range definitions {
		outputs[def.Name] = response
	}
	return outputs
}

// extractLastJSONBlock finds the last ```json ... ``` fenced code block in the text.
func extractLastJSONBlock(text string) string {
	lastIdx := -1
	searchFrom := 0
	for {
		idx := strings.Index(text[searchFrom:], "```json")
		if idx < 0 {
			break
		}
		lastIdx = searchFrom + idx
		searchFrom = lastIdx + 7
	}
	if lastIdx < 0 {
		return ""
	}

	start := lastIdx + 7 // skip "```json"
	// Skip optional whitespace/newline after ```json
	for start < len(text) && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n' || text[start] == '\r') {
		start++
	}

	end := strings.Index(text[start:], "```")
	if end < 0 {
		return ""
	}

	return strings.TrimSpace(text[start : start+end])
}

// extractRawJSON tries to find the last top-level JSON object {...} in the text.
func extractRawJSON(text string) string {
	// Find the last '{' and try to match it to a '}'.
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '}' {
			// Find matching opening brace.
			depth := 0
			for j := i; j >= 0; j-- {
				switch text[j] {
				case '}':
					depth++
				case '{':
					depth--
				}
				if depth == 0 {
					candidate := strings.TrimSpace(text[j : i+1])
					// Validate it's actually JSON.
					var parsed map[string]interface{}
					if json.Unmarshal([]byte(candidate), &parsed) == nil {
						return candidate
					}
					return ""
				}
			}
			break
		}
	}
	return ""
}

// backoff implements exponential backoff
func (e *Executor) backoff(attempt int) {
	baseDelay := time.Second
	maxDelay := time.Minute

	delay := baseDelay * time.Duration(1<<uint(attempt-1))
	if delay > maxDelay {
		delay = maxDelay
	}

	time.Sleep(delay)
}

// GetMetrics returns executor metrics
func (e *Executor) GetMetrics() *telemetry.Metrics {
	return e.metrics
}
