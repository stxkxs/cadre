package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BashTool executes shell commands
type BashTool struct {
	timeout     time.Duration
	workingDir  string
	blockedCmds []string
}

// NewBashTool creates a new bash tool
func NewBashTool() *BashTool {
	return &BashTool{
		timeout: 2 * time.Minute,
		blockedCmds: []string{
			"rm -rf /",
			"rm -rf /*",
			"mkfs",
			"dd if=/dev/zero",
			":(){:|:&};:",
		},
	}
}

// Name returns the tool name
func (t *BashTool) Name() string {
	return "bash"
}

// Description returns the tool description
func (t *BashTool) Description() string {
	return "Execute a shell command and return its output. Use for running commands, scripts, and system operations."
}

// Parameters returns the JSON schema for the tool parameters
func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": "The shell command to execute",
		},
		"working_dir": map[string]interface{}{
			"type":        "string",
			"description": "Optional: Working directory for the command",
		},
		"timeout": map[string]interface{}{
			"type":        "integer",
			"description": "Optional: Timeout in seconds (default: 120)",
		},
	}
}

// BashArgs represents the arguments for bash
type BashArgs struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
}

// Execute runs a shell command
func (t *BashTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args BashArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Security check
	for _, blocked := range t.blockedCmds {
		if strings.Contains(args.Command, blocked) {
			return "", fmt.Errorf("command blocked for security reasons")
		}
	}

	// Set timeout
	timeout := t.timeout
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)

	// Set working directory
	if args.WorkingDir != "" {
		cmd.Dir = args.WorkingDir
	} else if t.workingDir != "" {
		cmd.Dir = t.workingDir
	}

	// Set up environment
	cmd.Env = os.Environ()

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Build result
	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("STDERR:\n")
		result.WriteString(stderr.String())
	}

	// Handle errors
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result.String(), fmt.Errorf("command timed out after %v", timeout)
		}
		// Include exit code in result
		result.WriteString(fmt.Sprintf("\nExit error: %v", err))
	}

	return result.String(), nil
}

// Test verifies the tool works
func (t *BashTool) Test(ctx context.Context) (string, error) {
	result, err := t.Execute(ctx, []byte(`{"command": "echo 'bash tool test'"}`))
	if err != nil {
		return "", err
	}
	if !strings.Contains(result, "bash tool test") {
		return "", fmt.Errorf("unexpected result: %s", result)
	}
	return "bash tool operational", nil
}

// SetWorkingDir sets the default working directory
func (t *BashTool) SetWorkingDir(dir string) {
	t.workingDir = dir
}

// SetTimeout sets the default timeout
func (t *BashTool) SetTimeout(d time.Duration) {
	t.timeout = d
}
