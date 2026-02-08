package anthropic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cadre-oss/cadre/internal/provider"
)

func TestParseStream_TextOnly(t *testing.T) {
	// Simulate an SSE stream with a single text content block.
	sseData := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":25}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	client := &Client{}
	var textChunks []string
	var doneEvent provider.StreamEvent

	handler := func(ev provider.StreamEvent) {
		if ev.Type == "text" {
			textChunks = append(textChunks, ev.Content)
		}
		if ev.Done {
			doneEvent = ev
		}
	}

	err := client.parseStream(strings.NewReader(sseData), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have received two text chunks.
	if len(textChunks) != 2 {
		t.Fatalf("expected 2 text chunks, got %d", len(textChunks))
	}
	if textChunks[0] != "Hello" || textChunks[1] != " world" {
		t.Errorf("unexpected chunks: %v", textChunks)
	}

	// Done event should be emitted.
	if !doneEvent.Done {
		t.Fatal("expected done event")
	}
	if doneEvent.StopReason != "end_turn" {
		t.Errorf("expected stop_reason 'end_turn', got %q", doneEvent.StopReason)
	}

	// Should have one text content block.
	if len(doneEvent.ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(doneEvent.ContentBlocks))
	}
	if doneEvent.ContentBlocks[0].Type != "text" {
		t.Errorf("expected text block, got %q", doneEvent.ContentBlocks[0].Type)
	}
	if doneEvent.ContentBlocks[0].Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", doneEvent.ContentBlocks[0].Text)
	}

	// No tool calls.
	if len(doneEvent.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(doneEvent.ToolCalls))
	}

	// Usage should be populated.
	if doneEvent.Usage.InputTokens != 25 {
		t.Errorf("expected 25 input tokens, got %d", doneEvent.Usage.InputTokens)
	}
	if doneEvent.Usage.OutputTokens != 10 {
		t.Errorf("expected 10 output tokens, got %d", doneEvent.Usage.OutputTokens)
	}
}

func TestParseStream_ToolUse(t *testing.T) {
	// Simulate text + tool_use content blocks.
	sseData := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":50}}}`,
		// Text block
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me search."}}`,
		`data: {"type":"content_block_stop","index":0}`,
		// Tool use block
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_01","name":"grep"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query\":"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"test\"}"}}`,
		`data: {"type":"content_block_stop","index":1}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	client := &Client{}
	var textChunks []string
	var doneEvent provider.StreamEvent

	handler := func(ev provider.StreamEvent) {
		if ev.Type == "text" {
			textChunks = append(textChunks, ev.Content)
		}
		if ev.Done {
			doneEvent = ev
		}
	}

	err := client.parseStream(strings.NewReader(sseData), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Text deltas should stream in real-time.
	if len(textChunks) != 1 {
		t.Fatalf("expected 1 text chunk, got %d", len(textChunks))
	}
	if textChunks[0] != "Let me search." {
		t.Errorf("expected 'Let me search.', got %q", textChunks[0])
	}

	// Done event should have both content blocks.
	if len(doneEvent.ContentBlocks) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(doneEvent.ContentBlocks))
	}
	if doneEvent.ContentBlocks[0].Type != "text" {
		t.Errorf("expected first block type 'text', got %q", doneEvent.ContentBlocks[0].Type)
	}
	if doneEvent.ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected second block type 'tool_use', got %q", doneEvent.ContentBlocks[1].Type)
	}
	if doneEvent.ContentBlocks[1].ID != "toolu_01" {
		t.Errorf("expected tool ID 'toolu_01', got %q", doneEvent.ContentBlocks[1].ID)
	}
	if doneEvent.ContentBlocks[1].Name != "grep" {
		t.Errorf("expected tool name 'grep', got %q", doneEvent.ContentBlocks[1].Name)
	}

	// Tool calls should be populated.
	if len(doneEvent.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(doneEvent.ToolCalls))
	}
	if doneEvent.ToolCalls[0].ID != "toolu_01" {
		t.Errorf("expected tool call ID 'toolu_01', got %q", doneEvent.ToolCalls[0].ID)
	}
	if doneEvent.ToolCalls[0].Name != "grep" {
		t.Errorf("expected tool call name 'grep', got %q", doneEvent.ToolCalls[0].Name)
	}
	if doneEvent.ToolCalls[0].Input != `{"query":"test"}` {
		t.Errorf("expected input '{\"query\":\"test\"}', got %q", doneEvent.ToolCalls[0].Input)
	}

	if doneEvent.StopReason != "tool_use" {
		t.Errorf("expected stop_reason 'tool_use', got %q", doneEvent.StopReason)
	}
}

