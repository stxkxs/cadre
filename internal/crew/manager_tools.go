package crew

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cadre-oss/cadre/internal/agent"
	"github.com/cadre-oss/cadre/internal/event"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/task"
)

// delegateTaskTool lets the manager agent delegate a task to a worker agent.
type delegateTaskTool struct {
	dag      *task.DAG
	agents   map[string]*agent.Runtime
	executor task.TaskExecutor
	stateMgr *state.Manager
	eventBus *event.Bus
}

func (d *delegateTaskTool) Name() string { return "delegate_task" }

func (d *delegateTaskTool) Description() string {
	return "Delegate a task to an agent for execution. The task must be ready (pending with all dependencies completed) or retriable (failed with retries remaining). Returns the task result."
}

func (d *delegateTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"task_name": map[string]interface{}{
			"type":        "string",
			"description": "Name of the task to delegate",
		},
		"agent_name": map[string]interface{}{
			"type":        "string",
			"description": "Name of the agent to execute the task",
		},
		"instructions": map[string]interface{}{
			"type":        "string",
			"description": "Optional additional instructions for the agent",
		},
	}
}

func (d *delegateTaskTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		TaskName     string `json:"task_name"`
		AgentName    string `json:"agent_name"`
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.TaskName == "" || params.AgentName == "" {
		return "", fmt.Errorf("task_name and agent_name are required")
	}

	// Get the task from the DAG.
	t, ok := d.dag.GetTask(params.TaskName)
	if !ok {
		return marshalResult(map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("task %q not found", params.TaskName),
		})
	}

	// Validate the task is delegatable.
	status := t.GetStatus()
	if status != "pending" && !(status == "failed" && t.CanRetry()) {
		return marshalResult(map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("task %q is not ready for delegation (status: %s)", params.TaskName, status),
		})
	}

	// Check that all dependencies are completed.
	for _, depName := range d.dag.GetDependencies(params.TaskName) {
		depTask, _ := d.dag.GetTask(depName)
		if depTask == nil || depTask.GetStatus() != "completed" {
			return marshalResult(map[string]interface{}{
				"status": "error",
				"error":  fmt.Sprintf("dependency %q is not completed", depName),
			})
		}
	}

	// Get the agent runtime.
	rt, ok := d.agents[params.AgentName]
	if !ok {
		return marshalResult(map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("agent %q not found", params.AgentName),
		})
	}

	// Propagate dependency outputs as inputs.
	for _, depName := range d.dag.GetDependencies(params.TaskName) {
		depTask, _ := d.dag.GetTask(depName)
		if depTask != nil {
			for k, v := range depTask.GetOutputs() {
				if k != "_response" {
					t.SetInput(k, v)
				}
			}
		}
	}

	// Add manager instructions if provided.
	if params.Instructions != "" {
		t.SetInput("_manager_instructions", params.Instructions)
	}

	// Emit delegation event.
	d.eventBus.Emit(event.NewEvent(event.ManagerDelegated, map[string]interface{}{
		"task":  params.TaskName,
		"agent": params.AgentName,
	}))

	d.stateMgr.UpdateTaskState(params.TaskName, "running", nil, nil)
	d.eventBus.Emit(event.NewEvent(event.TaskStarted, map[string]interface{}{
		"task":  params.TaskName,
		"agent": params.AgentName,
	}))

	// Execute the task synchronously.
	err := d.executor.Execute(ctx, t, rt)
	if err != nil {
		d.stateMgr.UpdateTaskState(params.TaskName, "failed", nil, err)
		d.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
			"task":  params.TaskName,
			"error": err.Error(),
		}))
		return marshalResult(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
	}

	outputs := t.GetOutputs()
	d.stateMgr.UpdateTaskState(params.TaskName, "completed", outputs, nil)
	d.eventBus.Emit(event.NewEvent(event.TaskCompleted, map[string]interface{}{
		"task": params.TaskName,
	}))

	return marshalResult(map[string]interface{}{
		"status":  "completed",
		"outputs": outputs,
	})
}

