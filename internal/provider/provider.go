package provider

import (
	"context"
	"encoding/json"
)

// ContentBlock represents a content block in the Anthropic Messages API.
// Messages can contain arrays of content blocks for tool_use / tool_result flows.
type ContentBlock struct {
	Type      string          `json:"type"`                  // "text", "tool_use", "tool_result"
	Text      string          `json:"text,omitempty"`        // type="text"
	ID        string          `json:"id,omitempty"`          // type="tool_use"
	Name      string          `json:"name,omitempty"`        // type="tool_use"
	Input     json.RawMessage `json:"input,omitempty"`       // type="tool_use"
	ToolUseID string          `json:"tool_use_id,omitempty"` // type="tool_result"
	Content   string          `json:"content,omitempty"`     // type="tool_result"
	IsError   bool            `json:"is_error,omitempty"`    // type="tool_result"
}

// TextBlock creates a text content block.
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// ToolUseBlock creates a tool_use content block.
func ToolUseBlock(id, name string, input json.RawMessage) ContentBlock {
	return ContentBlock{Type: "tool_use", ID: id, Name: name, Input: input}
}

// ToolResultBlock creates a tool_result content block from a ToolResult.
func ToolResultBlock(tr ToolResult) ContentBlock {
	block := ContentBlock{
		Type:      "tool_result",
		ToolUseID: tr.ID,
	}
	if tr.Error != "" {
		block.IsError = true
		block.Content = tr.Error
	} else {
		block.Content = tr.Result
	}
	return block
}

// Message represents a conversation message
type Message struct {
	Role          string         `json:"role"` // user, assistant
	Content       string         `json:"content"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"` // JSON string
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID     string `json:"tool_use_id"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// Response represents a provider response
type Response struct {
	Content       string         `json:"content"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
	ToolCalls     []ToolCall     `json:"tool_calls,omitempty"`
	StopReason    string         `json:"stop_reason"`
	Usage         Usage          `json:"usage"`
}

// Usage tracks token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Tool represents a tool definition for the provider
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Complete sends a completion request
	Complete(ctx context.Context, req *CompletionRequest) (*Response, error)

	// Stream sends a streaming completion request
	Stream(ctx context.Context, req *CompletionRequest, handler StreamHandler) error
}

// CompletionRequest represents a completion request
type CompletionRequest struct {
	Model       string        `json:"model"`
	System      string        `json:"system"`
	Messages    []Message     `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	StopSeqs    []string      `json:"stop_sequences,omitempty"`
}

// StreamHandler handles streaming events
type StreamHandler func(event StreamEvent)

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type          string         `json:"type"`
	Content       string         `json:"content,omitempty"`
	Done          bool           `json:"done"`
	ToolCalls     []ToolCall     `json:"tool_calls,omitempty"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
	StopReason    string         `json:"stop_reason,omitempty"`
	Usage         Usage          `json:"usage,omitempty"`
}
