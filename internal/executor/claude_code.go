// Package executor provides execution backends for crew workflows
package executor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cadre-oss/cadre/internal/config"
)

// ClaudeCodeExecutor generates execution plans for Claude Code's Task tool
type ClaudeCodeExecutor struct {
	crewConfig *config.CrewConfig
	agents     map[string]*config.AgentConfig
	tasks      map[string]*config.TaskConfig
	quickMode  bool // Enable quick mode optimizations
}

// ExecutionPlan represents a plan that Claude Code can execute
type ExecutionPlan struct {
	ID          string           `json:"id"`
	CrewName    string           `json:"crew_name"`
	CreatedAt   time.Time        `json:"created_at"`
	Groups      []ParallelGroup  `json:"parallel_groups"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
}

// ParallelGroup represents tasks that can run in parallel
type ParallelGroup struct {
	ID        string       `json:"id"`
	DependsOn []string     `json:"depends_on,omitempty"`
	Tasks     []AgentTask  `json:"tasks"`
}

// AgentTask represents a single task for Claude Code to execute
type AgentTask struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Agent        string                 `json:"agent"`
	Description  string                 `json:"description"`
	Prompt       string                 `json:"prompt"`
	SubagentType string                 `json:"subagent_type"` // Explore, general-purpose, Bash, Plan
	Tools        []string               `json:"tools"`
	Inputs       map[string]interface{} `json:"inputs,omitempty"`
	OutputKeys   []string               `json:"output_keys,omitempty"`
}

// NewClaudeCodeExecutor creates a new Claude Code executor
func NewClaudeCodeExecutor(crewCfg *config.CrewConfig) (*ClaudeCodeExecutor, error) {
	return NewClaudeCodeExecutorWithOptions(crewCfg, false)
}

// NewClaudeCodeExecutorWithOptions creates a new Claude Code executor with options
func NewClaudeCodeExecutorWithOptions(crewCfg *config.CrewConfig, quickMode bool) (*ClaudeCodeExecutor, error) {
	executor := &ClaudeCodeExecutor{
		crewConfig: crewCfg,
		agents:     make(map[string]*config.AgentConfig),
		tasks:      make(map[string]*config.TaskConfig),
		quickMode:  quickMode,
	}

	// Load agent configs
	for _, agentName := range crewCfg.Agents {
		agentCfg, err := config.LoadAgent(agentName)
		if err != nil {
			return nil, fmt.Errorf("failed to load agent %s: %w", agentName, err)
		}
		executor.agents[agentName] = agentCfg
	}

	// Load task configs
	for _, taskCfg := range crewCfg.Tasks {
		fullTaskCfg, err := config.LoadTask(taskCfg.Name)
		if err != nil {
			// Use inline config if file doesn't exist
			fullTaskCfg = &config.TaskConfig{
				Name:         taskCfg.Name,
				Description:  taskCfg.Name,
				Agent:        taskCfg.Agent,
				Dependencies: taskCfg.DependsOn,
			}
		}
		executor.tasks[taskCfg.Name] = fullTaskCfg
	}

	return executor, nil
}

// GeneratePlan creates an execution plan from the crew configuration
func (e *ClaudeCodeExecutor) GeneratePlan(inputs map[string]interface{}) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		ID:        fmt.Sprintf("plan-%d", time.Now().Unix()),
		CrewName:  e.crewConfig.Name,
		CreatedAt: time.Now(),
		Groups:    []ParallelGroup{},
		Inputs:    inputs,
	}

	// Build dependency graph and find parallel groups
	groups := e.findParallelGroups()

	for i, group := range groups {
		pg := ParallelGroup{
			ID:        fmt.Sprintf("group_%d", i),
			DependsOn: []string{},
			Tasks:     []AgentTask{},
		}

		if i > 0 {
			pg.DependsOn = []string{fmt.Sprintf("group_%d", i-1)}
		}

		for _, taskName := range group {
			taskCfg := e.tasks[taskName]
			agentCfg := e.agents[taskCfg.Agent]

			// Filter inputs to only those needed by this task
			filteredInputs := e.filterInputsForTask(taskCfg, inputs)

			agentTask := AgentTask{
				ID:           fmt.Sprintf("%s-%d", taskName, time.Now().UnixNano()),
				Name:         taskName,
				Agent:        taskCfg.Agent,
				Description:  taskCfg.Description,
				Prompt:       e.buildPrompt(taskCfg, agentCfg, filteredInputs),
				SubagentType: e.mapToSubagentType(agentCfg),
				Tools:        agentCfg.Tools,
				Inputs:       filteredInputs,
				OutputKeys:   e.getOutputKeys(taskCfg),
			}

			pg.Tasks = append(pg.Tasks, agentTask)
		}

		plan.Groups = append(plan.Groups, pg)
	}

	return plan, nil
}

// findParallelGroups organizes tasks into groups that can run in parallel
func (e *ClaudeCodeExecutor) findParallelGroups() [][]string {
	// Build dependency map
	deps := make(map[string][]string)
	for _, taskCfg := range e.crewConfig.Tasks {
		deps[taskCfg.Name] = taskCfg.DependsOn
	}

	// Kahn's algorithm for topological sort with level tracking
	inDegree := make(map[string]int)
	for name := range e.tasks {
		inDegree[name] = len(deps[name])
	}

	var groups [][]string

	for {
		// Find all tasks with no remaining dependencies
		var ready []string
		for name, degree := range inDegree {
			if degree == 0 {
				ready = append(ready, name)
			}
		}

		if len(ready) == 0 {
			break
		}

		groups = append(groups, ready)

		// Remove these tasks and update degrees
		for _, name := range ready {
			delete(inDegree, name)
			// Decrease in-degree for tasks that depend on this
			for other, otherDeps := range deps {
				for _, dep := range otherDeps {
					if dep == name {
						inDegree[other]--
					}
				}
			}
		}
	}

	return groups
}

// filterInputsForTask returns only the inputs that a task needs
func (e *ClaudeCodeExecutor) filterInputsForTask(taskCfg *config.TaskConfig, allInputs map[string]interface{}) map[string]interface{} {
	// If task has no defined inputs, pass all inputs (backwards compatibility)
	if len(taskCfg.Inputs) == 0 {
		return allInputs
	}

	filtered := make(map[string]interface{})
	for _, input := range taskCfg.Inputs {
		if val, ok := allInputs[input.Name]; ok {
			filtered[input.Name] = val
		} else if input.Default != "" {
			filtered[input.Name] = input.Default
		}
	}
	return filtered
}

// buildPrompt creates the prompt for an agent task
func (e *ClaudeCodeExecutor) buildPrompt(taskCfg *config.TaskConfig, agentCfg *config.AgentConfig, inputs map[string]interface{}) string {
	// Use compact backstory in quick mode if available
	backstory := agentCfg.Backstory
	if e.quickMode && agentCfg.CompactBackstory != "" {
		backstory = agentCfg.CompactBackstory
	}

	var prompt string
	if e.quickMode {
		// Minimal prompt for quick mode
		prompt = fmt.Sprintf(`You are %s (%s). %s

## Task: %s

%s

`, agentCfg.Name, agentCfg.Role, backstory, taskCfg.Name, taskCfg.Description)
	} else {
		// Full prompt for normal mode
		prompt = fmt.Sprintf(`You are %s.

Role: %s
Goal: %s

%s

## Task: %s

%s

`, agentCfg.Name, agentCfg.Role, agentCfg.Goal, backstory, taskCfg.Name, taskCfg.Description)
	}

	if len(inputs) > 0 {
		prompt += "## Inputs\n\n"
		for k, v := range inputs {
			prompt += fmt.Sprintf("- **%s**: %v\n", k, v)
		}
		prompt += "\n"
	}

	if len(taskCfg.Outputs) > 0 {
		prompt += "## Expected Outputs\n\n"
		for _, out := range taskCfg.Outputs {
			prompt += fmt.Sprintf("- **%s** (%s)\n", out.Name, out.Type)
		}
		prompt += "\n"
	}

	// Add structured output schema if defined
	if taskCfg.OutputSchema != nil {
		prompt += e.buildOutputSchemaInstructions(taskCfg.OutputSchema)
	}

	if e.quickMode {
		prompt += `## Instructions

Be direct and efficient. Complete the task and provide output in the specified format.
`
	} else {
		prompt += `## Instructions

1. Complete the task thoroughly
2. Use available tools as needed
3. Provide clear, actionable output
4. Structure your response with the expected outputs clearly labeled
`
	}

	return prompt
}

// buildOutputSchemaInstructions generates structured output format instructions
func (e *ClaudeCodeExecutor) buildOutputSchemaInstructions(schema *config.OutputSchemaConfig) string {
	if schema == nil {
		return ""
	}

	var sb string
	sb = "## Output Format\n\n"

	if schema.Strict {
		sb += "**REQUIRED**: Your response MUST end with output in this exact format.\n\n"
	} else {
		sb += "End your response with output in this format:\n\n"
	}

	sb += fmt.Sprintf("```%s:output-manifest\n", schema.Format)

	if schema.Format == "json" && len(schema.Fields) > 0 {
		sb += "{\n"
		for i, field := range schema.Fields {
			typeExample := getTypeExample(field.Type)
			comment := ""
			if field.Description != "" {
				comment = " // " + field.Description
			}
			if field.Required {
				comment += " (required)"
			}
			comma := ","
			if i == len(schema.Fields)-1 {
				comma = ""
			}
			sb += fmt.Sprintf("  \"%s\": %s%s%s\n", field.Name, typeExample, comma, comment)
		}
		sb += "}\n"
	} else {
		sb += "<structured output here>\n"
	}

	sb += "```\n\n"
	return sb
}

