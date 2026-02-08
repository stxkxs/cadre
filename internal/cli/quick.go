package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/quick"
	"github.com/cadre-oss/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	quickOutputJSON bool
	quickDryRun     bool
)

var quickCmd = &cobra.Command{
	Use:   "quick <action> [target]",
	Short: "Quick single-agent execution for common tasks",
	Long: `Quick mode provides fast, streamlined execution for common tasks.
Uses single-agent execution with minimal overhead.

Actions:
  fix <issue>     - Quickly fix a bug or issue
  review <path>   - Review code at path
  test <path>     - Write tests for code at path
  doc <path>      - Generate documentation for code at path
  explain <path>  - Explain code at path

Examples:
  cadre quick fix "user auth returns 500 error"
  cadre quick review ./internal/api/
  cadre quick test ./pkg/controller/
  cadre quick doc ./internal/webhook/
  cadre quick explain ./cmd/main.go

Quick mode optimizations:
  - Single agent execution (no multi-agent orchestration)
  - Compact prompts (reduced token usage)
  - Reduced context limits (50K vs 100K)
  - No intermediate checkpoints
  - Direct output format`,
	Args: cobra.MinimumNArgs(1),
	RunE: runQuick,
}

// Quick action subcommands
var quickFixCmd = &cobra.Command{
	Use:   "fix <issue-description>",
	Short: "Quickly fix a bug or issue",
	Long: `Analyze and fix a reported issue with minimal overhead.

Uses the quick-fix workflow: analyze (3m) -> fix (5m) -> verify (5m)

Example:
  cadre quick fix "login page returns 500 when password is empty"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickAction("fix", args[0], "")
	},
}

var quickReviewCmd = &cobra.Command{
	Use:   "review <path>",
	Short: "Review code at the specified path",
	Long: `Perform a quick code review focusing on quality, security, and performance.

Example:
  cadre quick review ./internal/api/
  cadre quick review ./pkg/auth/handler.go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickAction("review", "", args[0])
	},
}

var quickTestCmd = &cobra.Command{
	Use:   "test <path>",
	Short: "Write tests for code at the specified path",
	Long: `Quickly write tests for the specified code.

Example:
  cadre quick test ./pkg/controller/
  cadre quick test ./internal/service/user.go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickAction("test", "", args[0])
	},
}

var quickDocCmd = &cobra.Command{
	Use:   "doc <path>",
	Short: "Generate documentation for code at the specified path",
	Long: `Quickly generate documentation for the specified code.

Example:
  cadre quick doc ./internal/webhook/
  cadre quick doc ./pkg/api/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickAction("doc", "", args[0])
	},
}

var quickExplainCmd = &cobra.Command{
	Use:   "explain <path>",
	Short: "Explain what the code does",
	Long: `Quickly explain the specified code.

Example:
  cadre quick explain ./cmd/main.go
  cadre quick explain ./internal/reconciler/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickAction("explain", "", args[0])
	},
}

func init() {
	quickCmd.PersistentFlags().BoolVar(&quickOutputJSON, "json", false, "output as JSON")
	quickCmd.PersistentFlags().BoolVar(&quickDryRun, "dry-run", false, "show what would be executed")

	quickCmd.AddCommand(quickFixCmd)
	quickCmd.AddCommand(quickReviewCmd)
	quickCmd.AddCommand(quickTestCmd)
	quickCmd.AddCommand(quickDocCmd)
	quickCmd.AddCommand(quickExplainCmd)
}

func runQuick(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("action required: fix, review, test, doc, or explain")
	}

	action := strings.ToLower(args[0])
	var issue, target string

	switch action {
	case "fix":
		if len(args) < 2 {
			return fmt.Errorf("issue description required for fix action")
		}
		issue = strings.Join(args[1:], " ")
	case "review", "test", "doc", "explain":
		if len(args) < 2 {
			return fmt.Errorf("path required for %s action", action)
		}
		target = args[1]
	default:
		return fmt.Errorf("unknown action: %s (use fix, review, test, doc, or explain)", action)
	}

	return runQuickAction(action, issue, target)
}

func runQuickAction(action, issue, target string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, stopping...")
		cancel()
	}()

	// Initialize logger
	logger := telemetry.NewLogger(verbose)

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve target path if provided
	if target != "" {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", target, err)
		}
		target = absTarget
	}

	// Create quick executor
	executor, err := quick.NewExecutor(cfg, logger, quick.ExecutorOptions{
		OutputJSON: quickOutputJSON,
		DryRun:     quickDryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to create quick executor: %w", err)
	}

	// Build inputs based on action
	inputs := make(map[string]interface{})
	inputs["repo_path"] = "."

	switch action {
	case "fix":
		inputs["issue"] = issue
		return executor.RunQuickFix(ctx, inputs)
	case "review":
		inputs["target"] = target
		return executor.RunQuickReview(ctx, inputs)
	case "test":
		inputs["target"] = target
		return executor.RunQuickTest(ctx, inputs)
	case "doc":
		inputs["target"] = target
		return executor.RunQuickDoc(ctx, inputs)
	case "explain":
		inputs["target"] = target
		return executor.RunQuickExplain(ctx, inputs)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
