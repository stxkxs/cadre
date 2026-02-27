package cli

import (
	"fmt"
	"os"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for viewing and modifying configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all configuration files",
	RunE:  runConfigValidate,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Pretty print config
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println("Current Configuration:")
	fmt.Println("----------------------")
	fmt.Println(string(out))

	// Show config file location
	if viper.ConfigFileUsed() != "" {
		fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Load existing config
	configFile := "cadre.yaml"
	if viper.ConfigFileUsed() != "" {
		configFile = viper.ConfigFileUsed()
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Set the value (supports nested keys with dot notation)
	setNestedValue(cfg, key, value)

	// Write back
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, out, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	errors := []string{}

	// Validate main config
	_, err := config.Load(".")
	if err != nil {
		errors = append(errors, fmt.Sprintf("cadre.yaml: %v", err))
	} else {
		fmt.Println("cadre.yaml: OK")
	}

	// Validate agents
	if entries, err := os.ReadDir("agents"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !isYAML(entry.Name()) {
				continue
			}
			name := stripYAML(entry.Name())
			if _, err := config.LoadAgent(name); err != nil {
				errors = append(errors, fmt.Sprintf("agents/%s: %v", entry.Name(), err))
			} else {
				fmt.Printf("agents/%s: OK\n", entry.Name())
			}
		}
	}

	// Validate tasks
	if entries, err := os.ReadDir("tasks"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !isYAML(entry.Name()) {
				continue
			}
			name := stripYAML(entry.Name())
			if _, err := config.LoadTask(name); err != nil {
				errors = append(errors, fmt.Sprintf("tasks/%s: %v", entry.Name(), err))
			} else {
				fmt.Printf("tasks/%s: OK\n", entry.Name())
			}
		}
	}

	// Validate crews
	if entries, err := os.ReadDir("crews"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !isYAML(entry.Name()) {
				continue
			}
			name := stripYAML(entry.Name())
			if _, err := config.LoadCrew(name); err != nil {
				errors = append(errors, fmt.Sprintf("crews/%s: %v", entry.Name(), err))
			} else {
				fmt.Printf("crews/%s: OK\n", entry.Name())
			}
		}
	}

	// Validate tools
	if entries, err := os.ReadDir("tools"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !isYAML(entry.Name()) {
				continue
			}
			name := stripYAML(entry.Name())
			if _, err := config.LoadTool(name); err != nil {
				errors = append(errors, fmt.Sprintf("tools/%s: %v", entry.Name(), err))
			} else {
				fmt.Printf("tools/%s: OK\n", entry.Name())
			}
		}
	}

	if len(errors) > 0 {
		fmt.Println("\nValidation Errors:")
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("validation failed with %d errors", len(errors))
	}

	fmt.Println("\nAll configurations valid.")
	return nil
}

func setNestedValue(m map[string]interface{}, key, value string) {
	// Simple implementation - could be enhanced for deeper nesting
	parts := splitKey(key)
	if len(parts) == 1 {
		m[key] = value
		return
	}

	current := m
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := current[parts[i]]; !ok {
			current[parts[i]] = make(map[string]interface{})
		}
		if next, ok := current[parts[i]].(map[string]interface{}); ok {
			current = next
		} else {
			return
		}
	}
	current[parts[len(parts)-1]] = value
}

func splitKey(key string) []string {
	var parts []string
	current := ""
	for _, c := range key {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func isYAML(name string) bool {
	return len(name) > 5 && (name[len(name)-5:] == ".yaml" || name[len(name)-4:] == ".yml")
}

func stripYAML(name string) string {
	if len(name) > 5 && name[len(name)-5:] == ".yaml" {
		return name[:len(name)-5]
	}
	if len(name) > 4 && name[len(name)-4:] == ".yml" {
		return name[:len(name)-4]
	}
	return name
}