// getTypeExample returns an example value for a type
func getTypeExample(t string) string {
	switch t {
	case "string":
		return "\"<string>\""
	case "string[]":
		return "[\"<string>\"]"
	case "int":
		return "<number>"
	case "bool", "boolean":
		return "<true|false>"
	case "object":
		return "{}"
	case "object[]":
		return "[{}]"
	default:
		return "\"<" + t + ">\""
	}
}

// mapToSubagentType maps agent tools to Claude Code subagent types
func (e *ClaudeCodeExecutor) mapToSubagentType(agentCfg *config.AgentConfig) string {
	hasWrite := false
	hasBash := false

	for _, tool := range agentCfg.Tools {
		if tool == "file_write" {
			hasWrite = true
		}
		if tool == "bash" {
			hasBash = true
		}
	}

	// If agent can write files or run bash, use general-purpose
	if hasWrite || hasBash {
		return "general-purpose"
	}

	// Read-only agents can use Explore
	return "Explore"
}

// getOutputKeys extracts output key names from task config
func (e *ClaudeCodeExecutor) getOutputKeys(taskCfg *config.TaskConfig) []string {
	keys := make([]string, len(taskCfg.Outputs))
	for i, out := range taskCfg.Outputs {
		keys[i] = out.Name
	}
	return keys
}

