package agent

import (
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/provider"
)

// Agent represents an AI agent with a role and capabilities
type Agent struct {
	config    *config.AgentConfig
	memory    *Memory
	toolNames []string
}

// NewAgent creates a new agent from configuration
func NewAgent(cfg *config.AgentConfig) *Agent {
	return &Agent{
		config:    cfg,
		memory:    NewMemory(cfg.Memory.Type, cfg.Memory.MaxTokens),
		toolNames: cfg.Tools,
	}
}

// Name returns the agent name
func (a *Agent) Name() string {
	return a.config.Name
}

// Role returns the agent role
func (a *Agent) Role() string {
	return a.config.Role
}

// Goal returns the agent goal
func (a *Agent) Goal() string {
	return a.config.Goal
}

// Backstory returns the agent backstory
func (a *Agent) Backstory() string {
	return a.config.Backstory
}

// Tools returns the list of tool names the agent can use
func (a *Agent) Tools() []string {
	return a.toolNames
}

// Memory returns the agent's memory
func (a *Agent) Memory() *Memory {
	return a.memory
}

// SystemPrompt generates the system prompt for the agent
func (a *Agent) SystemPrompt() string {
	prompt := "You are " + a.config.Name + ".\n\n"
	prompt += "Role: " + a.config.Role + "\n"
	prompt += "Goal: " + a.config.Goal + "\n\n"

	if a.config.Backstory != "" {
		prompt += "Background:\n" + a.config.Backstory + "\n\n"
	}

	if len(a.toolNames) > 0 {
		prompt += "You have access to the following tools:\n"
		for _, tool := range a.toolNames {
			prompt += "- " + tool + "\n"
		}
		prompt += "\n"
	}

	prompt += "Instructions:\n"
	prompt += "- Think step by step before taking action\n"
	prompt += "- Use tools when they would help accomplish your goal\n"
	prompt += "- Be thorough but efficient\n"
	prompt += "- Report results clearly and concisely\n"

	return prompt
}

// AddMessage adds a message to the agent's memory
func (a *Agent) AddMessage(role, content string) {
	a.memory.Add(Message{
		Role:    role,
		Content: content,
	})
}

// AddMessageWithBlocks adds a message with content blocks to the agent's memory.
func (a *Agent) AddMessageWithBlocks(role, content string, blocks []provider.ContentBlock) {
	a.memory.Add(Message{
		Role:          role,
		Content:       content,
		ContentBlocks: blocks,
	})
}

// GetMessages returns all messages from memory
func (a *Agent) GetMessages() []Message {
	return a.memory.Messages()
}

// ClearMemory clears the agent's memory
func (a *Agent) ClearMemory() {
	a.memory.Clear()
}
