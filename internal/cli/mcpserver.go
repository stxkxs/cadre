package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cadre-oss/cadre/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpDBPath  string
	mcpAgentID string
)

var mcpServerCmd = &cobra.Command{
	Use:    "mcp-server",
	Short:  "Run MCP IPC server (internal)",
	Long:   "Stdio MCP server for agent IPC. Invoked automatically by sprint sessions as a child process.",
	Hidden: true,
	RunE:   runMCPServer,
}

func init() {
	mcpServerCmd.Flags().StringVar(&mcpDBPath, "db", "", "path to session SQLite database (required)")
	mcpServerCmd.Flags().StringVar(&mcpAgentID, "agent", "", "agent ID for this server instance (required)")
	_ = mcpServerCmd.MarkFlagRequired("db")
	_ = mcpServerCmd.MarkFlagRequired("agent")
}

func runMCPServer(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	store, err := mcp.NewStore(mcpDBPath)
	if err != nil {
		return fmt.Errorf("open IPC store: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Ensure this agent is registered (may already exist from session launcher)
	if err := store.RegisterAgent(mcpAgentID, mcpAgentID, os.Getpid(), ""); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}
	if err := store.UpdateAgentStatus(mcpAgentID, "running", ""); err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}

	server := mcp.NewServer(store, mcpAgentID)
	return server.Run(ctx)
}
