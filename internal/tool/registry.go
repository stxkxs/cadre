package tool

import (
	"context"

	"github.com/stxkxs/cadre/internal/tool/builtin"
)

// builtinTools holds references to built-in tools
var builtinTools = map[string]Tool{}

func init() {
	// Register built-in tools
	RegisterBuiltins()
}

// RegisterBuiltins registers all built-in tools
func RegisterBuiltins() {
	builtinTools["file_read"] = builtin.NewFileReadTool()
	builtinTools["file_write"] = builtin.NewFileWriteTool()
	builtinTools["bash"] = builtin.NewBashTool()
	builtinTools["grep"] = builtin.NewGrepTool()

	// Register in default registry
	for _, t := range builtinTools {
		DefaultRegistry.Register(t)
	}
}

// IsBuiltin checks if a tool name is a built-in tool
func IsBuiltin(name string) bool {
	_, ok := builtinTools[name]
	return ok
}

// GetBuiltin returns a built-in tool by name
func GetBuiltin(name string) (Tool, error) {
	return Get(name)
}

// ListBuiltins returns information about all built-in tools
func ListBuiltins() []ToolInfo {
	infos := make([]ToolInfo, 0, len(builtinTools))
	for name, t := range builtinTools {
		infos = append(infos, ToolInfo{
			Name:        name,
			Description: t.Description(),
		})
	}
	return infos
}

// ExecuteTool executes a tool by name with the given arguments
func ExecuteTool(ctx context.Context, name string, args string) (string, error) {
	tool, err := Get(name)
	if err != nil {
		return "", err
	}
	return tool.Execute(ctx, []byte(args))
}
