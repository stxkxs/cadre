package config

import (
	"fmt"
	"strings"
	"time"
)

// validateAgent validates an agent configuration
func validateAgent(cfg *AgentConfig) error {
	var errors []string

	if cfg.Name == "" {
		errors = append(errors, "name is required")
	}
	if cfg.Role == "" {
		errors = append(errors, "role is required")
	}
	if cfg.Goal == "" {
		errors = append(errors, "goal is required")
	}

	// Validate memory type
	validMemoryTypes := map[string]bool{
		"conversation": true,
		"long_term":    true,
		"shared":       true,
		"":             true, // empty defaults to conversation
	}
	if !validMemoryTypes[cfg.Memory.Type] {
		errors = append(errors, fmt.Sprintf("invalid memory type: %s", cfg.Memory.Type))
	}

	if len(errors) > 0 {
		return fmt.Errorf("agent validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

// validateTask validates a task configuration
func validateTask(cfg *TaskConfig) error {
	var errors []string

	if cfg.Name == "" {
		errors = append(errors, "name is required")
	}
	if cfg.Description == "" {
		errors = append(errors, "description is required")
	}
	if cfg.Agent == "" {
		errors = append(errors, "agent is required")
	}

	// Validate input types
	validTypes := map[string]bool{
		"string":   true,
		"string[]": true,
		"int":      true,
		"bool":     true,
		"boolean":  true,
		"object":   true,
		"":         true,
	}
	for _, input := range cfg.Inputs {
		if input.Name == "" {
			errors = append(errors, "input name is required")
		}
		if !validTypes[input.Type] {
			errors = append(errors, fmt.Sprintf("invalid input type: %s", input.Type))
		}
	}

	// Validate output types
	for _, output := range cfg.Outputs {
		if output.Name == "" {
			errors = append(errors, "output name is required")
		}
		if !validTypes[output.Type] {
			errors = append(errors, fmt.Sprintf("invalid output type: %s", output.Type))
		}
	}

	// Validate retry config
	validBackoff := map[string]bool{
		"fixed":       true,
		"exponential": true,
		"":            true,
	}
	if !validBackoff[cfg.Retry.Backoff] {
		errors = append(errors, fmt.Sprintf("invalid backoff type: %s", cfg.Retry.Backoff))
	}

	// Validate timeout format at validation time
	if cfg.Timeout != "" {
		if _, err := time.ParseDuration(cfg.Timeout); err != nil {
			errors = append(errors, fmt.Sprintf("invalid timeout format %q: %s", cfg.Timeout, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("task validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

// ValidateCrew validates a crew configuration
func ValidateCrew(cfg *CrewConfig) error {
	var errors []string

	if cfg.Name == "" {
		errors = append(errors, "name is required")
	}
	if len(cfg.Agents) == 0 {
		errors = append(errors, "at least one agent is required")
	}

	// Validate process type
	validProcesses := map[string]bool{
		"sequential":   true,
		"parallel":     true,
		"hierarchical": true,
		"":             true, // defaults to sequential
	}
	if !validProcesses[cfg.Process] {
		errors = append(errors, fmt.Sprintf("invalid process type: %s", cfg.Process))
	}

	// Hierarchical requires manager
	if cfg.Process == "hierarchical" && cfg.Manager == "" {
		errors = append(errors, "hierarchical process requires a manager")
	}

	// Validate error strategy
	validErrorStrategies := map[string]bool{
		"fail-fast":        true,
		"complete-running": true,
		"continue-all":     true,
		"":                 true, // defaults to fail-fast
	}
	if !validErrorStrategies[cfg.ErrorStrategy] {
		errors = append(errors, fmt.Sprintf("invalid error_strategy: %s (must be fail-fast, complete-running, or continue-all)", cfg.ErrorStrategy))
	}

	// Validate max_iterations
	if cfg.MaxIterations < 0 {
		errors = append(errors, "max_iterations must be non-negative")
	}
	if cfg.MaxIterations > 100 {
		errors = append(errors, "max_iterations cannot exceed 100")
	}

	// Validate tasks
	agentSet := make(map[string]bool)
	for _, a := range cfg.Agents {
		agentSet[a] = true
	}

	taskNames := make(map[string]bool)
	for _, task := range cfg.Tasks {
		if task.Name == "" {
			errors = append(errors, "task name is required")
			continue
		}
		if taskNames[task.Name] {
			errors = append(errors, fmt.Sprintf("duplicate task name: %s", task.Name))
		}
		taskNames[task.Name] = true

		if task.Agent == "" {
			errors = append(errors, fmt.Sprintf("task %s requires an agent", task.Name))
		}

		// Validate task agent is in crew agents list
		if task.Agent != "" && !agentSet[task.Agent] {
			errors = append(errors, fmt.Sprintf("task %s references agent %q which is not in the crew agents list", task.Name, task.Agent))
		}
	}

	// Validate manager is in agents list (if hierarchical)
	if cfg.Manager != "" && !agentSet[cfg.Manager] {
		errors = append(errors, fmt.Sprintf("manager %q is not in the crew agents list", cfg.Manager))
	}

	// Validate dependencies exist
	for _, task := range cfg.Tasks {
		for _, dep := range task.DependsOn {
			if !taskNames[dep] {
				errors = append(errors, fmt.Sprintf("task %s depends on unknown task %q", task.Name, dep))
			}
		}
	}

	// Note: dependency cycles are allowed (loop workflows).
	// The runtime handles cycles via iteration limits.

	if len(errors) > 0 {
		return fmt.Errorf("crew validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

// validateDependencyCycles checks for circular dependencies using DFS.
func validateDependencyCycles(tasks []CrewTaskConfig) error {
	// Build adjacency list: task -> its dependencies
	deps := make(map[string][]string)
	for _, t := range tasks {
		deps[t.Name] = t.DependsOn
	}

	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully explored
	)
	color := make(map[string]int)

	var visit func(name string, path []string) error
	visit = func(name string, path []string) error {
		color[name] = gray
		path = append(path, name)

		for _, dep := range deps[name] {
			switch color[dep] {
			case gray:
				// Found a cycle — build the cycle path for a clear error.
				cycle := append(path, dep)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
			case white:
				if err := visit(dep, path); err != nil {
					return err
				}
			}
		}

		color[name] = black
		return nil
	}

	for _, t := range tasks {
		if color[t.Name] == white {
			if err := visit(t.Name, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateTool validates a tool configuration
func validateTool(cfg *ToolConfig) error {
	var errors []string

	if cfg.Name == "" {
		errors = append(errors, "name is required")
	}
	if cfg.Description == "" {
		errors = append(errors, "description is required")
	}

	// Validate provider
	validProviders := map[string]bool{
		"mcp":     true,
		"exec":    true,
		"http":    true,
		"builtin": true,
		"":        true,
	}
	if !validProviders[cfg.Provider] {
		errors = append(errors, fmt.Sprintf("invalid provider: %s", cfg.Provider))
	}

	// Validate exec tools have a command
	if cfg.Provider == "exec" {
		if cmd, ok := cfg.Config["command"]; !ok || cmd == "" {
			errors = append(errors, "exec tool requires a 'command' in config")
		}
	}

	// Validate http tools have a URL
	if cfg.Provider == "http" {
		if url, ok := cfg.Config["url"]; !ok || url == "" {
			errors = append(errors, "http tool requires a 'url' in config")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("tool validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}
