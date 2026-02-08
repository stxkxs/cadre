package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/provider"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// MockProvider implements provider.Provider for testing.
type MockProvider struct {
	mu         sync.Mutex
	Responses  []*provider.Response // queued responses, consumed in order
	Calls      []*provider.CompletionRequest
	ShouldFail bool
	FailErr    error
	Delay      time.Duration
	idx        int
}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.Response, error) {
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, req)

	if m.ShouldFail {
		if m.FailErr != nil {
			return nil, m.FailErr
		}
		return nil, fmt.Errorf("mock provider error")
	}

	if m.idx >= len(m.Responses) {
		return &provider.Response{
			Content:    "default mock response",
			StopReason: "end_turn",
		}, nil
	}

	resp := m.Responses[m.idx]
	m.idx++
	return resp, nil
}

func (m *MockProvider) Stream(ctx context.Context, req *provider.CompletionRequest, handler provider.StreamHandler) error {
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return err
	}

	// Stream text chunks character-by-character for realistic simulation,
	// then emit a terminal done event with full metadata.
	for _, ch := range resp.Content {
		handler(provider.StreamEvent{Type: "text", Content: string(ch)})
	}

	handler(provider.StreamEvent{
		Type:          "done",
		Done:          true,
		ToolCalls:     resp.ToolCalls,
		ContentBlocks: resp.ContentBlocks,
		StopReason:    resp.StopReason,
		Usage:         resp.Usage,
	})
	return nil
}

// CallCount returns the number of Complete calls made (thread-safe).
func (m *MockProvider) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// MockTool implements tool.Tool for testing.
type MockTool struct {
	Name_      string
	Desc       string
	Result     string
	ShouldFail bool
	mu         sync.Mutex
	Executions int
}

func (t *MockTool) Name() string        { return t.Name_ }
func (t *MockTool) Description() string  { return t.Desc }
func (t *MockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"input": map[string]interface{}{
			"type":        "string",
			"description": "test input",
		},
	}
}

func (t *MockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	t.mu.Lock()
	t.Executions++
	t.mu.Unlock()

	if t.ShouldFail {
		return "", fmt.Errorf("mock tool error")
	}
	return t.Result, nil
}

func (t *MockTool) Test(ctx context.Context) (string, error) {
	return "mock tool operational", nil
}

// ExecutionCount returns the number of times Execute was called (thread-safe).
func (t *MockTool) ExecutionCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Executions
}

// TestLogger returns a logger suitable for tests (verbose, no file output).
func TestLogger() *telemetry.Logger {
	return telemetry.NewLogger(true)
}

// TestConfig returns a minimal config for testing.
func TestConfig() *config.Config {
	return &config.Config{
		Name:    "test-project",
		Version: "1.0",
		Provider: config.ProviderConfig{
			Name:  "mock",
			Model: "mock-model",
		},
		Defaults: config.DefaultsConfig{
			Timeout:    "5m",
			MaxRetries: 1,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
		State: config.StateConfig{
			Driver: "memory",
		},
	}
}

// TestAgentConfig returns a minimal agent config for testing.
func TestAgentConfig(name string) *config.AgentConfig {
	return &config.AgentConfig{
		Name:      name,
		Role:      "Test Agent",
		Goal:      "Complete test tasks",
		Backstory: "A test agent.",
		Tools:     []string{},
		Memory: config.MemoryConfig{
			Type:      "conversation",
			MaxTokens: 10000,
		},
	}
}
