package cli

import (
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/stxkxs/cadre/internal/tool"
)

// registerCustomTools loads exec/http tools from tools/*.yaml and registers
// them in the default tool registry so agents can reference them by name.
func registerCustomTools(logger *telemetry.Logger) {
	names, err := config.LoadToolList()
	if err != nil {
		logger.Debug("Failed to load tool list", "error", err)
		return
	}
	if len(names) == 0 {
		return
	}

	var configs []config.ToolConfig
	for _, name := range names {
		cfg, err := config.LoadTool(name)
		if err != nil {
			logger.Debug("Skipping tool", "name", name, "error", err)
			continue
		}
		configs = append(configs, *cfg)
	}

	if len(configs) > 0 {
		if err := tool.RegisterFromConfig(configs); err != nil {
			logger.Warn("Failed to register custom tools", "error", err)
		} else {
			logger.Debug("Registered custom tools", "count", len(configs))
		}
	}
}
