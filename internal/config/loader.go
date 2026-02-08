package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load loads the main project configuration
func Load(dir string) (*Config, error) {
	configFile := filepath.Join(dir, "cadre.yaml")

	content, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if no file exists
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Interpolate environment variables
	content = []byte(interpolateEnv(string(content)))

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// LoadAgent loads an agent configuration
func LoadAgent(name string) (*AgentConfig, error) {
	agentFile := filepath.Join("agents", name+".yaml")

	content, err := os.ReadFile(agentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file: %w", err)
	}

	content = []byte(interpolateEnv(string(content)))

	var cfg AgentConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse agent config: %w", err)
	}

	if err := validateAgent(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadTask loads a task configuration
func LoadTask(name string) (*TaskConfig, error) {
	taskFile := filepath.Join("tasks", name+".yaml")

	content, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	content = []byte(interpolateEnv(string(content)))

	var cfg TaskConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse task config: %w", err)
	}

	if err := validateTask(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadCrew loads a crew configuration
func LoadCrew(name string) (*CrewConfig, error) {
	// Try both name and "default" crew file
	crewFile := filepath.Join("crews", name+".yaml")
	if _, err := os.Stat(crewFile); os.IsNotExist(err) {
		// Try development.yaml as default
		crewFile = filepath.Join("crews", "development.yaml")
	}

	content, err := os.ReadFile(crewFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read crew file: %w", err)
	}

	content = []byte(interpolateEnv(string(content)))

	var cfg CrewConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse crew config: %w", err)
	}

	if err := validateCrew(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadTool loads a tool configuration
func LoadTool(name string) (*ToolConfig, error) {
	toolFile := filepath.Join("tools", name+".yaml")

	content, err := os.ReadFile(toolFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool file: %w", err)
	}

	content = []byte(interpolateEnv(string(content)))

	var cfg ToolConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse tool config: %w", err)
	}

	if err := validateTool(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// interpolateEnv replaces ${env.VAR} and ${VAR} with environment values
func interpolateEnv(content string) string {
	// Match ${env.VAR} pattern
	envPattern := regexp.MustCompile(`\$\{env\.([^}]+)\}`)
	content = envPattern.ReplaceAllStringFunc(content, func(match string) string {
		varName := envPattern.FindStringSubmatch(match)[1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // keep original if not found
	})

	// Match ${VAR} pattern
	varPattern := regexp.MustCompile(`\$\{([^}]+)\}`)
	content = varPattern.ReplaceAllStringFunc(content, func(match string) string {
		varName := varPattern.FindStringSubmatch(match)[1]
		// Skip if it's not an env reference
		if strings.HasPrefix(varName, "input.") || strings.HasPrefix(varName, "output.") {
			return match
		}
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})

	return content
}

func defaultConfig() *Config {
	return &Config{
		Name:    "cadre-project",
		Version: "1.0",
		Provider: ProviderConfig{
			Name:  "anthropic",
			Model: "claude-sonnet-4-20250514",
		},
		Defaults: DefaultsConfig{
			Timeout:    "30m",
			MaxRetries: 3,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		State: StateConfig{
			Driver: "sqlite",
			Path:   ".cadre/state.db",
		},
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Provider.Name == "" {
		cfg.Provider.Name = "anthropic"
	}
	if cfg.Provider.Model == "" {
		cfg.Provider.Model = "claude-sonnet-4-20250514"
	}
	if cfg.Defaults.Timeout == "" {
		cfg.Defaults.Timeout = "30m"
	}
	if cfg.Defaults.MaxRetries == 0 {
		cfg.Defaults.MaxRetries = 3
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "text"
	}
	if cfg.State.Driver == "" {
		cfg.State.Driver = "sqlite"
	}
	if cfg.State.Path == "" {
		cfg.State.Path = ".cadre/state.db"
	}

	// Load API key from environment if not set
	if cfg.Provider.APIKey == "" {
		cfg.Provider.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
}
