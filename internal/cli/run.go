package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/provider/claudecode"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	runTask       string
	runAgent      string
	runResume     string
	runInputs     map[string]string
	runDryRun     bool
	runMode       string
	runOutputJSON bool
)

var runCmd = &cobra.Command{
	Use:   "run [crew-name]",
	Short: "Run a crew or task",
	Long: `Run a crew workflow, single task, or interactive agent session.

Execution Modes:
  direct     - Execute using Anthropic API directly (requires API key)
  claudecode - Generate execution plan for Claude Code's Task tool

Examples:
  cadre run                           # Run default crew (direct mode)
  cadre run development               # Run specific crew
  cadre run --mode claudecode         # Generate Claude Code execution plan
  cadre run --mode claudecode --json  # Output plan as JSON
  cadre run --task implement          # Run single task
  cadre run --agent developer         # Interactive agent session
  cadre run --resume abc123           # Resume from checkpoint`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringVarP(&runTask, "task", "t", "", "run a single task")
	runCmd.Flags().StringVarP(&runAgent, "agent", "a", "", "interactive session with agent")
	runCmd.Flags().StringVarP(&runResume, "resume", "r", "", "resume from checkpoint ID")
	runCmd.Flags().StringToStringVarP(&runInputs, "input", "i", nil, "input values (key=value)")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "show what would be executed without running")
	runCmd.Flags().StringVarP(&runMode, "mode", "m", "direct", "execution mode: direct (API) or claudecode")
	runCmd.Flags().BoolVar(&runOutputJSON, "json", false, "output as JSON (claudecode mode only)")
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, saving checkpoint...")
		cancel()
	}()

	// Initialize telemetry
	logger := telemetry.NewLogger(verbose)
	registerCustomTools(logger)

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize state manager
	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	// Determine what to run
	crewName := "default"
	if len(args) > 0 {
		crewName = args[0]
	}

	// Handle resume
	if runResume != "" {
		return resumeFromCheckpoint(ctx, cfg, stateMgr, logger, runResume)
	}

	// Handle single agent session
	if runAgent != "" {
		return runAgentSession(ctx, cfg, stateMgr, logger, runAgent)
	}

	// Handle single task
	if runTask != "" {
		return runSingleTask(ctx, cfg, stateMgr, logger, runTask, runInputs)
	}

	// Check execution mode
	if runMode == "claudecode" {
		return runClaudeCodeMode(ctx, cfg, logger, crewName, runInputs, runOutputJSON)
	}

	// Run full crew (direct mode)
	return runCrew(ctx, cfg, stateMgr, logger, crewName, runInputs, runDryRun)
}

func runCrew(ctx context.Context, cfg *config.Config, stateMgr *state.Manager, logger *telemetry.Logger, crewName string, inputs map[string]string, dryRun bool) error {
	logger.Info("Loading crew", "name", crewName)

	// Load crew configuration
	crewCfg, err := config.LoadCrew(crewName)
	if err != nil {
		return fmt.Errorf("failed to load crew %s: %w", crewName, err)
	}

	if dryRun {
		fmt.Println("Dry run - would execute:")
		fmt.Printf("  Crew: %s\n", crewCfg.Name)
		fmt.Printf("  Process: %s\n", crewCfg.Process)
		fmt.Printf("  Agents: %v\n", crewCfg.Agents)
		fmt.Printf("  Tasks:\n")
		for _, t := range crewCfg.Tasks {
			fmt.Printf("    - %s (agent: %s)\n", t.Name, t.Agent)
		}
		return nil
	}

	// Create orchestrator
	orchestrator, err := crew.NewOrchestrator(cfg, crewCfg, stateMgr, logger, nil)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Convert inputs
	inputMap := make(map[string]interface{})
	for k, v := range inputs {
		inputMap[k] = v
	}

	// Execute
	result, err := orchestrator.Execute(ctx, inputMap)
	if err != nil {
		// Save checkpoint on error
		if checkpointErr := stateMgr.SaveCheckpoint(orchestrator.GetState()); checkpointErr != nil {
			logger.Error("Failed to save checkpoint", "error", checkpointErr)
		}
		return fmt.Errorf("crew execution failed: %w", err)
	}

	logger.Info("Crew completed successfully")

	// Print results
	if result != nil {
		fmt.Println("\nResults:")
		for k, v := range result {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

func runSingleTask(ctx context.Context, cfg *config.Config, stateMgr *state.Manager, logger *telemetry.Logger, taskName string, inputs map[string]string) error {
	logger.Info("Running single task", "name", taskName)

	// Load task configuration
	taskCfg, err := config.LoadTask(taskName)
	if err != nil {
		return fmt.Errorf("failed to load task %s: %w", taskName, err)
	}

	// Load the agent for this task
	agentCfg, err := config.LoadAgent(taskCfg.Agent)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", taskCfg.Agent, err)
	}

	// Create orchestrator for single task
	orchestrator, err := crew.NewTaskOrchestrator(cfg, taskCfg, agentCfg, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to create task orchestrator: %w", err)
	}

	// Convert inputs
	inputMap := make(map[string]interface{})
	for k, v := range inputs {
		inputMap[k] = v
	}

	// Execute
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

func runAgentSession(ctx context.Context, cfg *config.Config, stateMgr *state.Manager, logger *telemetry.Logger, agentName string) error {
	logger.Info("Starting interactive agent session", "agent", agentName)

	// Load agent configuration
	agentCfg, err := config.LoadAgent(agentName)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", agentName, err)
	}

	// Create interactive session
	session, err := crew.NewInteractiveSession(cfg, agentCfg, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return session.Run(ctx)
}

func runClaudeCodeMode(ctx context.Context, cfg *config.Config, logger *telemetry.Logger, crewName string, inputs map[string]string, outputJSON bool) error {
	logger.Info("Generating Claude Code execution plan", "crew", crewName)

	// Load crew configuration
	crewCfg, err := config.LoadCrew(crewName)
	if err != nil {
		return fmt.Errorf("failed to load crew %s: %w", crewName, err)
	}

	// Create Claude Code provider
	provider, err := claudecode.NewProvider(crewCfg)
	if err != nil {
		return fmt.Errorf("failed to create Claude Code provider: %w", err)
	}

	// Convert inputs
	inputMap := make(map[string]interface{})
	for k, v := range inputs {
		inputMap[k] = v
	}

	// Generate execution plan
	result, err := provider.Execute(ctx, inputMap)
	if err != nil {
		return fmt.Errorf("failed to generate execution plan: %w", err)
	}

	// Output based on format
	if outputJSON {
		fmt.Println(result.JSON)
	} else {
		fmt.Println(result.Instructions)
	}

	return nil
}

func resumeFromCheckpoint(ctx context.Context, cfg *config.Config, stateMgr *state.Manager, logger *telemetry.Logger, checkpointID string) error {
	logger.Info("Resuming from checkpoint", "id", checkpointID)

	checkpoint, err := stateMgr.LoadCheckpoint(checkpointID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint %s: %w", checkpointID, err)
	}

	// Recreate orchestrator from checkpoint
	orchestrator, err := crew.NewOrchestratorFromCheckpoint(cfg, checkpoint, stateMgr, logger)
	if err != nil {
		return fmt.Errorf("failed to restore from checkpoint: %w", err)
	}

	// Resume execution
	result, err := orchestrator.Resume(ctx)
	if err != nil {
		return fmt.Errorf("resumed execution failed: %w", err)
	}

	logger.Info("Resumed execution completed")

	if result != nil {
		fmt.Println("\nResults:")
		for k, v := range result {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}
