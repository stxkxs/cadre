package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/tool"
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage tools",
	Long:  `Commands for managing and testing tools.`,
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tools",
	RunE:  runToolList,
}

var toolAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a tool from registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolAdd,
}

var toolTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test tool connectivity",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolTest,
}

func init() {
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolAddCmd)
	toolCmd.AddCommand(toolTestCmd)
}

func runToolList(cmd *cobra.Command, args []string) error {
	fmt.Println("Built-in Tools:")
	fmt.Println("---------------")

	builtins := tool.ListBuiltins()
	for _, t := range builtins {
		fmt.Printf("  %s\n", t.Name)
		fmt.Printf("    Description: %s\n", t.Description)
		fmt.Println()
	}

	// List configured tools
	toolsDir := "tools"
	if _, err := os.Stat(toolsDir); err == nil {
		fmt.Println("\nConfigured Tools:")
		fmt.Println("-----------------")

		entries, err := os.ReadDir(toolsDir)
		if err != nil {
			return fmt.Errorf("failed to read tools directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".yaml")
			toolCfg, err := config.LoadTool(name)
			if err != nil {
				fmt.Printf("  %s (error: %v)\n", name, err)
				continue
			}

			fmt.Printf("  %s\n", toolCfg.Name)
			fmt.Printf("    Description: %s\n", toolCfg.Description)
			fmt.Printf("    Provider: %s\n", toolCfg.Provider)
			fmt.Println()
		}
	}

	return nil
}

func runToolAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	toolFile := filepath.Join("tools", name+".yaml")

	if _, err := os.Stat(toolFile); err == nil {
		return fmt.Errorf("tool %s already exists", name)
	}

	// Get template based on known tools
	var content string
	switch name {
	case "github":
		content = `name: github
description: Interact with GitHub repositories
provider: mcp
server: github
capabilities:
  - create_issue
  - create_pr
  - list_prs
  - review_pr
config:
  # token: ${GITHUB_TOKEN}
`
	case "linear":
		content = `name: linear
description: Interact with Linear issues
provider: mcp
server: linear
capabilities:
  - create_issue
  - list_issues
  - update_issue
config:
  # api_key: ${LINEAR_API_KEY}
`
	default:
		content = fmt.Sprintf(`name: %s
description: Describe what this tool does
provider: mcp  # mcp | exec | http
server: %s
capabilities:
  - capability_1
  - capability_2
config:
  # Add tool-specific configuration
`, name, name)
	}

	if err := os.MkdirAll("tools", 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	if err := os.WriteFile(toolFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write tool file: %w", err)
	}

	fmt.Printf("Tool '%s' added at %s\n", name, toolFile)
	fmt.Println("Edit the file to configure the tool.")
	return nil
}

func runToolTest(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if it's a built-in
	if tool.IsBuiltin(name) {
		fmt.Printf("Testing built-in tool: %s\n", name)

		t, err := tool.GetBuiltin(name)
		if err != nil {
			return fmt.Errorf("failed to get tool: %w", err)
		}

		// Run a simple test
		ctx := context.Background()
		result, err := t.Test(ctx)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return nil
		}

		fmt.Printf("OK: %s\n", result)
		return nil
	}

	// Load configured tool
	toolCfg, err := config.LoadTool(name)
	if err != nil {
		return fmt.Errorf("failed to load tool: %w", err)
	}

	fmt.Printf("Testing tool: %s (provider: %s)\n", name, toolCfg.Provider)

	// Test based on provider
	switch toolCfg.Provider {
	case "mcp":
		fmt.Println("MCP tool testing requires active server connection.")
		fmt.Println("Ensure the MCP server is running and configured.")
	case "exec":
		fmt.Println("Exec tool testing not yet implemented.")
	case "http":
		fmt.Println("HTTP tool testing not yet implemented.")
	default:
		fmt.Printf("Unknown provider: %s\n", toolCfg.Provider)
	}

	return nil
}
