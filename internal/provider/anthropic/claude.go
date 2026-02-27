package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	cadreErrors "github.com/stxkxs/cadre/internal/errors"
	"github.com/stxkxs/cadre/internal/provider"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	defaultModel   = "claude-sonnet-4-20250514"
)

// Client implements the Anthropic provider
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Anthropic client
func NewClient(apiKey, model string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = defaultModel
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "anthropic"
}

// Complete sends a completion request to Claude
func (c *Client) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.Response, error) {
	if c.apiKey == "" {
		return nil, cadreErrors.New(cadreErrors.CodeAPIKeyMissing, "ANTHROPIC_API_KEY not set").
			WithSuggestion("Set the ANTHROPIC_API_KEY environment variable or add api_key to your cadre.yaml provider config")
	}

	// Build API request
	apiReq := c.buildRequest(req)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return c.parseResponse(respBody)
}

// Stream sends a streaming completion request
func (c *Client) Stream(ctx context.Context, req *provider.CompletionRequest, handler provider.StreamHandler) error {
	if c.apiKey == "" {
		return cadreErrors.New(cadreErrors.CodeAPIKeyMissing, "ANTHROPIC_API_KEY not set").
			WithSuggestion("Set the ANTHROPIC_API_KEY environment variable or add api_key to your cadre.yaml provider config")
	}

	// Build API request with streaming
	apiReq := c.buildRequest(req)
	apiReq["stream"] = true

	body, err := json.Marshal(apiReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream
	return c.parseStream(resp.Body, handler)
}

// buildRequest converts our request to Anthropic API format
func (c *Client) buildRequest(req *provider.CompletionRequest) map[string]interface{} {
	model := req.Model
	if model == "" {
		model = c.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	apiReq := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
	}

	if req.System != "" {
		apiReq["system"] = req.System
	}

	// Convert messages
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if len(msg.ContentBlocks) > 0 {
			// Serialize as content block array for tool_use / tool_result flows
			var blocks []map[string]interface{}
			for _, block := range msg.ContentBlocks {
				b := map[string]interface{}{"type": block.Type}
				switch block.Type {
				case "text":
					b["text"] = block.Text
				case "tool_use":
					b["id"] = block.ID
					b["name"] = block.Name
					b["input"] = block.Input
				case "tool_result":
					b["tool_use_id"] = block.ToolUseID
					b["content"] = block.Content
					if block.IsError {
						b["is_error"] = true
					}
				}
				blocks = append(blocks, b)
			}
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": blocks,
			})
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	apiReq["messages"] = messages

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			})
		}
		apiReq["tools"] = tools
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	if len(req.StopSeqs) > 0 {
		apiReq["stop_sequences"] = req.StopSeqs
	}

	return apiReq
}

// parseResponse parses the API response
func (c *Client) parseResponse(body []byte) (*provider.Response, error) {
	var apiResp struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	resp := &provider.Response{
		StopReason: apiResp.StopReason,
		Usage: provider.Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls, and preserve full content blocks
	var textContent []string
	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			textContent = append(textContent, block.Text)
			resp.ContentBlocks = append(resp.ContentBlocks, provider.TextBlock(block.Text))
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			resp.ToolCalls = append(resp.ToolCalls, provider.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: string(inputJSON),
			})
			resp.ContentBlocks = append(resp.ContentBlocks, provider.ToolUseBlock(block.ID, block.Name, inputJSON))
		}
	}

	resp.Content = joinStrings(textContent, "\n")

	return resp, nil
}

// blockAccumulator tracks state for a single content block being streamed.
type blockAccumulator struct {
	blockType string          // "text" or "tool_use"
	text      strings.Builder // text_delta accumulation
	toolID    string
	toolName  string
	inputJSON strings.Builder // input_json_delta accumulation
}

