package agent

import (
	"context"
	"fmt"

	cadreErrors "github.com/stxkxs/cadre/internal/errors"
	"github.com/stxkxs/cadre/internal/provider"
)

// StreamCallback receives streaming content chunks.
type StreamCallback func(chunk string)

// StreamExecute runs the agent with streaming output.
// It calls callback with each text chunk as it arrives, and handles
// tool call loops the same way Execute does.
func (r *Runtime) StreamExecute(ctx context.Context, prompt string, callback StreamCallback) (string, error) {
	r.logger.Debug("Starting streaming execution", "agent", r.agent.Name())

	r.agent.AddMessage("user", prompt)

	toolDefs := r.buildToolDefinitions()

	for i := 0; i < r.maxIterations; i++ {
		r.logger.Debug("Stream iteration", "iteration", i+1)

		req := &provider.CompletionRequest{
			System:    r.agent.SystemPrompt(),
			Messages:  r.convertMessages(r.agent.GetMessages()),
			Tools:     toolDefs,
			MaxTokens: 4096,
		}

		r.metrics.IncAPIRequests()

		var fullContent string
		var finalEvent provider.StreamEvent
		handler := func(ev provider.StreamEvent) {
			if ev.Content != "" {
				fullContent += ev.Content
				callback(ev.Content)
			}
			if ev.Done {
				finalEvent = ev
			}
		}

		if err := r.provider.Stream(ctx, req, handler); err != nil {
			return "", fmt.Errorf("streaming error: %w", err)
		}

		// Handle tool calls
		if len(finalEvent.ToolCalls) > 0 {
			// Use content blocks from the stream if available, otherwise build from text.
			blocks := finalEvent.ContentBlocks
			if len(blocks) == 0 && fullContent != "" {
				blocks = []provider.ContentBlock{provider.TextBlock(fullContent)}
			}
			r.agent.AddMessageWithBlocks("assistant", fullContent, blocks)

			// Execute tools and collect results
			toolResults := r.executeTools(ctx, finalEvent.ToolCalls)

			// Build tool_result content blocks
			var resultBlocks []provider.ContentBlock
			for _, tr := range toolResults {
				resultBlocks = append(resultBlocks, provider.ToolResultBlock(tr))
			}
			r.agent.AddMessageWithBlocks("user", "", resultBlocks)

			// Reset for next iteration
			fullContent = ""
			continue
		}

		// No tool calls — we're done
		r.agent.AddMessage("assistant", fullContent)
		return fullContent, nil
	}

	return "", cadreErrors.New(cadreErrors.CodeMaxIterations,
		fmt.Sprintf("max iterations (%d) exceeded", r.maxIterations)).
		WithSuggestion("Increase max iterations with SetMaxIterations() or simplify the task to require fewer tool calls")
}