func (d *delegateTaskTool) Test(ctx context.Context) (string, error) {
	return "delegate_task tool operational", nil
}

// checkStatusTool lets the manager query DAG status.
type checkStatusTool struct {
	dag *task.DAG
}

func (c *checkStatusTool) Name() string { return "check_status" }

func (c *checkStatusTool) Description() string {
	return "Check the current status of tasks. Omit task_name to get status of all tasks."
}

func (c *checkStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"task_name": map[string]interface{}{
			"type":        "string",
			"description": "Optional: name of a specific task to check. Omit for all tasks.",
		},
	}
}

func (c *checkStatusTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		TaskName string `json:"task_name"`
	}
	// Ignore unmarshal error — params are optional.
	json.Unmarshal(args, &params)

	if params.TaskName != "" {
		t, ok := c.dag.GetTask(params.TaskName)
		if !ok {
			return marshalResult(map[string]interface{}{
				"error": fmt.Sprintf("task %q not found", params.TaskName),
			})
		}
		return marshalResult(taskStatusInfo(c.dag, t))
	}

	// Return status of all tasks.
	tasks := c.dag.GetTasks()
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, taskStatusInfo(c.dag, t))
	}

	ready := c.dag.GetReady()
	readyNames := make([]string, 0, len(ready))
	for _, t := range ready {
		readyNames = append(readyNames, t.Name())
	}

	return marshalResult(map[string]interface{}{
		"tasks":        result,
		"ready":        readyNames,
		"all_complete": c.dag.IsComplete(),
	})
}

func (c *checkStatusTool) Test(ctx context.Context) (string, error) {
	return "check_status tool operational", nil
}

// provideFeedbackTool lets the manager add context to a task before delegation.
type provideFeedbackTool struct {
	dag      *task.DAG
	eventBus *event.Bus
}

func (p *provideFeedbackTool) Name() string { return "provide_feedback" }

func (p *provideFeedbackTool) Description() string {
	return "Provide feedback or additional context to a task before delegating it. The feedback will be included in the task prompt."
}

func (p *provideFeedbackTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"task_name": map[string]interface{}{
			"type":        "string",
			"description": "Name of the task to provide feedback for",
		},
		"feedback": map[string]interface{}{
			"type":        "string",
			"description": "Feedback or additional context for the task",
		},
	}
}

func (p *provideFeedbackTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		TaskName string `json:"task_name"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.TaskName == "" || params.Feedback == "" {
		return "", fmt.Errorf("task_name and feedback are required")
	}

	t, ok := p.dag.GetTask(params.TaskName)
	if !ok {
		return marshalResult(map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("task %q not found", params.TaskName),
		})
	}

	t.SetInput("_manager_feedback", params.Feedback)

	p.eventBus.Emit(event.NewEvent(event.ManagerFeedback, map[string]interface{}{
		"task":     params.TaskName,
		"feedback": params.Feedback,
	}))

	return marshalResult(map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Feedback set for task %q", params.TaskName),
	})
}

func (p *provideFeedbackTool) Test(ctx context.Context) (string, error) {
	return "provide_feedback tool operational", nil
}

// --- helpers ---

func taskStatusInfo(dag *task.DAG, t *task.Task) map[string]interface{} {
	info := map[string]interface{}{
		"name":   t.Name(),
		"status": t.GetStatus(),
	}

	deps := dag.GetDependencies(t.Name())
	if len(deps) > 0 {
		info["dependencies"] = deps

		// Find which deps are blocking.
		var blocking []string
		for _, dep := range deps {
			dt, _ := dag.GetTask(dep)
			if dt != nil && dt.GetStatus() != "completed" {
				blocking = append(blocking, dep)
			}
		}
		if len(blocking) > 0 {
			info["blocked_by"] = blocking
		}
	}

	outputs := t.GetOutputs()
	// Filter out _response from status display.
	filtered := make(map[string]interface{})
	for k, v := range outputs {
		if !strings.HasPrefix(k, "_") {
			filtered[k] = v
		}
	}
	if len(filtered) > 0 {
		info["outputs"] = filtered
	}

	return info
}

func marshalResult(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