// parseStream parses an SSE stream, handling `data: ` prefixes per the SSE spec.
// It accumulates content blocks across events and emits a final Done event with
// assembled ToolCalls and ContentBlocks.
func (c *Client) parseStream(body io.Reader, handler provider.StreamHandler) error {
	scanner := newLineScanner(body)

	var blocks []blockAccumulator
	var stopReason string
	var usage provider.Usage

	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines are prefixed with "data: " — strip the prefix.
		if len(line) < 6 || line[:6] != "data: " {
			continue // skip event:, id:, empty lines, etc.
		}
		jsonData := line[6:]

		if jsonData == "[DONE]" {
			break
		}

		var event struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue // skip malformed lines
		}

		switch event.Type {
		case "message_start":
			// Extract input token usage from message_start if present.
			var msgStart struct {
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(jsonData), &msgStart); err == nil {
				usage.InputTokens = msgStart.Message.Usage.InputTokens
			}

		case "content_block_start":
			// Initialize accumulator at the given index.
			for len(blocks) <= event.Index {
				blocks = append(blocks, blockAccumulator{})
			}
			blocks[event.Index] = blockAccumulator{
				blockType: event.ContentBlock.Type,
				toolID:    event.ContentBlock.ID,
				toolName:  event.ContentBlock.Name,
			}

		case "content_block_delta":
			if event.Index >= len(blocks) {
				continue
			}
			acc := &blocks[event.Index]
			switch event.Delta.Type {
			case "text_delta":
				acc.text.WriteString(event.Delta.Text)
				// Stream text deltas in real-time.
				handler(provider.StreamEvent{
					Type:    "text",
					Content: event.Delta.Text,
				})
			case "input_json_delta":
				acc.inputJSON.WriteString(event.Delta.PartialJSON)
			}

		case "message_delta":
			if event.Delta.StopReason != "" {
				stopReason = event.Delta.StopReason
			}
			if event.Usage.OutputTokens > 0 {
				usage.OutputTokens = event.Usage.OutputTokens
			}

		case "message_stop":
			// Assemble final event from accumulated blocks.
			var contentBlocks []provider.ContentBlock
			var toolCalls []provider.ToolCall
			var textParts []string

			for _, acc := range blocks {
				switch acc.blockType {
				case "text":
					text := acc.text.String()
					textParts = append(textParts, text)
					contentBlocks = append(contentBlocks, provider.TextBlock(text))
				case "tool_use":
					inputStr := acc.inputJSON.String()
					if inputStr == "" {
						inputStr = "{}"
					}
					contentBlocks = append(contentBlocks, provider.ToolUseBlock(
						acc.toolID, acc.toolName, json.RawMessage(inputStr),
					))
					toolCalls = append(toolCalls, provider.ToolCall{
						ID:    acc.toolID,
						Name:  acc.toolName,
						Input: inputStr,
					})
				}
			}

			handler(provider.StreamEvent{
				Type:          "done",
				Done:          true,
				ToolCalls:     toolCalls,
				ContentBlocks: contentBlocks,
				StopReason:    stopReason,
				Usage:         usage,
			})
			return nil
		}
	}

	// Stream ended without message_stop (e.g. connection dropped).
	// Emit done event with whatever we accumulated.
	var contentBlocks []provider.ContentBlock
	var toolCalls []provider.ToolCall

	for _, acc := range blocks {
		switch acc.blockType {
		case "text":
			contentBlocks = append(contentBlocks, provider.TextBlock(acc.text.String()))
		case "tool_use":
			inputStr := acc.inputJSON.String()
			if inputStr == "" {
				inputStr = "{}"
			}
			contentBlocks = append(contentBlocks, provider.ToolUseBlock(
				acc.toolID, acc.toolName, json.RawMessage(inputStr),
			))
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:    acc.toolID,
				Name:  acc.toolName,
				Input: inputStr,
			})
		}
	}

	handler(provider.StreamEvent{
		Type:          "done",
		Done:          true,
		ToolCalls:     toolCalls,
		ContentBlocks: contentBlocks,
		StopReason:    stopReason,
		Usage:         usage,
	})

	return nil
}

// lineScanner wraps bufio.Scanner for line-by-line reading.
type lineScanner struct {
	buf  []byte
	pos  int
	body io.Reader
	text string
	done bool
}

func newLineScanner(r io.Reader) *lineScanner {
	return &lineScanner{
		buf:  make([]byte, 0, 4096),
		body: r,
	}
}

func (s *lineScanner) Scan() bool {
	if s.done {
		return false
	}

	for {
		// Look for newline in existing buffer
		for i := s.pos; i < len(s.buf); i++ {
			if s.buf[i] == '\n' {
				line := string(s.buf[s.pos:i])
				s.pos = i + 1
				// Trim trailing \r if present
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				s.text = line
				return true
			}
		}

		// Compact buffer
		if s.pos > 0 {
			copy(s.buf, s.buf[s.pos:])
			s.buf = s.buf[:len(s.buf)-s.pos]
			s.pos = 0
		}

		// Read more data
		tmp := make([]byte, 4096)
		n, err := s.body.Read(tmp)
		if n > 0 {
			s.buf = append(s.buf, tmp[:n]...)
		}
		if err != nil {
			s.done = true
			// Process remaining data
			if len(s.buf) > s.pos {
				s.text = string(s.buf[s.pos:])
				s.pos = len(s.buf)
				return true
			}
			return false
		}
	}
}

func (s *lineScanner) Text() string {
	return s.text
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