func TestParseStream_InputJsonDeltaAccumulation(t *testing.T) {
	// Simulate heavily fragmented input_json_delta across many events.
	sseData := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":10}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_frag","name":"bash"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"command"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\":"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"ls -la\""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"}"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	client := &Client{}
	var doneEvent provider.StreamEvent

	handler := func(ev provider.StreamEvent) {
		if ev.Done {
			doneEvent = ev
		}
	}

	err := client.parseStream(strings.NewReader(sseData), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doneEvent.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(doneEvent.ToolCalls))
	}

	// Verify the reassembled JSON is valid and correct.
	expectedInput := `{"command":"ls -la"}`
	if doneEvent.ToolCalls[0].Input != expectedInput {
		t.Errorf("expected input %q, got %q", expectedInput, doneEvent.ToolCalls[0].Input)
	}

	// Verify it's valid JSON by attempting to unmarshal.
	var parsed map[string]string
	if err := json.Unmarshal([]byte(doneEvent.ToolCalls[0].Input), &parsed); err != nil {
		t.Errorf("reassembled JSON is not valid: %v", err)
	}
	if parsed["command"] != "ls -la" {
		t.Errorf("expected command 'ls -la', got %q", parsed["command"])
	}
}

func TestParseStream_MultipleToolUseBlocks(t *testing.T) {
	// Simulate text block (index 0) + two tool_use blocks (index 1, 2).
	sseData := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":30}}}`,
		// Text block
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I'll run two tools."}}`,
		`data: {"type":"content_block_stop","index":0}`,
		// First tool_use block
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_a","name":"file_read"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"main.go\"}"}}`,
		`data: {"type":"content_block_stop","index":1}`,
		// Second tool_use block
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"toolu_b","name":"bash"}}`,
		`data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"go test\"}"}}`,
		`data: {"type":"content_block_stop","index":2}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	client := &Client{}
	var doneEvent provider.StreamEvent

	handler := func(ev provider.StreamEvent) {
		if ev.Done {
			doneEvent = ev
		}
	}

	err := client.parseStream(strings.NewReader(sseData), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 content blocks: text + 2 tool_use
	if len(doneEvent.ContentBlocks) != 3 {
		t.Fatalf("expected 3 content blocks, got %d", len(doneEvent.ContentBlocks))
	}
	if doneEvent.ContentBlocks[0].Type != "text" {
		t.Errorf("expected first block type 'text', got %q", doneEvent.ContentBlocks[0].Type)
	}
	if doneEvent.ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected second block type 'tool_use', got %q", doneEvent.ContentBlocks[1].Type)
	}
	if doneEvent.ContentBlocks[2].Type != "tool_use" {
		t.Errorf("expected third block type 'tool_use', got %q", doneEvent.ContentBlocks[2].Type)
	}

	// Should have 2 tool calls
	if len(doneEvent.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(doneEvent.ToolCalls))
	}

	// First tool call
	if doneEvent.ToolCalls[0].ID != "toolu_a" {
		t.Errorf("expected first tool ID 'toolu_a', got %q", doneEvent.ToolCalls[0].ID)
	}
	if doneEvent.ToolCalls[0].Name != "file_read" {
		t.Errorf("expected first tool name 'file_read', got %q", doneEvent.ToolCalls[0].Name)
	}
	if doneEvent.ToolCalls[0].Input != `{"path":"main.go"}` {
		t.Errorf("expected first tool input '{\"path\":\"main.go\"}', got %q", doneEvent.ToolCalls[0].Input)
	}

	// Second tool call
	if doneEvent.ToolCalls[1].ID != "toolu_b" {
		t.Errorf("expected second tool ID 'toolu_b', got %q", doneEvent.ToolCalls[1].ID)
	}
	if doneEvent.ToolCalls[1].Name != "bash" {
		t.Errorf("expected second tool name 'bash', got %q", doneEvent.ToolCalls[1].Name)
	}
	if doneEvent.ToolCalls[1].Input != `{"command":"go test"}` {
		t.Errorf("expected second tool input '{\"command\":\"go test\"}', got %q", doneEvent.ToolCalls[1].Input)
	}

	if doneEvent.StopReason != "tool_use" {
		t.Errorf("expected stop_reason 'tool_use', got %q", doneEvent.StopReason)
	}
}
