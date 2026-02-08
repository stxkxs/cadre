package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cadreErrors "github.com/cadre-oss/cadre/internal/errors"
)

// Tool represents a capability an agent can invoke
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns a description for the LLM
	Description() string

	// Parameters returns JSON schema for the tool parameters
	Parameters() map[string]interface{}

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args json.RawMessage) (string, error)

	// Test verifies the tool is working
	Test(ctx context.Context) (string, error)
}

// ToolInfo provides basic tool information
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		available := make([]string, 0, len(r.tools))
		for n := range r.tools {
			available = append(available, n)
		}
		return nil, cadreErrors.New(cadreErrors.CodeToolNotFound,
			fmt.Sprintf("tool not found: %s", name)).
			WithSuggestion(fmt.Sprintf("Available tools: %s", strings.Join(available, ", ")))
	}
	return tool, nil
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Has checks if a tool is registered
func (r *Registry) Has(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// ForLLM returns tool definitions formatted for LLM consumption
func (r *Registry) ForLLM() []map[string]interface{} {
	tools := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": t.Parameters(),
			},
		})
	}
	return tools
}

// DefaultRegistry is the global tool registry
var DefaultRegistry = NewRegistry()

// Register registers a tool in the default registry
func Register(tool Tool) {
	DefaultRegistry.Register(tool)
}

// Get retrieves a tool from the default registry
func Get(name string) (Tool, error) {
	return DefaultRegistry.Get(name)
}

// List returns all tools in the default registry
func List() []Tool {
	return DefaultRegistry.List()
}
