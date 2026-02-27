package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
	Long:  `Commands for managing and interacting with agents.`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agents",
	RunE:  runAgentList,
}

var agentCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new agent interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentCreate,
}

var agentTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test agent in isolation",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentTest,
}

var agentChatCmd = &cobra.Command{
	Use:   "chat <name>",
	Short: "Interactive chat with agent",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentChat,
}

func init() {
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentTestCmd)
	agentCmd.AddCommand(agentChatCmd)
}

func runAgentList(cmd *cobra.Command, args []string) error {
	agentsDir := "agents"

	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		fmt.Println("No agents directory found. Run 'cadre init' first.")
		return nil
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return fmt.Errorf("failed to read agents directory: %w", err)
	}

	fmt.Println("Configured Agents:")
	fmt.Println("------------------")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		agentCfg, err := config.LoadAgent(name)
		if err != nil {
			fmt.Printf("  %s (error: %v)\n", name, err)
			continue
		}

		fmt.Printf("  %s\n", agentCfg.Name)
		fmt.Printf("    Role: %s\n", agentCfg.Role)
		fmt.Printf("    Tools: %v\n", agentCfg.Tools)
		fmt.Println()
	}

	return nil
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	agentFile := filepath.Join("agents", name+".yaml")

	if _, err := os.Stat(agentFile); err == nil {
		return fmt.Errorf("agent %s already exists", name)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Creating agent: %s\n\n", name)

	fmt.Print("Role (e.g., Senior Software Engineer): ")
	role, _ := reader.ReadString('\n')
	role = strings.TrimSpace(role)

	fmt.Print("Goal (what should this agent accomplish): ")
	goal, _ := reader.ReadString('\n')
	goal = strings.TrimSpace(goal)

	fmt.Print("Backstory (press Enter twice to finish):\n")
	var backstoryLines []string
	for {
		line, _ := reader.ReadString('\n')
		if line == "\n" {
			break
		}
		backstoryLines = append(backstoryLines, strings.TrimRight(line, "\n"))
	}
	backstory := strings.Join(backstoryLines, "\n")

	fmt.Print("Tools (comma-separated, e.g., file_read,file_write,bash): ")
	toolsStr, _ := reader.ReadString('\n')
	tools := strings.Split(strings.TrimSpace(toolsStr), ",")
	for i := range tools {
		tools[i] = strings.TrimSpace(tools[i])
	}

	// Generate YAML
	content := fmt.Sprintf(`name: %s
role: %s
goal: %s
backstory: |
  %s
tools:
`, name, role, goal, strings.ReplaceAll(backstory, "\n", "\n  "))

	for _, tool := range tools {
		if tool != "" {
			content += fmt.Sprintf("  - %s\n", tool)
		}
	}

	content += `memory:
  type: conversation
  max_tokens: 100000
`

	if err := os.MkdirAll("agents", 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	fmt.Printf("\nAgent '%s' created at %s\n", name, agentFile)
	return nil
}

func runAgentTest(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	agentCfg, err := config.LoadAgent(name)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}

	logger := telemetry.NewLogger(verbose)
	logger.Info("Testing agent", "name", name)

	// Create a simple test task
	testPrompt := fmt.Sprintf(`You are %s.

Role: %s
Goal: %s

This is a test to verify you are configured correctly.
Please respond with:
1. A brief confirmation that you understand your role
2. List the tools you have access to
3. A simple example of what you could help with

Keep your response brief and focused.`, agentCfg.Name, agentCfg.Role, agentCfg.Goal)

	// Initialize provider and run test
	stateMgr, err := state.NewManager("memory", "")
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer stateMgr.Close()

	runtime, err := crew.NewAgentRuntime(cfg, agentCfg, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to create agent runtime: %w", err)
	}

	ctx := context.Background()

	fmt.Println("\nAgent Response:")
	fmt.Println("---------------")
	_, err = runtime.StreamExecute(ctx, testPrompt, func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		return fmt.Errorf("agent test failed: %w", err)
	}
	fmt.Println()

	return nil
}

func runAgentChat(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	agentCfg, err := config.LoadAgent(name)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}

	logger := telemetry.NewLogger(verbose)
	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer stateMgr.Close()

	session, err := crew.NewInteractiveSession(cfg, agentCfg, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return session.Run(context.Background())
}
