package tool

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

// ExecTool wraps a shell command as a tool.
// Template variables in the command are replaced with input arguments.
type ExecTool struct {
	name        string
	description string
	command     string
	timeout     time.Duration
	workingDir  string
}

// NewExecTool creates a new exec tool from a config.
func NewExecTool(name, description, command string) *ExecTool {
	return &ExecTool{
		name:        name,
		description: description,
		command:     command,
		timeout:     2 * time.Minute,
	}
}

func (t *ExecTool) Name() string        { return t.name }
func (t *ExecTool) Description() string  { return t.description }

func (t *ExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"input": map[string]interface{}{
			"type":        "string",
			"description": "Input to the command (substituted for {{input}} in the command template)",
		},
	}
}

type execArgs struct {
	Input string `json:"input"`
}

func (t *ExecTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args execArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Substitute template variable
	cmd := strings.ReplaceAll(t.command, "{{input}}", args.Input)

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	command := exec.CommandContext(ctx, "sh", "-c", cmd)
	if t.workingDir != "" {
		command.Dir = t.workingDir
	}
	command.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

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

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result.String(), fmt.Errorf("command timed out after %v", t.timeout)
		}
		result.WriteString(fmt.Sprintf("\nExit error: %v", err))
	}

	return result.String(), nil
}

func (t *ExecTool) Test(ctx context.Context) (string, error) {
	_, err := t.Execute(ctx, []byte(`{"input":"test"}`))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("exec tool %q operational", t.name), nil
}

// SetWorkingDir sets the working directory for the command.
func (t *ExecTool) SetWorkingDir(dir string) {
	t.workingDir = dir
}

// SetTimeout sets the execution timeout.
func (t *ExecTool) SetTimeout(d time.Duration) {
	t.timeout = d
}
