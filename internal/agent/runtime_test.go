package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/provider"
	"github.com/stxkxs/cadre/internal/testutil"
)

func TestRuntime_Execute_DirectResponse(t *testing.T) {
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

	result, err := runtime.Execute(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", result)
	}

	if mock.CallCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.CallCount())
	}
}

func TestRuntime_Execute_ToolCallLoop(t *testing.T) {
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{
				Content:    "",
				StopReason: "tool_use",
				ToolCalls: []provider.ToolCall{
					{ID: "call-1", Name: "unknown_tool", Input: `{"input": "test"}`},
				},
			},
			{Content: "Done with tools", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := runtime.Execute(context.Background(), "use a tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Done with tools" {
		t.Errorf("expected 'Done with tools', got '%s'", result)
	}

	// Should have called provider twice: once for tool call, once for final response
	if mock.CallCount() != 2 {
		t.Errorf("expected 2 calls, got %d", mock.CallCount())
	}
}

func TestRuntime_Execute_MaxIterations(t *testing.T) {
	// Provider always returns tool calls, causing max iterations
	toolCallResp := &provider.Response{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []provider.ToolCall{
			{ID: "call-1", Name: "unknown_tool", Input: `{}`},
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

	_, err = runtime.Execute(context.Background(), "infinite tools")
	if err == nil {
		t.Fatal("expected error for max iterations")
	}
}

func TestRuntime_Execute_ContextCancellation(t *testing.T) {
	mock := &testutil.MockProvider{
		Delay: 5 * time.Second,
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = runtime.Execute(ctx, "slow prompt")
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestRuntime_Execute_ProviderError(t *testing.T) {
	mock := &testutil.MockProvider{ShouldFail: true}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = runtime.Execute(context.Background(), "fail please")
	if err == nil {
		t.Fatal("expected error from provider")
	}
}

func TestNewRuntimeWithProvider(t *testing.T) {
	mock := &testutil.MockProvider{}
	cfg := testutil.TestConfig()
	agentCfg := &config.AgentConfig{
		Name:  "test",
		Role:  "tester",
		Goal:  "test things",
		Tools: []string{"nonexistent_tool"},
		Memory: config.MemoryConfig{
			Type:      "conversation",
			MaxTokens: 10000,
		},
	}
	logger := testutil.TestLogger()

	runtime, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runtime.GetAgent().Name() != "test" {
		t.Errorf("expected agent name 'test', got '%s'", runtime.GetAgent().Name())
	}

	// Nonexistent tool should be skipped (warned), not cause error
	if len(runtime.tools) != 0 {
		t.Errorf("expected 0 loaded tools (nonexistent skipped), got %d", len(runtime.tools))
	}
}

func TestRuntime_SetMaxIterations(t *testing.T) {
	mock := &testutil.MockProvider{}
	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, _ := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	runtime.SetMaxIterations(5)

	if runtime.maxIterations != 5 {
		t.Errorf("expected maxIterations 5, got %d", runtime.maxIterations)
	}
}

func TestRuntime_GetMetrics(t *testing.T) {
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{Content: "ok", StopReason: "end_turn"},
		},
	}
	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	runtime, _ := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	runtime.Execute(context.Background(), "test")

	metrics := runtime.GetMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	summary := metrics.GetSummary()
	if summary["api_requests"].(int64) != 1 {
		t.Errorf("expected 1 API request, got %v", summary["api_requests"])
	}
}

func TestRuntime_Execute_ContentBlocks_ToolCallFlow(t *testing.T) {
	// Simulate a tool call flow with proper ContentBlocks
	toolInput := json.RawMessage(`{"query":"test"}`)
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{
				Content:    "Let me search for that.",
				StopReason: "tool_use",
				ToolCalls: []provider.ToolCall{
					{ID: "toolu_01", Name: "grep", Input: `{"query":"test"}`},
				},
				ContentBlocks: []provider.ContentBlock{
					provider.TextBlock("Let me search for that."),
					provider.ToolUseBlock("toolu_01", "grep", toolInput),
				},
			},
			{Content: "Found results.", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("test-agent")
	logger := testutil.TestLogger()

	rt, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := rt.Execute(context.Background(), "search for test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Found results." {
		t.Errorf("expected 'Found results.', got %q", result)
	}

	// Verify memory has proper content blocks
	msgs := rt.GetAgent().GetMessages()
	// Messages: user("search for test"), assistant(tool_use blocks), user(tool_result blocks), assistant("Found results.")
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// Assistant message should have ContentBlocks with tool_use
	assistantMsg := msgs[1]
	if len(assistantMsg.ContentBlocks) != 2 {
		t.Fatalf("expected 2 content blocks on assistant message, got %d", len(assistantMsg.ContentBlocks))
	}
	if assistantMsg.ContentBlocks[0].Type != "text" {
		t.Errorf("expected first block type 'text', got %q", assistantMsg.ContentBlocks[0].Type)
	}
	if assistantMsg.ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected second block type 'tool_use', got %q", assistantMsg.ContentBlocks[1].Type)
	}
	if assistantMsg.ContentBlocks[1].ID != "toolu_01" {
		t.Errorf("expected tool_use ID 'toolu_01', got %q", assistantMsg.ContentBlocks[1].ID)
	}

	// User message with tool results should have tool_result ContentBlocks
	toolResultMsg := msgs[2]
	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block on tool result message, got %d", len(toolResultMsg.ContentBlocks))
	}
	if toolResultMsg.ContentBlocks[0].Type != "tool_result" {
		t.Errorf("expected block type 'tool_result', got %q", toolResultMsg.ContentBlocks[0].Type)
	}
	if toolResultMsg.ContentBlocks[0].ToolUseID != "toolu_01" {
		t.Errorf("expected tool_use_id 'toolu_01', got %q", toolResultMsg.ContentBlocks[0].ToolUseID)
	}

	// Second provider call should have ContentBlocks in messages
	if mock.CallCount() != 2 {
		t.Fatalf("expected 2 provider calls, got %d", mock.CallCount())
	}
	secondReq := mock.Calls[1]
	// The messages sent to provider should include content blocks
	// Message at index 1 (assistant) should have ContentBlocks
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

func TestRuntime_Execute_ContentBlocks_ToolError(t *testing.T) {
	// Tool call that results in an error
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

	rt, err := NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := rt.Execute(context.Background(), "run bad command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "The tool failed." {
		t.Errorf("expected 'The tool failed.', got %q", result)
	}

	// The tool result should have is_error=true (unknown tool)
	msgs := rt.GetAgent().GetMessages()
	toolResultMsg := msgs[2]
	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(toolResultMsg.ContentBlocks))
	}
	if !toolResultMsg.ContentBlocks[0].IsError {
		t.Error("expected is_error=true on tool result for unknown tool")
	}
}
