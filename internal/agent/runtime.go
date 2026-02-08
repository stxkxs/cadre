package agent

import (
	"context"
	"fmt"

	"github.com/cadre-oss/cadre/internal/config"
	cadreErrors "github.com/cadre-oss/cadre/internal/errors"
	"github.com/cadre-oss/cadre/internal/provider"
	"github.com/cadre-oss/cadre/internal/provider/anthropic"
	"github.com/cadre-oss/cadre/internal/provider/claudecodesub"
	"github.com/cadre-oss/cadre/internal/telemetry"
	"github.com/cadre-oss/cadre/internal/tool"
)

// Runtime executes an agent with its tools
type Runtime struct {
	agent    *Agent
	provider provider.Provider
	tools    map[string]tool.Tool
	logger   *telemetry.Logger
	metrics  *telemetry.Metrics

	maxIterations int
	memoryStore   MemoryStore // non-nil when agent uses long_term memory
}

// NewRuntime creates a new agent runtime.
// If agentCfg has a Provider override, that provider is used instead of the global config.
func NewRuntime(cfg *config.Config, agentCfg *config.AgentConfig, logger *telemetry.Logger) (*Runtime, error) {
	var p provider.Provider
	switch agentCfg.Provider {
	case "claudecode":
		model := agentCfg.ProviderModel
		if model == "" {
			model = cfg.Provider.Model
		}
		workDir := agentCfg.WorkDir
		if workDir == "" {
			workDir = cfg.Provider.WorkDir
		}
		p = claudecodesub.New(model, workDir)
	default: // "anthropic" or empty — use Anthropic API
		apiKey := agentCfg.APIKey
		if apiKey == "" {
			apiKey = cfg.Provider.APIKey
		}
		model := agentCfg.ProviderModel
		if model == "" {
			model = cfg.Provider.Model
		}
		p = anthropic.NewClient(apiKey, model)
	}
	rp := provider.NewRetryProvider(p, provider.DefaultRetryConfig())
	return NewRuntimeWithProvider(cfg, agentCfg, rp, logger)
}

// NewRuntimeWithProvider creates a new agent runtime with an injected provider.
// This enables testing with mock providers.
func NewRuntimeWithProvider(cfg *config.Config, agentCfg *config.AgentConfig, p provider.Provider, logger *telemetry.Logger) (*Runtime, error) {
	agent := NewAgent(agentCfg)

	tools := make(map[string]tool.Tool)
	for _, toolName := range agentCfg.Tools {
		t, err := tool.Get(toolName)
		if err != nil {
			logger.Warn("Tool not found", "tool", toolName, "error", err)
			continue
		}
		tools[toolName] = t
	}

	r := &Runtime{
		agent:         agent,
		provider:      p,
		tools:         tools,
		logger:        logger,
		metrics:       telemetry.NewMetrics(),
		maxIterations: 10,
	}

	// Open persistent memory store for long_term memory agents.
	if agentCfg.Memory.Type == "long_term" {
		dbPath := ".cadre/memory.db"
		if cfg.State.Path != "" {
			// Co-locate memory DB next to state DB.
			dbPath = cfg.State.Path + ".memory"
		}
		store, err := NewSQLiteMemoryStore(dbPath)
		if err != nil {
			logger.Warn("Failed to open memory store, falling back to in-memory", "error", err)
		} else {
			r.memoryStore = store
			if err := agent.Memory().SetStore(store, agentCfg.Name); err != nil {
				logger.Warn("Failed to bootstrap memory from store", "error", err)
			}
		}
	}

	return r, nil
}

// Execute runs the agent with a prompt and returns the response
func (r *Runtime) Execute(ctx context.Context, prompt string) (string, error) {
	r.logger.Debug("Starting agent execution", "agent", r.agent.Name())

	// Add user message to memory
	r.agent.AddMessage("user", prompt)

	// Build tool definitions
	toolDefs := r.buildToolDefinitions()

	// Execute loop (handle tool calls)
	for i := 0; i < r.maxIterations; i++ {
		r.logger.Debug("Agent iteration", "iteration", i+1)

		// Build request
		req := &provider.CompletionRequest{
			System:    r.agent.SystemPrompt(),
			Messages:  r.convertMessages(r.agent.GetMessages()),
			Tools:     toolDefs,
			MaxTokens: 4096,
		}

		r.metrics.IncAPIRequests()

		// Call provider
		resp, err := r.provider.Complete(ctx, req)
		if err != nil {
			return "", fmt.Errorf("provider error: %w", err)
		}

		r.logger.Debug("Provider response",
			"stop_reason", resp.StopReason,
			"tool_calls", len(resp.ToolCalls),
			"input_tokens", resp.Usage.InputTokens,
			"output_tokens", resp.Usage.OutputTokens,
		)

		// Handle tool calls
		if len(resp.ToolCalls) > 0 {
			// Add assistant message preserving tool_use content blocks
			r.agent.AddMessageWithBlocks("assistant", resp.Content, resp.ContentBlocks)

			// Execute tools and collect results
			toolResults := r.executeTools(ctx, resp.ToolCalls)

			// Build tool_result content blocks
			var resultBlocks []provider.ContentBlock
			for _, tr := range toolResults {
				resultBlocks = append(resultBlocks, provider.ToolResultBlock(tr))
			}
			r.agent.AddMessageWithBlocks("user", "", resultBlocks)

			// Continue loop
			continue
		}

		// No tool calls - we're done
		r.agent.AddMessage("assistant", resp.Content)
		return resp.Content, nil
	}

	return "", cadreErrors.New(cadreErrors.CodeMaxIterations,
		fmt.Sprintf("max iterations (%d) exceeded", r.maxIterations)).
		WithSuggestion("Increase max iterations with SetMaxIterations() or simplify the task to require fewer tool calls")
}

