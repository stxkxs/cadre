package crew

import (
	"context"
	"fmt"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/task"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// TaskOrchestrator handles single task execution
type TaskOrchestrator struct {
	config    *config.Config
	taskCfg   *config.TaskConfig
	agentCfg  *config.AgentConfig
	stateMgr  *state.Manager
	logger    *telemetry.Logger
	runtime   *agent.Runtime
	executor  *task.Executor
}

// NewTaskOrchestrator creates an orchestrator for a single task
func NewTaskOrchestrator(cfg *config.Config, taskCfg *config.TaskConfig, agentCfg *config.AgentConfig, stateMgr *state.Manager, logger *telemetry.Logger) (*TaskOrchestrator, error) {
	runtime, err := agent.NewRuntime(cfg, agentCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runtime: %w", err)
	}

	return &TaskOrchestrator{
		config:   cfg,
		taskCfg:  taskCfg,
		agentCfg: agentCfg,
		stateMgr: stateMgr,
		logger:   logger,
		runtime:  runtime,
		executor: task.NewExecutor(cfg, logger),
	}, nil
}

// ExecuteTask runs the task and returns outputs
func (o *TaskOrchestrator) ExecuteTask(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	o.logger.Info("Executing task", "task", o.taskCfg.Name, "agent", o.taskCfg.Agent)

	// Create task instance
	t := task.NewTask(o.taskCfg)

	// Set inputs
	for k, v := range inputs {
		t.SetInput(k, v)
	}

	// Execute
	err := o.executor.Execute(ctx, t, o.runtime)
	if err != nil {
		return nil, err
	}

	return t.GetOutputs(), nil
}

// NewAgentRuntime creates an agent runtime for direct usage
func NewAgentRuntime(cfg *config.Config, agentCfg *config.AgentConfig, stateMgr *state.Manager, logger *telemetry.Logger) (*agent.Runtime, error) {
	return agent.NewRuntime(cfg, agentCfg, logger)
}
