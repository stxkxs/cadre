package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/crew"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  `Commands for managing and running tasks.`,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	RunE:  runTaskList,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskCreate,
}

var taskRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a task in isolation",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskRun,
}

var (
	taskRunInputs map[string]string
)

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskRunCmd)

	taskRunCmd.Flags().StringToStringVarP(&taskRunInputs, "input", "i", nil, "input values (key=value)")
}

func runTaskList(cmd *cobra.Command, args []string) error {
	tasksDir := "tasks"

	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		fmt.Println("No tasks directory found. Run 'cadre init' first.")
		return nil
	}

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks directory: %w", err)
	}

	fmt.Println("Configured Tasks:")
	fmt.Println("-----------------")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		taskCfg, err := config.LoadTask(name)
		if err != nil {
			fmt.Printf("  %s (error: %v)\n", name, err)
			continue
		}

		fmt.Printf("  %s\n", taskCfg.Name)
		fmt.Printf("    Description: %s\n", taskCfg.Description)
		fmt.Printf("    Agent: %s\n", taskCfg.Agent)
		if len(taskCfg.Dependencies) > 0 {
			fmt.Printf("    Dependencies: %v\n", taskCfg.Dependencies)
		}
		fmt.Printf("    Timeout: %s\n", taskCfg.Timeout)
		fmt.Println()
	}

	return nil
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	taskFile := filepath.Join("tasks", name+".yaml")

	if _, err := os.Stat(taskFile); err == nil {
		return fmt.Errorf("task %s already exists", name)
	}

	content := fmt.Sprintf(`name: %s
description: Describe what this task does
agent: developer
inputs:
  - name: input_name
    type: string
    required: true
outputs:
  - name: output_name
    type: string
dependencies: []
timeout: 30m
retry:
  max_attempts: 3
  backoff: exponential
`, name)

	if err := os.MkdirAll("tasks", 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	if err := os.WriteFile(taskFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	fmt.Printf("Task '%s' created at %s\n", name, taskFile)
	fmt.Println("Edit the file to customize the task configuration.")
	return nil
}

func runTaskRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	taskCfg, err := config.LoadTask(name)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	agentCfg, err := config.LoadAgent(taskCfg.Agent)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", taskCfg.Agent, err)
	}

	logger := telemetry.NewLogger(verbose)
	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer stateMgr.Close()

	logger.Info("Running task", "name", name, "agent", taskCfg.Agent)

	orchestrator, err := crew.NewTaskOrchestrator(cfg, taskCfg, agentCfg, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to create task orchestrator: %w", err)
	}

	// Convert inputs
	inputMap := make(map[string]interface{})
	for k, v := range taskRunInputs {
		inputMap[k] = v
	}

	ctx := context.Background()
	result, err := orchestrator.ExecuteTask(ctx, inputMap)
	if err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	fmt.Println("\nTask completed. Results:")
	for k, v := range result {
		fmt.Printf("  %s: %v\n", k, v)
	}

	return nil
}
