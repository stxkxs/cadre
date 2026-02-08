package config

import "time"

// Config represents the main project configuration (crew.yaml)
type Config struct {
	Name     string          `yaml:"name" json:"name"`
	Version  string          `yaml:"version" json:"version"`
	Provider ProviderConfig  `yaml:"provider" json:"provider"`
	Defaults DefaultsConfig  `yaml:"defaults" json:"defaults"`
	Logging  LoggingConfig   `yaml:"logging" json:"logging"`
	State    StateConfig     `yaml:"state" json:"state"`
	Hooks    HooksConfig     `yaml:"hooks" json:"hooks"`
}

// HooksConfig configures lifecycle event hooks.
type HooksConfig struct {
	Enabled bool         `yaml:"enabled" json:"enabled"`
	Hooks   []HookConfig `yaml:"hooks" json:"hooks"`
}

// HookConfig defines a single hook.
type HookConfig struct {
	Name     string   `yaml:"name" json:"name"`
	Type     string   `yaml:"type" json:"type"`         // shell, webhook, log, pause
	Events   []string `yaml:"events" json:"events"`     // event types to match
	Blocking bool     `yaml:"blocking" json:"blocking"`
	Command  string   `yaml:"command,omitempty" json:"command,omitempty"`  // for shell hooks
	URL      string   `yaml:"url,omitempty" json:"url,omitempty"`         // for webhook hooks
	Message  string   `yaml:"message,omitempty" json:"message,omitempty"` // for pause hooks
	Level    string   `yaml:"level,omitempty" json:"level,omitempty"`     // for log hooks (debug, info, warn)
}

// ProviderConfig configures the LLM provider
type ProviderConfig struct {
	Name    string `yaml:"name" json:"name"`                           // anthropic, openai, etc.
	Model   string `yaml:"model" json:"model"`                        // claude-sonnet-4-20250514, etc.
	APIKey  string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	WorkDir string `yaml:"work_dir,omitempty" json:"work_dir,omitempty"` // working directory for CLI providers
}

// DefaultsConfig provides default values
type DefaultsConfig struct {
	Timeout    string `yaml:"timeout" json:"timeout"`         // e.g., "30m"
	MaxRetries int    `yaml:"max_retries" json:"max_retries"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // debug, info, warn, error
	Format string `yaml:"format" json:"format"` // text, json
}

// StateConfig configures state storage
type StateConfig struct {
	Driver string `yaml:"driver" json:"driver"` // sqlite, postgres, memory
	Path   string `yaml:"path" json:"path"`     // connection string or file path
}

// AgentConfig represents an agent definition
type AgentConfig struct {
	Name             string       `yaml:"name" json:"name"`
	Role             string       `yaml:"role" json:"role"`
	Goal             string       `yaml:"goal" json:"goal"`
	Backstory        string       `yaml:"backstory" json:"backstory"`
	CompactBackstory string       `yaml:"compact_backstory,omitempty" json:"compact_backstory,omitempty"`
	Tools            []string     `yaml:"tools" json:"tools"`
	Memory           MemoryConfig `yaml:"memory" json:"memory"`
	QuickMode        bool         `yaml:"quick_mode,omitempty" json:"quick_mode,omitempty"`
	Provider         string       `yaml:"provider,omitempty" json:"provider,omitempty"`               // "anthropic" or "claudecode" (empty = use global)
	ProviderModel    string       `yaml:"provider_model,omitempty" json:"provider_model,omitempty"`   // model override
	APIKey           string       `yaml:"api_key,omitempty" json:"api_key,omitempty"`                 // API key override (anthropic only)
	WorkDir          string       `yaml:"work_dir,omitempty" json:"work_dir,omitempty"`               // working directory override (claudecode only)
}

// MemoryConfig configures agent memory
type MemoryConfig struct {
	Type      string `yaml:"type" json:"type"`
	MaxTokens int    `yaml:"max_tokens" json:"max_tokens"`
}

// TaskConfig represents a task definition
type TaskConfig struct {
	Name         string             `yaml:"name" json:"name"`
	Description  string             `yaml:"description" json:"description"`
	Agent        string             `yaml:"agent" json:"agent"`
	Inputs       []InputConfig      `yaml:"inputs" json:"inputs"`
	Outputs      []OutputConfig     `yaml:"outputs" json:"outputs"`
	OutputSchema *OutputSchemaConfig `yaml:"output_schema,omitempty" json:"output_schema,omitempty"`
	Dependencies []string           `yaml:"dependencies" json:"dependencies"`
	Timeout      string             `yaml:"timeout" json:"timeout"`
	Retry        RetryConfig        `yaml:"retry" json:"retry"`
}

// OutputSchemaConfig configures structured output requirements
type OutputSchemaConfig struct {
	Format string              `yaml:"format" json:"format"`
	Strict bool                `yaml:"strict" json:"strict"`
	Fields []OutputFieldConfig `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// OutputFieldConfig defines an expected field in structured output
type OutputFieldConfig struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
}

// InputConfig defines a task input
type InputConfig struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Required bool   `yaml:"required" json:"required"`
	Default  string `yaml:"default,omitempty" json:"default,omitempty"`
}

// OutputConfig defines a task output
type OutputConfig struct {
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type" json:"type"`
}

// RetryConfig configures task retry behavior
type RetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts" json:"max_attempts"`
	Backoff     string `yaml:"backoff" json:"backoff"`
}

// CrewConfig represents a crew definition
type CrewConfig struct {
	Name          string           `yaml:"name" json:"name"`
	Description   string           `yaml:"description" json:"description"`
	Agents        []string         `yaml:"agents" json:"agents"`
	Process       string           `yaml:"process" json:"process"`
	Concurrency   int              `yaml:"concurrency" json:"concurrency"`
	ErrorStrategy string           `yaml:"error_strategy" json:"error_strategy"`
	Tasks         []CrewTaskConfig `yaml:"tasks" json:"tasks"`
	Manager       string           `yaml:"manager,omitempty" json:"manager,omitempty"`
}

// CrewTaskConfig represents a task within a crew
type CrewTaskConfig struct {
	Name      string   `yaml:"name" json:"name"`
	Agent     string   `yaml:"agent" json:"agent"`
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// ToolConfig represents a tool definition
type ToolConfig struct {
	Name         string                 `yaml:"name" json:"name"`
	Description  string                 `yaml:"description" json:"description"`
	Provider     string                 `yaml:"provider" json:"provider"`
	Server       string                 `yaml:"server,omitempty" json:"server,omitempty"`
	Capabilities []string               `yaml:"capabilities" json:"capabilities"`
	Config       map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// ParsedTimeout converts a timeout string to time.Duration
func (t *TaskConfig) ParsedTimeout() (time.Duration, error) {
	if t.Timeout == "" {
		return 30 * time.Minute, nil // default
	}
	return time.ParseDuration(t.Timeout)
}

// ParsedTimeout converts a timeout string to time.Duration
func (d *DefaultsConfig) ParsedTimeout() (time.Duration, error) {
	if d.Timeout == "" {
		return 30 * time.Minute, nil // default
	}
	return time.ParseDuration(d.Timeout)
}
