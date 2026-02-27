package crew

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/task"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/stxkxs/cadre/internal/tool"
)

// hierarchicalCoordinator manages task execution via a manager agent
// that delegates work using tool calls.
type hierarchicalCoordinator struct {
	dag      *task.DAG
	agents   map[string]*agent.Runtime
	executor task.TaskExecutor
	stateMgr *state.Manager
	logger   *telemetry.Logger
	eventBus *event.Bus

	managerName    string
	managerRuntime *agent.Runtime
	timeout        time.Duration // 0 means no timeout
}

func newHierarchicalCoordinator(
	dag *task.DAG,
	agents map[string]*agent.Runtime,
	executor task.TaskExecutor,
	stateMgr *state.Manager,
	managerName string,
	logger *telemetry.Logger,
	eventBus *event.Bus,
) (*hierarchicalCoordinator, error) {
	managerRT, ok := agents[managerName]
	if !ok {
		return nil, fmt.Errorf("manager agent %q not found", managerName)
	}

	return &hierarchicalCoordinator{
		dag:            dag,
		agents:         agents,
		executor:       executor,
		stateMgr:       stateMgr,
		logger:         logger,
		eventBus:       eventBus,
		managerName:    managerName,
		managerRuntime: managerRT,
	}, nil
}

func (hc *hierarchicalCoordinator) execute(ctx context.Context) (map[string]interface{}, error) {
	// Build the delegation tools with references to coordinator state.
	delegationTools := map[string]tool.Tool{
		"delegate_task": &delegateTaskTool{
			dag:      hc.dag,
			agents:   hc.agents,
			executor: hc.executor,
			stateMgr: hc.stateMgr,
			eventBus: hc.eventBus,
		},
		"check_status": &checkStatusTool{
			dag: hc.dag,
		},
		"provide_feedback": &provideFeedbackTool{
			dag:      hc.dag,
			eventBus: hc.eventBus,
		},
	}

	// Inject delegation tools into the manager's runtime.
	hc.managerRuntime.AddTools(delegationTools)
	hc.managerRuntime.SetMaxIterations(50) // Many tool calls expected.

	// Build manager prompt.
	prompt := hc.buildManagerPrompt()

	hc.logger.Info("Starting hierarchical execution", "manager", hc.managerName)

	// Apply timeout to the manager execution if configured.
	execCtx := ctx
	if hc.timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, hc.timeout)
		defer cancel()
	}

	// Execute the manager agent — it will delegate tasks via tool calls.
	_, err := hc.managerRuntime.Execute(execCtx, prompt)
	if err != nil {
		return nil, fmt.Errorf("manager agent failed: %w", err)
	}

	// Verify all tasks completed.
	var incomplete []string
	for _, t := range hc.dag.GetTasks() {
		if t.GetStatus() != "completed" {
			incomplete = append(incomplete, t.Name())
		}
	}
	if len(incomplete) > 0 {
		return nil, fmt.Errorf("manager finished without completing all tasks: %s", strings.Join(incomplete, ", "))
	}

	// Collect outputs from leaf tasks.
	outputs := make(map[string]interface{})
	for _, t := range hc.dag.GetTasks() {
		if len(hc.dag.GetChildren(t.Name())) == 0 {
			for k, v := range t.GetOutputs() {
				outputs[k] = v
			}
		}
	}

	return outputs, nil
}

func (hc *hierarchicalCoordinator) buildManagerPrompt() string {
	var b strings.Builder

	b.WriteString("You are the manager of this crew. Oversee execution by delegating tasks to agents.\n\n")

	// Available agents.
	b.WriteString("## Available Agents\n")
	for name, rt := range hc.agents {
		if name == hc.managerName {
			continue
		}
		if rt == nil {
			b.WriteString(fmt.Sprintf("- %s\n", name))
			continue
		}
		a := rt.GetAgent()
		toolNames := strings.Join(a.Tools(), ", ")
		b.WriteString(fmt.Sprintf("- %s (Role: %s", name, a.Role()))
		if toolNames != "" {
			b.WriteString(fmt.Sprintf(", Tools: %s", toolNames))
		}
		b.WriteString(")\n")
	}
	b.WriteString("\n")

	// Tasks to complete.
	tasks, _ := hc.dag.TopologicalSort()
	b.WriteString("## Tasks to Complete\n")
	for i, t := range tasks {
		deps := hc.dag.GetDependencies(t.Name())
		depStr := "none"
		if len(deps) > 0 {
			depStr = "[" + strings.Join(deps, ", ") + "]"
		}
		b.WriteString(fmt.Sprintf("%d. %s (Description: %s, Suggested agent: %s, Dependencies: %s)\n",
			i+1, t.Name(), t.Description(), t.Agent(), depStr))
	}
	b.WriteString("\n")

	// Current status.
	b.WriteString("## Current Status\n")
	for _, t := range tasks {
		status := t.GetStatus()
		extra := ""
		if status == "pending" {
			// Check if ready.
			allReady := true
			var blockers []string
			for _, dep := range hc.dag.GetDependencies(t.Name()) {
				dt, _ := hc.dag.GetTask(dep)
				if dt != nil && dt.GetStatus() != "completed" {
					allReady = false
					blockers = append(blockers, dep)
				}
			}
			if allReady {
				extra = " (READY)"
			} else {
				extra = " (blocked by: " + strings.Join(blockers, ", ") + ")"
			}
		}
		b.WriteString(fmt.Sprintf("- %s: %s%s\n", t.Name(), status, extra))
	}
	b.WriteString("\n")

	// Instructions.
	b.WriteString("## Instructions\n")
	b.WriteString("- Use delegate_task to assign tasks to agents\n")
	b.WriteString("- You may override the suggested agent if a different one is better suited\n")
	b.WriteString("- Review results via check_status after each delegation\n")
	b.WriteString("- If a task fails, you may re-delegate to the same or different agent\n")
	b.WriteString("- Use provide_feedback to add context before delegating\n")
	b.WriteString("- Complete all tasks, then provide a final summary\n")

	return b.String()
}
