// Package quick provides optimized single-agent execution for common tasks
package quick

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/executor"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// ExecutorOptions configures the quick executor
type ExecutorOptions struct {
	OutputJSON bool
	DryRun     bool
}

// Executor handles quick mode execution
type Executor struct {
	cfg     *config.Config
	logger  *telemetry.Logger
	options ExecutorOptions
}

// QuickResult holds the result of a quick execution
type QuickResult struct {
	Action  string                 `json:"action"`
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewExecutor creates a new quick mode executor
func NewExecutor(cfg *config.Config, logger *telemetry.Logger, options ExecutorOptions) (*Executor, error) {
	return &Executor{
		cfg:     cfg,
		logger:  logger,
		options: options,
	}, nil
}

// RunQuickFix executes a quick fix workflow
func (e *Executor) RunQuickFix(ctx context.Context, inputs map[string]interface{}) error {
	e.logger.Info("Starting quick fix", "issue", inputs["issue"])

	if e.options.DryRun {
		return e.printDryRun("fix", "quick-fix", inputs)
	}

	// Load quick-fix crew
	crewCfg, err := config.LoadCrew("quick-fix")
	if err != nil {
		return fmt.Errorf("failed to load quick-fix crew: %w", err)
	}

	return e.executeQuickCrew(ctx, crewCfg, inputs, "fix")
}

// RunQuickReview executes a quick code review
func (e *Executor) RunQuickReview(ctx context.Context, inputs map[string]interface{}) error {
	e.logger.Info("Starting quick review", "target", inputs["target"])

	if e.options.DryRun {
		return e.printDryRun("review", "code-review", inputs)
	}

	// For review, we use code-reviewer agent directly
	return e.executeSingleAgent(ctx, "code-reviewer", "code-review", inputs, "review")
}

// RunQuickTest executes quick test writing
func (e *Executor) RunQuickTest(ctx context.Context, inputs map[string]interface{}) error {
	e.logger.Info("Starting quick test", "target", inputs["target"])

	if e.options.DryRun {
		return e.printDryRun("test", "write-tests", inputs)
	}

	// For test writing, use test-engineer agent
	return e.executeSingleAgent(ctx, "test-engineer", "write-tests", inputs, "test")
}

// RunQuickDoc executes quick documentation generation
func (e *Executor) RunQuickDoc(ctx context.Context, inputs map[string]interface{}) error {
	e.logger.Info("Starting quick doc", "target", inputs["target"])

	if e.options.DryRun {
		return e.printDryRun("doc", "generate-docs", inputs)
	}

	// For documentation, use doc-writer agent
	return e.executeSingleAgent(ctx, "doc-writer", "generate-docs", inputs, "doc")
}

// RunQuickExplain executes quick code explanation
func (e *Executor) RunQuickExplain(ctx context.Context, inputs map[string]interface{}) error {
	e.logger.Info("Starting quick explain", "target", inputs["target"])

	if e.options.DryRun {
		return e.printDryRun("explain", "analyze-codebase", inputs)
	}

	// For explanation, use codebase-analyst agent
	return e.executeSingleAgent(ctx, "codebase-analyst", "analyze-codebase", inputs, "explain")
}

// executeQuickCrew runs a crew in quick mode
func (e *Executor) executeQuickCrew(ctx context.Context, crewCfg *config.CrewConfig, inputs map[string]interface{}, action string) error {
	// Create executor with quick mode enabled
	exec, err := executor.NewClaudeCodeExecutorWithOptions(crewCfg, true)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Generate plan
	plan, err := exec.GeneratePlan(inputs)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// Output result
	return e.outputResult(action, plan)
}

// executeSingleAgent runs a single agent in quick mode
func (e *Executor) executeSingleAgent(ctx context.Context, agentName, taskName string, inputs map[string]interface{}, action string) error {
	// Load agent config
	agentCfg, err := config.LoadAgent(agentName)
	if err != nil {
		// Fall back to quick-task agent if specific agent not found
		agentCfg, err = config.LoadAgent("quick-task")
		if err != nil {
			return fmt.Errorf("failed to load agent %s: %w", agentName, err)
		}
	}

	// Load task config if exists
	taskCfg, err := config.LoadTask(taskName)
	if err != nil {
		// Create a minimal task config
		taskCfg = e.createQuickTask(taskName, agentName, action, inputs)
	}

	// Create a single-task crew config
	crewCfg := &config.CrewConfig{
		Name:    fmt.Sprintf("quick-%s", action),
		Agents:  []string{agentCfg.Name},
		Process: "sequential",
		Tasks: []config.CrewTaskConfig{
			{
				Name:  taskCfg.Name,
				Agent: agentCfg.Name,
			},
		},
	}

	// Create executor with quick mode
	exec, err := executor.NewClaudeCodeExecutorWithOptions(crewCfg, true)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Generate plan
	plan, err := exec.GeneratePlan(inputs)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	return e.outputResult(action, plan)
}

// createQuickTask creates a minimal task config for quick actions
func (e *Executor) createQuickTask(name, agent, action string, inputs map[string]interface{}) *config.TaskConfig {
	descriptions := map[string]string{
		"review":  "Review the code at the specified target path for quality, security, and performance issues.",
		"test":    "Write tests for the code at the specified target path.",
		"doc":     "Generate documentation for the code at the specified target path.",
		"explain": "Explain what the code at the specified target path does.",
	}

	description, ok := descriptions[action]
	if !ok {
		description = fmt.Sprintf("Perform %s action on the target", action)
	}

	return &config.TaskConfig{
		Name:        name,
		Description: description,
		Agent:       agent,
		Inputs: []config.InputConfig{
			{Name: "target", Type: "string", Required: true},
			{Name: "repo_path", Type: "string", Required: false, Default: "."},
		},
		Outputs: []config.OutputConfig{
			{Name: "result", Type: "string"},
			{Name: "summary", Type: "string"},
		},
		Timeout: "5m",
		Retry: config.RetryConfig{
			MaxAttempts: 1,
			Backoff:     "fixed",
		},
	}
}

// outputResult formats and prints the execution result
func (e *Executor) outputResult(action string, plan *executor.ExecutionPlan) error {
	if e.options.OutputJSON {
		jsonOutput, err := plan.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize plan: %w", err)
		}
		fmt.Println(jsonOutput)
	} else {
		// Output Claude Code instructions
		fmt.Println(plan.ToClaudeCodeInstructions())
	}

	return nil
}

// printDryRun shows what would be executed
func (e *Executor) printDryRun(action, crewOrTask string, inputs map[string]interface{}) error {
	result := QuickResult{
		Action:  action,
		Success: true,
		Output: map[string]interface{}{
			"mode":        "dry-run",
			"crew_or_task": crewOrTask,
			"inputs":      inputs,
		},
	}

	if e.options.OutputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Dry run - would execute:")
		fmt.Printf("  Action: %s\n", action)
		fmt.Printf("  Crew/Task: %s\n", crewOrTask)
		fmt.Printf("  Mode: quick (optimized)\n")
		fmt.Printf("  Inputs:\n")
		for k, v := range inputs {
			fmt.Printf("    %s: %v\n", k, v)
		}
		fmt.Println("\nOptimizations applied:")
		fmt.Println("  - Compact backstory (reduced tokens)")
		fmt.Println("  - Input filtering (only required inputs)")
		fmt.Println("  - Reduced context limit (50K)")
		fmt.Println("  - No intermediate checkpoints")
	}

	return nil
}
