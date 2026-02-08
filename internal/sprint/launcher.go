package sprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AgentProcess tracks a running Claude Code instance.
type AgentProcess struct {
	Cmd       *exec.Cmd
	SessionID string
	AgentID   string
	PID       int
	Done      chan struct{}
}

// mcpConfig is the JSON structure Claude Code expects for --mcp-config.
type mcpConfig struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpServerEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// LaunchAgent spawns a Claude Code instance with an MCP IPC server sidecar.
// The returned AgentProcess's Done channel is closed when the process exits.
func LaunchAgent(ctx context.Context, ws *WorkstreamConfig, sprintName, dbPath, crewBinary, sessionID string) (*AgentProcess, error) {
	agentID := ws.Name

	// Generate temporary MCP config JSON
	cfg := mcpConfig{
		MCPServers: map[string]mcpServerEntry{
			"cadre-ipc": {
				Command: crewBinary,
				Args:    []string{"mcp-server", "--db", dbPath, "--agent", agentID},
			},
		},
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("cadre-mcp-%s-*.json", agentID))
	if err != nil {
		return nil, fmt.Errorf("create mcp config: %w", err)
	}

	if err := json.NewEncoder(tmpFile).Encode(cfg); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("write mcp config: %w", err)
	}
	_ = tmpFile.Close()

	// Build system prompt context
	systemPrompt := buildAgentPrompt(ws, sprintName)

	// Resolve working directory
	workDir := ws.WorkingDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Build claude command
	args := []string{
		"--print",
		"--resume", sessionID,
		"--mcp-config", tmpFile.Name(),
		"--dangerously-skip-permissions",
		"--append-system-prompt", systemPrompt,
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Build task prompt for stdin
	taskPrompt := buildTaskPrompt(ws)
	cmd.Stdin = strings.NewReader(taskPrompt)

	if err := cmd.Start(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("start claude: %w", err)
	}

	ap := &AgentProcess{
		Cmd:       cmd,
		SessionID: sessionID,
		AgentID:   agentID,
		PID:       cmd.Process.Pid,
		Done:      make(chan struct{}),
	}

	// Monitor process exit in background
	go func() {
		defer close(ap.Done)
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		_ = cmd.Wait()
	}()

	return ap, nil
}

func buildAgentPrompt(ws *WorkstreamConfig, sprintName string) string {
	prompt := fmt.Sprintf(`You are agent "%s" in sprint session "%s".

Your role is to complete the assigned issues on branch "%s".
You have access to IPC tools (ipc_send_message, ipc_check_messages, etc.) to communicate with other agents.

When you finish your work:
1. Commit all changes to your branch
2. Call ipc_mark_complete with a summary of what you accomplished
`, ws.Name, sprintName, ws.Branch)
	return prompt
}

func buildTaskPrompt(ws *WorkstreamConfig) string {
	prompt := fmt.Sprintf("Work on the following issues: %v\n", ws.Issues)
	if ws.Branch != "" {
		prompt += fmt.Sprintf("Create or check out branch: %s\n", ws.Branch)
	}
	return prompt
}

// CadreBinaryPath returns the absolute path to the current cadre binary.
func CadreBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}

