package config

import (
	"strings"
	"testing"
)

func TestValidateCrew_CircularDeps(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []CrewTaskConfig
		wantErr string
	}{
		{
			name: "simple cycle A->B->A",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"b"}},
				{Name: "b", Agent: "dev", DependsOn: []string{"a"}},
			},
			wantErr: "circular dependency",
		},
		{
			name: "three-way cycle A->B->C->A",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"b"}},
				{Name: "b", Agent: "dev", DependsOn: []string{"c"}},
				{Name: "c", Agent: "dev", DependsOn: []string{"a"}},
			},
			wantErr: "circular dependency",
		},
		{
			name: "self-referencing",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev", DependsOn: []string{"a"}},
			},
			wantErr: "circular dependency",
		},
		{
			name: "no cycle",
			tasks: []CrewTaskConfig{
				{Name: "a", Agent: "dev"},
				{Name: "b", Agent: "dev", DependsOn: []string{"a"}},
				{Name: "c", Agent: "dev", DependsOn: []string{"a", "b"}},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &CrewConfig{
				Name:   "test-crew",
				Agents: []string{"dev"},
				Tasks:  tt.tasks,
			}
			err := validateCrew(cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
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
	err := validateCrew(cfg)
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
	err := validateCrew(cfg)
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
	err := validateCrew(cfg)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "unknown task") {
		t.Errorf("expected unknown task error, got: %v", err)
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