// ExecuteWithToolResults continues execution after tool results.
// It delegates to executeWithToolResultsDepth with depth=0.
func (r *Runtime) ExecuteWithToolResults(ctx context.Context, toolResults []provider.ToolResult) (string, error) {
	return r.executeWithToolResultsDepth(ctx, toolResults, 0)
}

// executeWithToolResultsDepth is the recursive implementation with a depth guard.
func (r *Runtime) executeWithToolResultsDepth(ctx context.Context, toolResults []provider.ToolResult, depth int) (string, error) {
	if depth >= r.maxIterations {
		return "", cadreErrors.New(cadreErrors.CodeMaxIterations,
			fmt.Sprintf("ExecuteWithToolResults: max iterations (%d) exceeded", r.maxIterations)).
			WithSuggestion("Increase max iterations or simplify the task")
	}

	// Add tool results as a user message with tool_result content blocks
	var resultBlocks []provider.ContentBlock
	for _, tr := range toolResults {
		resultBlocks = append(resultBlocks, provider.ToolResultBlock(tr))
	}
	r.agent.AddMessageWithBlocks("user", "", resultBlocks)

	// Build request
	req := &provider.CompletionRequest{
		System:    r.agent.SystemPrompt(),
		Messages:  r.convertMessages(r.agent.GetMessages()),
		Tools:     r.buildToolDefinitions(),
		MaxTokens: 4096,
	}

	resp, err := r.provider.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	// Handle more tool calls if needed
	if len(resp.ToolCalls) > 0 {
		r.agent.AddMessageWithBlocks("assistant", resp.Content, resp.ContentBlocks)
		results := r.executeTools(ctx, resp.ToolCalls)
		return r.executeWithToolResultsDepth(ctx, results, depth+1)
	}

	r.agent.AddMessage("assistant", resp.Content)
	return resp.Content, nil
}

// buildToolDefinitions converts tools to provider format
func (r *Runtime) buildToolDefinitions() []provider.Tool {
	defs := make([]provider.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, provider.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": t.Parameters(),
			},
		})
	}
	return defs
}

// executeTools executes tool calls and returns results
func (r *Runtime) executeTools(ctx context.Context, calls []provider.ToolCall) []provider.ToolResult {
	results := make([]provider.ToolResult, len(calls))

	for i, call := range calls {
		r.logger.Debug("Executing tool", "tool", call.Name, "id", call.ID)
		r.metrics.IncToolCalls()

		t, ok := r.tools[call.Name]
		if !ok {
			results[i] = provider.ToolResult{
				ID:    call.ID,
				Error: fmt.Sprintf("unknown tool: %s", call.Name),
			}
			continue
		}

		result, err := t.Execute(ctx, []byte(call.Input))
		if err != nil {
			r.logger.Warn("Tool execution failed", "tool", call.Name, "error", err)
			results[i] = provider.ToolResult{
				ID:    call.ID,
				Error: err.Error(),
			}
		} else {
			r.logger.Debug("Tool execution succeeded", "tool", call.Name, "result_length", len(result))
			results[i] = provider.ToolResult{
				ID:     call.ID,
				Result: result,
			}
		}
	}

	return results
}

// convertMessages converts agent messages to provider format
func (r *Runtime) convertMessages(messages []Message) []provider.Message {
	result := make([]provider.Message, len(messages))
	for i, msg := range messages {
		result[i] = provider.Message{
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: msg.ContentBlocks,
		}
	}
	return result
}

// GetAgent returns the underlying agent
func (r *Runtime) GetAgent() *Agent {
	return r.agent
}

// GetMetrics returns the runtime metrics
func (r *Runtime) GetMetrics() *telemetry.Metrics {
	return r.metrics
}

// SetMaxIterations sets the maximum number of tool use iterations
func (r *Runtime) SetMaxIterations(n int) {
	r.maxIterations = n
}

// AddTools merges additional tools into the runtime's tool map.
// Used to inject delegation tools for hierarchical execution.
func (r *Runtime) AddTools(tools map[string]tool.Tool) {
	for name, t := range tools {
		r.tools[name] = t
	}
}

// Close releases resources held by the runtime (e.g. memory store).
func (r *Runtime) Close() error {
	if r.memoryStore != nil {
		return r.memoryStore.Close()
	}
	return nil
}

// SetMemoryStore attaches a memory store to the agent's memory.
// Used to inject shared memory from crew.
func (r *Runtime) SetMemoryStore(store MemoryStore, namespace string) error {
	return r.agent.Memory().SetStore(store, namespace)
}
