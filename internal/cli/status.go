package cli

import (
	"fmt"
	"time"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/spf13/cobra"
)

var (
	statusWatch bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current execution status",
	Long: `Display the current status of crew execution.

Examples:
  cadre status          # Show current status
  cadre status --watch  # Live dashboard`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "watch mode with live updates")
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	if statusWatch {
		return watchStatus(stateMgr)
	}

	return showStatus(stateMgr)
}

func showStatus(stateMgr *state.Manager) error {
	runs, err := stateMgr.ListRuns(10)
	if err != nil {
		return fmt.Errorf("failed to list runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	fmt.Println("Recent Runs:")
	fmt.Println("------------")

	for _, run := range runs {
		statusIcon := getStatusIcon(run.Status)
		fmt.Printf("%s %s  %s  (%s)\n",
			statusIcon,
			run.ID[:8],
			run.CrewName,
			run.Status,
		)
		fmt.Printf("   Started: %s\n", run.StartedAt.Format(time.RFC3339))
		if !run.CompletedAt.IsZero() {
			fmt.Printf("   Completed: %s (duration: %s)\n",
				run.CompletedAt.Format(time.RFC3339),
				run.CompletedAt.Sub(run.StartedAt).Round(time.Second),
			)
		}
		if run.Error != "" {
			fmt.Printf("   Error: %s\n", run.Error)
		}
		fmt.Println()
	}

	// Show active run details
	activeRun, err := stateMgr.GetActiveRun()
	if err == nil && activeRun != nil {
		fmt.Println("\nActive Run Details:")
		fmt.Println("-------------------")
		fmt.Printf("Run ID: %s\n", activeRun.ID)
		fmt.Printf("Crew: %s\n", activeRun.CrewName)
		fmt.Printf("Status: %s\n", activeRun.Status)
		fmt.Println("\nTasks:")

		for _, task := range activeRun.Tasks {
			taskIcon := getStatusIcon(task.Status)
			fmt.Printf("  %s %s (%s)\n", taskIcon, task.Name, task.Status)
			if task.Agent != "" {
				fmt.Printf("      Agent: %s\n", task.Agent)
			}
		}
	}

	return nil
}

func watchStatus(stateMgr *state.Manager) error {
	fmt.Println("Watching for updates... (Ctrl+C to stop)")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// Clear screen (simple approach)
		fmt.Print("\033[H\033[2J")

		if err := showStatus(stateMgr); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		fmt.Printf("\nLast updated: %s\n", time.Now().Format(time.RFC3339))

		<-ticker.C
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "○"
	case "running", "in_progress":
		return "◐"
	case "completed", "success":
		return "●"
	case "failed", "error":
		return "✗"
	case "cancelled":
		return "◌"
	default:
		return "?"
	}
}
