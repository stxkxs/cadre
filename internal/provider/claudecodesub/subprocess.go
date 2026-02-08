// Package claudecodesub implements a provider.Provider that shells out to
// the Claude Code CLI (`claude --print --output-format stream-json`).
// This lets agents use a Max/Pro subscription instead of direct API keys.
package claudecodesub

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cadre-oss/cadre/internal/provider"
)

// Provider runs the Claude Code CLI as a subprocess.
type Provider struct {
	model   string
	workDir string
}

// New creates a Claude Code subprocess provider.
func New(model, workDir string) *Provider {
	return &Provider{model: model, workDir: workDir}
}

func (p *Provider) Name() string { return "claudecode" }

// Complete runs claude --print and returns the final result.
func (p *Provider) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.Response, error) {
	args := p.buildArgs(req, false)
	stdin := formatMessages(req.Messages)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(stdin)
	if p.workDir != "" {
		cmd.Dir = p.workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude CLI error: %w (stderr: %s)", err, stderr.String())
	}

	return &provider.Response{
		Content:    strings.TrimSpace(stdout.String()),
		StopReason: "end_turn",
	}, nil
}

// Stream runs claude --print --output-format stream-json and emits StreamEvents.
func (p *Provider) Stream(ctx context.Context, req *provider.CompletionRequest, handler provider.StreamHandler) error {
	args := p.buildArgs(req, true)
	stdin := formatMessages(req.Messages)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(stdin)
	if p.workDir != "" {
		cmd.Dir = p.workDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg streamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // skip malformed lines
		}

		switch msg.Type {
		case "assistant":
			// Extract text content from the message's content blocks.
			text := extractText(msg.Message)
			if text != "" {
				// Emit incremental content (delta from what we've seen before).
				delta := text[fullContent.Len():]
				if delta != "" {
					fullContent.WriteString(delta)
					handler(provider.StreamEvent{
						Type:    "text",
						Content: delta,
					})
				}
			}

		case "result":
			// Final result — emit done event.
			result := extractResultText(msg)
			if result != "" && result != fullContent.String() {
				// Emit any remaining content not yet streamed.
				remaining := result[fullContent.Len():]
				if remaining != "" {
					handler(provider.StreamEvent{
						Type:    "text",
						Content: remaining,
					})
				}
			}
			handler(provider.StreamEvent{
				Type: "done",
				Done: true,
			})

			// Wait for process to finish.
			_ = cmd.Wait()
			return nil
		}
	}

	// Process ended without result event — wait and check error.
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude CLI error: %w (stderr: %s)", err, stderr.String())
	}

	// Emit done even if no result event arrived.
	handler(provider.StreamEvent{
		Type: "done",
		Done: true,
	})
	return nil
}

// buildArgs constructs the CLI arguments.
func (p *Provider) buildArgs(req *provider.CompletionRequest, streaming bool) []string {
	args := []string{
		"--print",
		"--verbose",
	}

	if streaming {
		args = append(args, "--output-format", "stream-json")
	}

	if req.System != "" {
		args = append(args, "--system-prompt", req.System)
	}

	model := req.Model
	if model == "" {
		model = p.model
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	return args
}

// formatMessages converts provider.Messages into a text conversation for stdin.
// Claude --print reads the user prompt from stdin.
func formatMessages(msgs []provider.Message) string {
	if len(msgs) == 0 {
		return ""
	}

	// If there's only one user message, just send it directly.
	if len(msgs) == 1 && msgs[0].Role == "user" {
		return msgs[0].Content
	}

	// For multi-turn conversations, format as a transcript.
	// The last user message is the actual prompt; prior messages provide context.
	var sb strings.Builder
	for i, msg := range msgs {
		if i == len(msgs)-1 && msg.Role == "user" {
			// Last user message — this is the actual prompt.
			// Prepend conversation context if there were prior messages.
			if i > 0 {
				sb.WriteString("\n---\n\nBased on the conversation above, respond to:\n\n")
			}
			sb.WriteString(msg.Content)
		} else {
			switch msg.Role {
			case "user":
				sb.WriteString("User: ")
			case "assistant":
				sb.WriteString("Assistant: ")
			}
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// --- JSON types for stream-json output ---

type streamMessage struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	Result  string          `json:"result,omitempty"`
}

type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func extractText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var msg assistantMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}

func extractResultText(msg streamMessage) string {
	if msg.Result != "" {
		return msg.Result
	}
	return ""
}
