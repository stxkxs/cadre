//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/provider"
	"github.com/stxkxs/cadre/internal/testutil"
)

func TestStreamExecute_WithToolCalls_Integration(t *testing.T) {
	toolInput := json.RawMessage(`{"query":"integration"}`)
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{
				Content:    "Searching...",
				StopReason: "tool_use",
				ToolCalls: []provider.ToolCall{
					{ID: "toolu_int", Name: "grep", Input: `{"query":"integration"}`},
				},
				ContentBlocks: []provider.ContentBlock{
					provider.TextBlock("Searching..."),
					provider.ToolUseBlock("toolu_int", "grep", toolInput),
				},
			},
			{Content: "Found 3 matches.", StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	agentCfg := testutil.TestAgentConfig("stream-agent")
	logger := testutil.TestLogger()

	runtime, err := agent.NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}

	var chunks []string
	result, err := runtime.StreamExecute(context.Background(), "search for integration tests", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify final response
	if result != "Found 3 matches." {
		t.Errorf("expected 'Found 3 matches.', got %q", result)
	}

	// Verify chunks arrived (streaming happened)
	if len(chunks) == 0 {
		t.Fatal("expected streaming chunks, got none")
	}

	// Verify tool loop completed (2 provider calls)
	if mock.CallCount() != 2 {
		t.Errorf("expected 2 provider calls, got %d", mock.CallCount())
	}

	// Verify message memory state: user, assistant(tool_use), user(tool_result), assistant(final)
	msgs := runtime.GetAgent().GetMessages()
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages in memory, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first message role 'user', got %q", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second message role 'assistant', got %q", msgs[1].Role)
	}
	if len(msgs[1].ContentBlocks) < 2 {
		t.Fatalf("expected at least 2 content blocks on assistant message, got %d", len(msgs[1].ContentBlocks))
	}
	if msgs[1].ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected tool_use block, got %q", msgs[1].ContentBlocks[1].Type)
	}
	if msgs[2].Role != "user" {
		t.Errorf("expected third message role 'user', got %q", msgs[2].Role)
	}
	if len(msgs[2].ContentBlocks) != 1 || msgs[2].ContentBlocks[0].Type != "tool_result" {
		t.Error("expected tool_result content block on third message")
	}
	if msgs[3].Role != "assistant" {
		t.Errorf("expected fourth message role 'assistant', got %q", msgs[3].Role)
	}
	if msgs[3].Content != "Found 3 matches." {
		t.Errorf("expected final assistant content 'Found 3 matches.', got %q", msgs[3].Content)
	}
}
