package config

import (
	"strings"
	"testing"
)

func TestValidateCrew_CyclicDepsAllowed(t *testing.T) {
	// Cycles are allowed — loop workflows let agents iterate through each other.
	tests := []struct {
		name  string
		tasks []CrewTaskConfig
	}{
		{
			name: "simple cycle A->B->A",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"b"}},
				{Name: "b", Agent: "dev", DependsOn: []string{"a"}},
			},
		},
		{
			name: "three-way cycle A->B->C->A",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"b"}},
				{Name: "b", Agent: "dev", DependsOn: []string{"c"}},
				{Name: "c", Agent: "dev", DependsOn: []string{"a"}},
			},
		},
		{
			name: "self-referencing",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"a"}},
			},
		},
		{
			name: "no cycle",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev"},
				{Name: "b", Agent: "dev", DependsOn: []string{"a"}},
				{Name: "c", Agent: "dev", DependsOn: []string{"a", "b"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &CrewConfig{
				Name:   "test-crew",
				Agents: []string{"dev"},
				Tasks:  tt.tasks,
			}
			err := ValidateCrew(cfg)
			if err != nil {
				t.Errorf("expected no error (cycles allowed), got: %v", err)
			}
		})
	}
}

func TestValidateCrew_MissingAgentRef(t *testing.T) {
	cfg := &CrewConfig{
		Name:   "test-crew",
		Agents: []string{"developer"},
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "reviewer"}, // reviewer not in agents list
		},
	}
	err := ValidateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for missing agent reference")
	}
	if !strings.Contains(err.Error(), "not in the crew agents list") {
		t.Errorf("expected agent reference error, got: %v", err)
	}
}

func TestValidateCrew_MissingManagerRef(t *testing.T) {
	cfg := &CrewConfig{
		Name:    "test-crew",
		Agents:  []string{"developer"},
		Process: "hierarchical",
		Manager: "lead", // lead not in agents list
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "developer"},
		},
	}
	err := ValidateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for missing manager reference")
	}
	if !strings.Contains(err.Error(), "manager") && !strings.Contains(err.Error(), "not in the crew agents list") {
		t.Errorf("expected manager reference error, got: %v", err)
	}
}

func TestValidateCrew_MissingDependency(t *testing.T) {
	cfg := &CrewConfig{
		Name:   "test-crew",
		Agents: []string{"dev"},
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "dev", DependsOn: []string{"nonexistent"}},
		},
	}
	err := ValidateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "unknown task") {
		t.Errorf("expected unknown task error, got: %v", err)
	}
}

func TestValidateCrew_MaxIterationsValid(t *testing.T) {
	cfg := &CrewConfig{
		Name:          "test-crew",
		Agents:        []string{"dev"},
		MaxIterations: 5,
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "dev"},
		},
	}
	if err := ValidateCrew(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateCrew_MaxIterationsNegative(t *testing.T) {
	cfg := &CrewConfig{
		Name:          "test-crew",
		Agents:        []string{"dev"},
		MaxIterations: -1,
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "dev"},
		},
	}
	err := ValidateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for negative max_iterations")
	}
	if !strings.Contains(err.Error(), "max_iterations must be non-negative") {
		t.Errorf("expected non-negative error, got: %v", err)
	}
}

func TestValidateCrew_MaxIterationsExceedsLimit(t *testing.T) {
	cfg := &CrewConfig{
		Name:          "test-crew",
		Agents:        []string{"dev"},
		MaxIterations: 101,
		Tasks: []CrewTaskConfig{
			{Name: "task1", Agent: "dev"},
		},
	}
	err := ValidateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for max_iterations > 100")
	}
	if !strings.Contains(err.Error(), "max_iterations cannot exceed 100") {
		t.Errorf("expected exceed limit error, got: %v", err)
	}
}

func TestValidateTask_BadTimeout(t *testing.T) {
	cfg := &TaskConfig{
		Name:        "test-task",
		Description: "test",
		Agent:       "dev",
		Timeout:     "not-a-duration",
	}
	err := validateTask(cfg)
	if err == nil {
		t.Fatal("expected error for bad timeout format")
	}
	if !strings.Contains(err.Error(), "invalid timeout format") {
		t.Errorf("expected timeout format error, got: %v", err)
	}
}

func TestValidateTask_ValidTimeout(t *testing.T) {
	cfg := &TaskConfig{
		Name:        "test-task",
		Description: "test",
		Agent:       "dev",
		Timeout:     "5m30s",
	}
	if err := validateTask(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateTool_ExecMissingCommand(t *testing.T) {
	cfg := &ToolConfig{
		Name:        "my-tool",
		Description: "test",
		Provider:    "exec",
		Config:      map[string]interface{}{},
	}
	err := validateTool(cfg)
	if err == nil {
		t.Fatal("expected error for exec tool missing command")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("expected command error, got: %v", err)
	}
}

func TestValidateTool_HttpMissingURL(t *testing.T) {
	cfg := &ToolConfig{
		Name:        "my-tool",
		Description: "test",
		Provider:    "http",
		Config:      map[string]interface{}{},
	}
	err := validateTool(cfg)
	if err == nil {
		t.Fatal("expected error for http tool missing url")
	}
	if !strings.Contains(err.Error(), "url") {
		t.Errorf("expected url error, got: %v", err)
	}
}