// ToJSON serializes the plan to JSON
func (p *ExecutionPlan) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToMarkdown generates a markdown representation of the plan
func (p *ExecutionPlan) ToMarkdown() string {
	md := fmt.Sprintf("# Execution Plan: %s\n\n", p.CrewName)
	md += fmt.Sprintf("**ID**: %s\n", p.ID)
	md += fmt.Sprintf("**Created**: %s\n\n", p.CreatedAt.Format(time.RFC3339))

	if len(p.Inputs) > 0 {
		md += "## Inputs\n\n"
		for k, v := range p.Inputs {
			md += fmt.Sprintf("- **%s**: %v\n", k, v)
		}
		md += "\n"
	}

	md += "## Execution Groups\n\n"
	for i, group := range p.Groups {
		md += fmt.Sprintf("### Group %d", i+1)
		if len(group.Tasks) > 1 {
			md += " (parallel)"
		}
		md += "\n\n"

		if len(group.DependsOn) > 0 {
			md += fmt.Sprintf("*Depends on: %v*\n\n", group.DependsOn)
		}

		for _, task := range group.Tasks {
			md += fmt.Sprintf("#### Task: %s\n\n", task.Name)
			md += fmt.Sprintf("- **Agent**: %s\n", task.Agent)
			md += fmt.Sprintf("- **Subagent Type**: %s\n", task.SubagentType)
			md += fmt.Sprintf("- **Tools**: %v\n", task.Tools)
			md += "\n"
		}
	}

	return md
}

