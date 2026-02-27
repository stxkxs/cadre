package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stxkxs/cadre/internal/sprint"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Manage sprint execution sessions",
	Long: `Launch and manage multi-agent sprint sessions.

Sprint configs define workstreams, phases, and gate checks.
Each workstream spawns a Claude Code instance with IPC tools
for inter-agent communication via a shared SQLite database.`,
}

var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available sprint configurations",
	RunE:  runSprintList,
}

var sprintStartCmd = &cobra.Command{
	Use:   "start <sprint-name>",
	Short: "Start a sprint session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSprintStart,
}

var sprintStatusCmd = &cobra.Command{
	Use:   "status <sprint-name>",
	Short: "Show agent statuses for a sprint session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSprintStatus,
}

var sprintStopCmd = &cobra.Command{
	Use:   "stop <sprint-name>",
	Short: "Stop a running sprint session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSprintStop,
}

var sprintGateCmd = &cobra.Command{
	Use:   "gate <sprint-name>",
	Short: "Run gate checks for a sprint",
	Args:  cobra.ExactArgs(1),
	RunE:  runSprintGate,
}

var sprintDryRun bool

func init() {
	sprintStartCmd.Flags().BoolVar(&sprintDryRun, "dry-run", false, "validate config without launching agents")

	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintStartCmd)
	sprintCmd.AddCommand(sprintStatusCmd)
	sprintCmd.AddCommand(sprintStopCmd)
	sprintCmd.AddCommand(sprintGateCmd)
}

func runSprintList(_ *cobra.Command, _ []string) error {
	names, err := sprint.ListSprints()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No sprint configurations found.")
		fmt.Println("Create one at sprints/<name>.yaml or .cadre-workspace/sprints/<name>.yaml")
		return nil
	}

	fmt.Println("Available sprints:")
	for _, name := range names {
		cfg, err := sprint.LoadSprint(name)
		if err != nil {
			fmt.Printf("  %s (error: %s)\n", name, err)
			continue
		}
		fmt.Printf("  %s - %s (%d workstreams, %d phases)\n",
			cfg.Name, cfg.Description, len(cfg.Workstreams), len(cfg.Phases))
	}
	return nil
}

func runSprintStart(_ *cobra.Command, args []string) error {
	logger := telemetry.NewLogger(verbose)
	name := args[0]

	cfg, err := sprint.LoadSprint(name)
	if err != nil {
		return err
	}

	if sprintDryRun {
		fmt.Printf("Sprint: %s\n", cfg.Name)
		fmt.Printf("Description: %s\n", cfg.Description)
		fmt.Printf("Workstreams:\n")
		for _, ws := range cfg.Workstreams {
			fmt.Printf("  - %s (crew: %s, branch: %s, issues: %v)\n", ws.Name, ws.Crew, ws.Branch, ws.Issues)
		}
		fmt.Printf("Phases:\n")
		for _, ph := range cfg.Phases {
			parallel := ""
			if ph.Parallel {
				parallel = " [parallel]"
			}
			fmt.Printf("  - %s: %v%s\n", ph.Name, ph.Workstreams, parallel)
		}
		if len(cfg.Gate.Checks) > 0 {
			fmt.Printf("Gate checks:\n")
			for _, c := range cfg.Gate.Checks {
				fmt.Printf("  - %s\n", c)
			}
		}
		fmt.Println("\nConfig is valid. Use without --dry-run to launch.")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	session, err := sprint.NewSession(cfg, logger)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Handle shutdown
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping agents...")
		cancel()
		_ = session.Stop()
	}()

	logger.Info("Starting sprint", "name", name, "workstreams", len(cfg.Workstreams))

	if err := session.Start(ctx); err != nil {
		return fmt.Errorf("sprint execution failed: %w", err)
	}

	// Wait for all agents
	if err := session.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for agents: %w", err)
	}

	// Run gate checks
	if err := session.RunGate(ctx); err != nil {
		return err
	}

	logger.Info("Sprint completed", "name", name)
	return nil
}

func runSprintStatus(_ *cobra.Command, args []string) error {
	logger := telemetry.NewLogger(verbose)
	name := args[0]

	cfg, err := sprint.LoadSprint(name)
	if err != nil {
		return err
	}

	session, err := sprint.NewSession(cfg, logger)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}

	statuses, err := session.Status()
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	if len(statuses) == 0 {
		fmt.Printf("No agents found for sprint %s. Is it running?\n", name)
		return nil
	}

	fmt.Printf("Sprint: %s\n\n", name)
	fmt.Printf("%-20s %-12s %-8s %s\n", "AGENT", "STATUS", "PID", "TASK/SUMMARY")
	fmt.Printf("%-20s %-12s %-8s %s\n", "-----", "------", "---", "------------")

	for _, s := range statuses {
		detail := s.Task
		if s.Summary != "" {
			detail = s.Summary
		}
		running := ""
		if s.IsRunning {
			running = " (running)"
		}
		fmt.Printf("%-20s %-12s %-8d %s%s\n", s.Name, s.Status, s.PID, detail, running)
	}

	return nil
}

func runSprintStop(_ *cobra.Command, args []string) error {
	logger := telemetry.NewLogger(verbose)
	name := args[0]

	cfg, err := sprint.LoadSprint(name)
	if err != nil {
		return err
	}

	session, err := sprint.NewSession(cfg, logger)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}

	logger.Info("Stopping sprint", "name", name)
	return session.Stop()
}

func runSprintGate(_ *cobra.Command, args []string) error {
	logger := telemetry.NewLogger(verbose)
	name := args[0]

	cfg, err := sprint.LoadSprint(name)
	if err != nil {
		return err
	}

	session, err := sprint.NewSession(cfg, logger)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}

	return session.RunGate(context.Background())
}
