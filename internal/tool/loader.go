package tool

import (
	"fmt"

	"github.com/cadre-oss/cadre/internal/config"
)

// LoadToolsFromConfig creates tools from YAML tool configurations.
func LoadToolsFromConfig(configs []config.ToolConfig) ([]Tool, error) {
	var tools []Tool

	for _, cfg := range configs {
		t, err := createToolFromConfig(&cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create tool %s: %w", cfg.Name, err)
		}
		tools = append(tools, t)
	}

	return tools, nil
}

// createToolFromConfig creates a single tool from config.
func createToolFromConfig(cfg *config.ToolConfig) (Tool, error) {
	switch cfg.Provider {
	case "exec":
		command, _ := cfg.Config["command"].(string)
		if command == "" {
			return nil, fmt.Errorf("exec tool %q requires a 'command' in config", cfg.Name)
		}
		return NewExecTool(cfg.Name, cfg.Description, command), nil

	case "http":
		url, _ := cfg.Config["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("http tool %q requires a 'url' in config", cfg.Name)
		}
		method, _ := cfg.Config["method"].(string)

		headers := make(map[string]string)
		if hdr, ok := cfg.Config["headers"].(map[string]interface{}); ok {
			for k, v := range hdr {
				if s, ok := v.(string); ok {
					headers[k] = s
				}
			}
		}

		return NewHTTPTool(cfg.Name, cfg.Description, url, method, headers), nil

	case "builtin":
		return Get(cfg.Name)

	case "mcp":
		// MCP tools are handled separately by the MCP server.
		return nil, fmt.Errorf("MCP tools are loaded via the MCP server, not the tool loader")

	default:
		return nil, fmt.Errorf("unknown tool provider: %s", cfg.Provider)
	}
}

// RegisterFromConfig loads tools from config and registers them in the default registry.
func RegisterFromConfig(configs []config.ToolConfig) error {
	tools, err := LoadToolsFromConfig(configs)
	if err != nil {
		return err
	}
	for _, t := range tools {
		DefaultRegistry.Register(t)
	}
	return nil
}
