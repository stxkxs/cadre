package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cadre-oss/cadre/internal/provider"
	"github.com/cadre-oss/cadre/internal/testutil"
)

func TestStreamExecute_TextOnly(t *testing.T) {
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{Content: "Hello, world!", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	result, err := runtime.StreamExecute(context.Background(), "say hello", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", result)
	}

	// MockProvider streams character-by-character
	if len(chunks) != len("Hello, world!") {
		t.Errorf("expected %d chunks, got %d", len("Hello, world!"), len(chunks))
	}

	if mock.CallCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.CallCount())
	}

	// Verify memory: user + assistant messages
	msgs := runtime.GetAgent().GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages in memory, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "say hello" {
		t.Errorf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "Hello, world!" {
		t.Errorf("unexpected second message: %+v", msgs[1])
	}
}

func TestStreamExecute_ToolCallLoop(t *testing.T) {
	toolInput := json.RawMessage(`{"query":"test"}`)
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{
				Content:    "Let me search.",
				StopReason: "tool_use",
				ToolCalls: []provider.ToolCall{
					{ID: "toolu_01", Name: "grep", Input: `{"query":"test"}`},
				},
				ContentBlocks: []provider.ContentBlock{
					provider.TextBlock("Let me search."),
					provider.ToolUseBlock("toolu_01", "grep", toolInput),
				},
			},
			{Content: "Found results.", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	result, err := runtime.StreamExecute(context.Background(), "search for test", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Found results." {
		t.Errorf("expected 'Found results.', got %q", result)
	}

	// Should have called provider twice
	if mock.CallCount() != 2 {
		t.Errorf("expected 2 calls, got %d", mock.CallCount())
	}

	// Verify memory: user, assistant(tool_use), user(tool_result), assistant(final)
	msgs := runtime.GetAgent().GetMessages()
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// Assistant message should have ContentBlocks with tool_use
	if len(msgs[1].ContentBlocks) != 2 {
		t.Fatalf("expected 2 content blocks on assistant, got %d", len(msgs[1].ContentBlocks))
	}
	if msgs[1].ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected tool_use block, got %q", msgs[1].ContentBlocks[1].Type)
	}

	// User message should have tool_result content blocks
	if len(msgs[2].ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block on tool result, got %d", len(msgs[2].ContentBlocks))
	}
	if msgs[2].ContentBlocks[0].Type != "tool_result" {
		t.Errorf("expected tool_result block, got %q", msgs[2].ContentBlocks[0].Type)
	}

	// Second provider call should have proper messages with content blocks
	secondReq := mock.Calls[1]
	if len(secondReq.Messages) < 3 {
		t.Fatalf("expected at least 3 messages in second request, got %d", len(secondReq.Messages))
	}
	if len(secondReq.Messages[1].ContentBlocks) == 0 {
		t.Error("expected assistant message in provider request to have ContentBlocks")
	}
	if len(secondReq.Messages[2].ContentBlocks) == 0 {
		t.Error("expected tool result message in provider request to have ContentBlocks")
	}
}

func TestStreamExecute_ToolError(t *testing.T) {
	toolInput := json.RawMessage(`{"cmd":"bad"}`)
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{
				Content:    "",
				StopReason: "tool_use",
				ToolCalls: []provider.ToolCall{
					{ID: "toolu_err", Name: "unknown_tool", Input: `{"cmd":"bad"}`},
				},
				ContentBlocks: []provider.ContentBlock{
					provider.ToolUseBlock("toolu_err", "unknown_tool", toolInput),
				},
			},
			{Content: "The tool failed.", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := runtime.StreamExecute(context.Background(), "run bad command", func(chunk string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "The tool failed." {
		t.Errorf("expected 'The tool failed.', got %q", result)
	}

	// The tool result should have is_error=true (unknown tool)
	msgs := runtime.GetAgent().GetMessages()
	toolResultMsg := msgs[2]
	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(toolResultMsg.ContentBlocks))
	}
	if !toolResultMsg.ContentBlocks[0].IsError {
		t.Error("expected is_error=true on tool result for unknown tool")
	}
}

func TestStreamExecute_MaxIterations(t *testing.T) {
	// Provider always returns tool calls
	toolCallResp := &provider.Response{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []provider.ToolCall{
			{ID: "call-1", Name: "unknown_tool", Input: `{}`},
		},
		ContentBlocks: []provider.ContentBlock{
			provider.ToolUseBlock("call-1", "unknown_tool", json.RawMessage(`{}`)),
		},
	}

	responses := make([]*provider.Response, 15)
	for i := range responses {
		responses[i] = toolCallResp
	}

	mock := &testutil.MockProvider{Responses: responses}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runtime.SetMaxIterations(3)

	_, err = runtime.StreamExecute(context.Background(), "infinite tools", func(chunk string) {})
	if err == nil {
		t.Fatal("expected error for max iterations")
	}
}
