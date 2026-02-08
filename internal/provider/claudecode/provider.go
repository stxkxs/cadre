// Package claudecode provides execution plan generation for Claude Code's Task tool
package claudecode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/executor"
)

// Provider generates execution plans for Claude Code
type Provider struct {
	executor *executor.ClaudeCodeExecutor
	crewCfg  *config.CrewConfig
}

// NewProvider creates a new Claude Code provider
func NewProvider(crewCfg *config.CrewConfig) (*Provider, error) {
	exec, err := executor.NewClaudeCodeExecutor(crewCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return &Provider{
		executor: exec,
		crewCfg:  crewCfg,
	}, nil
}

// GeneratePlan creates an execution plan from inputs
func (p *Provider) GeneratePlan(ctx context.Context, inputs map[string]interface{}) (*executor.ExecutionPlan, error) {
	return p.executor.GeneratePlan(inputs)
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "claudecode"
}

// Execute generates the execution plan and returns it as formatted instructions
func (p *Provider) Execute(ctx context.Context, inputs map[string]interface{}) (*ExecutionResult, error) {
	plan, err := p.executor.GeneratePlan(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	return &ExecutionResult{
		Plan:         plan,
		Instructions: plan.ToClaudeCodeInstructions(),
		JSON:         mustJSON(plan),
	}, nil
}

// ExecutionResult contains the generated execution plan and formatted output
type ExecutionResult struct {
	Plan         *executor.ExecutionPlan
	Instructions string // Markdown formatted for Claude Code
	JSON         string // JSON format for programmatic use
}

func mustJSON(plan *executor.ExecutionPlan) string {
	j, err := plan.ToJSON()
	if err != nil {
		errMsg, _ := json.Marshal(err.Error())
		return fmt.Sprintf(`{"error": %s}`, string(errMsg))
	}
	return j
}