// ToClaudeCodeInstructions generates detailed instructions for Claude Code's Task tool
// This output is designed to be copy-pasted directly into a Claude Code session
func (p *ExecutionPlan) ToClaudeCodeInstructions() string {
	var md string

	// Header
	md = fmt.Sprintf("# Crew Execution Plan: %s\n\n", p.CrewName)
	md += fmt.Sprintf("**Plan ID**: `%s`\n", p.ID)
	md += fmt.Sprintf("**Generated**: %s\n\n", p.CreatedAt.Format(time.RFC3339))

	// Overview
	md += "## Overview\n\n"
	md += "This execution plan organizes tasks into parallel groups based on their dependencies.\n"
	md += "Tasks within the same group can run concurrently using Claude Code's Task tool.\n\n"

	// Inputs section
	if len(p.Inputs) > 0 {
		md += "## Inputs\n\n"
		md += "These values should be passed to the first group of tasks:\n\n"
		for k, v := range p.Inputs {
			md += fmt.Sprintf("- `%s`: %v\n", k, v)
		}
		md += "\n"
	}

	// Execution instructions
	md += "## Execution Instructions\n\n"
	md += "**IMPORTANT**: For each group, launch all tasks in a **single message** with multiple Task tool calls.\n"
	md += "This enables parallel execution. Wait for all tasks in a group to complete before proceeding to the next group.\n\n"

	// Groups
	for i, group := range p.Groups {
		groupNum := i + 1
		isParallel := len(group.Tasks) > 1

		md += fmt.Sprintf("---\n\n### Group %d", groupNum)
		if isParallel {
			md += fmt.Sprintf(" (%d tasks - PARALLEL)", len(group.Tasks))
		} else {
			md += " (1 task)"
		}
		md += "\n\n"

		if len(group.DependsOn) > 0 {
			md += fmt.Sprintf("**Wait for**: %v to complete\n\n", group.DependsOn)
		}

		if isParallel {
			md += "**Launch these tasks in a SINGLE message (parallel execution)**:\n\n"
		}

		for j, task := range group.Tasks {
			md += fmt.Sprintf("#### %d.%d: %s\n\n", groupNum, j+1, task.Name)
			md += fmt.Sprintf("- **Agent**: %s\n", task.Agent)
			md += fmt.Sprintf("- **Subagent Type**: `%s`\n", task.SubagentType)
			md += fmt.Sprintf("- **Description**: %s\n", task.Description)

			if len(task.Tools) > 0 {
				md += fmt.Sprintf("- **Tools**: %v\n", task.Tools)
			}

			if len(task.OutputKeys) > 0 {
				md += fmt.Sprintf("- **Expected Outputs**: %v\n", task.OutputKeys)
			}

			md += "\n**Task Tool Call**:\n"
			md += "```\n"
			md += fmt.Sprintf("Task(\n")
			md += fmt.Sprintf("  subagent_type: \"%s\",\n", task.SubagentType)
			md += fmt.Sprintf("  description: \"%s\",\n", truncateString(task.Description, 50))
			md += fmt.Sprintf("  prompt: <see below>\n")
			md += fmt.Sprintf(")\n")
			md += "```\n\n"

			md += "<details>\n"
			md += fmt.Sprintf("<summary>Full Prompt for %s</summary>\n\n", task.Name)
			md += "```\n"
			md += task.Prompt
			md += "```\n\n"
			md += "</details>\n\n"
		}
	}

	// Output capture format
	md += "---\n\n## Output Capture Format\n\n"
	md += "After each group completes, capture outputs in this format for passing to dependent tasks:\n\n"
	md += "```yaml\n"
	md += "outputs:\n"
	md += "  <task-name>:\n"
	md += "    <output-key>: \"<value>\"\n"
	md += "    # ... additional outputs\n"
	md += "```\n\n"

	md += "Pass relevant outputs to subsequent tasks via their inputs.\n\n"

	// Summary
	md += "---\n\n## Execution Summary\n\n"
	md += "| Group | Tasks | Execution |\n"
	md += "|-------|-------|----------|\n"
	for i, group := range p.Groups {
		taskNames := make([]string, len(group.Tasks))
		for j, t := range group.Tasks {
			taskNames[j] = t.Name
		}
		execType := "Sequential"
		if len(group.Tasks) > 1 {
			execType = "**Parallel**"
		}
		md += fmt.Sprintf("| %d | %s | %s |\n", i+1, joinStrings(taskNames, ", "), execType)
	}
	md += "\n"

	return md
}

// truncateString truncates a string to max length, adding "..." if truncated
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
