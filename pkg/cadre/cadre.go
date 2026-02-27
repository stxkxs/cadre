// Package cadre provides a public API for the cadre orchestration framework.
//
// Example usage:
//
//	import "github.com/stxkxs/cadre/pkg/cadre"
//
//	// Run a crew
//	result, err := cadre.Run("development", map[string]interface{}{
//		"requirements": "Add a new login feature",
//	})
//
//	// Run a single agent
//	response, err := cadre.RunAgent("developer", "Write a function to sort a list")
package cadre

import (
	"context"
	"fmt"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	internalCrew "github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// StreamCallback receives text chunks as they arrive.
type StreamCallback = agent.StreamCallback

// Run executes a crew workflow
func Run(crewName string, inputs map[string]interface{}) (map[string]interface{}, error) {
	return RunWithContext(context.Background(), crewName, inputs)
}

// RunWithContext executes a crew workflow with a context
func RunWithContext(ctx context.Context, crewName string, inputs map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	crewCfg, err := config.LoadCrew(crewName)
	if err != nil {
		return nil, fmt.Errorf("failed to load crew: %w", err)
	}

	logger := telemetry.NewLogger(false)
	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	orchestrator, err := internalCrew.NewOrchestrator(cfg, crewCfg, stateMgr, logger, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return orchestrator.Execute(ctx, inputs)
}

// RunTask executes a single task
func RunTask(taskName string, inputs map[string]interface{}) (map[string]interface{}, error) {
	return RunTaskWithContext(context.Background(), taskName, inputs)
}

// RunTaskWithContext executes a single task with a context
func RunTaskWithContext(ctx context.Context, taskName string, inputs map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	taskCfg, err := config.LoadTask(taskName)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	agentCfg, err := config.LoadAgent(taskCfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent: %w", err)
	}

	logger := telemetry.NewLogger(false)
	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	orchestrator, err := internalCrew.NewTaskOrchestrator(cfg, taskCfg, agentCfg, stateMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return orchestrator.ExecuteTask(ctx, inputs)
}

// RunAgent runs an agent with a single prompt
func RunAgent(agentName, prompt string) (string, error) {
	return RunAgentWithContext(context.Background(), agentName, prompt)
}

// RunAgentWithContext runs an agent with a single prompt and context
func RunAgentWithContext(ctx context.Context, agentName, prompt string) (string, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	agentCfg, err := config.LoadAgent(agentName)
	if err != nil {
		return "", fmt.Errorf("failed to load agent: %w", err)
	}

	logger := telemetry.NewLogger(false)
	stateMgr, err := state.NewManager("memory", "")
	if err != nil {
		return "", fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	runtime, err := internalCrew.NewAgentRuntime(cfg, agentCfg, stateMgr, logger)
	if err != nil {
		return "", fmt.Errorf("failed to create runtime: %w", err)
	}

	return runtime.Execute(ctx, prompt)
}

// StreamAgent runs an agent with streaming output.
func StreamAgent(agentName, prompt string, callback StreamCallback) (string, error) {
	return StreamAgentWithContext(context.Background(), agentName, prompt, callback)
}

// StreamAgentWithContext runs an agent with streaming output and context.
func StreamAgentWithContext(ctx context.Context, agentName, prompt string, callback StreamCallback) (string, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	agentCfg, err := config.LoadAgent(agentName)
	if err != nil {
		return "", fmt.Errorf("failed to load agent: %w", err)
	}

	logger := telemetry.NewLogger(false)
	stateMgr, err := state.NewManager("memory", "")
	if err != nil {
		return "", fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	runtime, err := internalCrew.NewAgentRuntime(cfg, agentCfg, stateMgr, logger)
	if err != nil {
		return "", fmt.Errorf("failed to create runtime: %w", err)
	}

	return runtime.StreamExecute(ctx, prompt, callback)
}

// ListAgents returns all configured agents
func ListAgents() ([]string, error) {
	entries, err := config.LoadAgentList()
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ListTasks returns all configured tasks
func ListTasks() ([]string, error) {
	entries, err := config.LoadTaskList()
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ListCrews returns all configured crews
func ListCrews() ([]string, error) {
	entries, err := config.LoadCrewList()
	if err != nil {
		return nil, err
	}
	return entries, nil
}
