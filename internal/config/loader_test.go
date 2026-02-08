package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := `
name: test-project
version: "2.0"
provider:
  name: anthropic
  model: claude-sonnet-4-20250514
defaults:
  timeout: 10m
  max_retries: 5
logging:
  level: debug
  format: json
state:
  driver: memory
`
	if err := os.WriteFile(filepath.Join(dir, "cadre.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "test-project" {
		t.Errorf("expected name test-project, got %s", cfg.Name)
	}
	if cfg.Version != "2.0" {
		t.Errorf("expected version 2.0, got %s", cfg.Version)
	}
	if cfg.Provider.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", cfg.Provider.Model)
	}
	if cfg.Defaults.Timeout != "10m" {
		t.Errorf("expected timeout 10m, got %s", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.MaxRetries != 5 {
		t.Errorf("expected max_retries 5, got %d", cfg.Defaults.MaxRetries)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected level debug, got %s", cfg.Logging.Level)
	}
	if cfg.State.Driver != "memory" {
		t.Errorf("expected driver memory, got %s", cfg.State.Driver)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	// Should return default config, not error
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "cadre-project" {
		t.Errorf("expected default name, got %s", cfg.Name)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `{{{invalid yaml content`
	if err := os.WriteFile(filepath.Join(dir, "cadre.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_ApplyDefaults(t *testing.T) {
	dir := t.TempDir()
	content := `
name: minimal
`
	if err := os.WriteFile(filepath.Join(dir, "cadre.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Provider.Name != "anthropic" {
		t.Errorf("expected default provider anthropic, got %s", cfg.Provider.Name)
	}
	if cfg.Provider.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected default model, got %s", cfg.Provider.Model)
	}
	if cfg.Defaults.Timeout != "30m" {
		t.Errorf("expected default timeout 30m, got %s", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.MaxRetries != 3 {
		t.Errorf("expected default max_retries 3, got %d", cfg.Defaults.MaxRetries)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default level info, got %s", cfg.Logging.Level)
	}
	if cfg.State.Driver != "sqlite" {
		t.Errorf("expected default driver sqlite, got %s", cfg.State.Driver)
	}
}

func TestLoad_EnvInterpolation(t *testing.T) {
	dir := t.TempDir()
	content := `
name: ${TEST_CADRE_PROJECT_NAME}
provider:
  api_key: ${env.TEST_CADRE_API_KEY}
`
	if err := os.WriteFile(filepath.Join(dir, "cadre.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_CADRE_PROJECT_NAME", "env-project")
	t.Setenv("TEST_CADRE_API_KEY", "sk-test-123")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "env-project" {
		t.Errorf("expected env-project, got %s", cfg.Name)
	}
	if cfg.Provider.APIKey != "sk-test-123" {
		t.Errorf("expected sk-test-123, got %s", cfg.Provider.APIKey)
	}
}

func TestLoad_EnvInterpolation_Unset(t *testing.T) {
	dir := t.TempDir()
	content := `
name: ${UNSET_CADRE_VAR}
`
	if err := os.WriteFile(filepath.Join(dir, "cadre.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should keep original if not found
	if cfg.Name != "${UNSET_CADRE_VAR}" {
		t.Errorf("expected uninterpolated value, got %s", cfg.Name)
	}
}

func TestValidateAgent_MissingName(t *testing.T) {
	cfg := &AgentConfig{Role: "r", Goal: "g"}
	if err := validateAgent(cfg); err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestValidateAgent_Valid(t *testing.T) {
	cfg := &AgentConfig{Name: "a", Role: "r", Goal: "g"}
	if err := validateAgent(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCrew_EmptyProvider(t *testing.T) {
	cfg := &CrewConfig{
		Name:   "test",
		Agents: []string{"a"},
		Tasks: []CrewTaskConfig{
			{Name: "t", Agent: "a"},
		},
	}
	// Empty process defaults to sequential, which is valid
	if err := validateCrew(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCrew_MissingName(t *testing.T) {
	cfg := &CrewConfig{Agents: []string{"a"}}
	if err := validateCrew(cfg); err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestValidateCrew_NoAgents(t *testing.T) {
	cfg := &CrewConfig{Name: "test"}
	if err := validateCrew(cfg); err == nil {
		t.Fatal("expected validation error for no agents")
	}
}
