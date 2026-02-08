package mcp

import (
	"encoding/json"
	"fmt"
)

// ToolDef describes an MCP tool for tools/list.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// AllTools returns the full set of IPC tool definitions.
func AllTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "ipc_send_message",
			Description: "Send a message to another agent in this sprint session",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to_agent": map[string]any{"type": "string", "description": "Target agent ID"},
					"message":  map[string]any{"type": "string", "description": "Message content"},
					"priority": map[string]any{"type": "string", "description": "Priority: normal or urgent", "default": "normal"},
				},
				"required": []string{"to_agent", "message"},
			},
		},
		{
			Name:        "ipc_check_messages",
			Description: "Check for pending messages addressed to you",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "ipc_respond",
			Description: "Respond to a received message",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message_id": map[string]any{"type": "string", "description": "ID of the message to respond to"},
					"response":   map[string]any{"type": "string", "description": "Response content"},
					"status":     map[string]any{"type": "string", "description": "Response status (default: responded)", "default": "responded"},
				},
				"required": []string{"message_id", "response"},
			},
		},
		{
			Name:        "ipc_list_agents",
			Description: "List all agents in this sprint session with their current status",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "ipc_get_agent_status",
			Description: "Get detailed status of a specific agent",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent_id": map[string]any{"type": "string", "description": "Agent ID to query"},
				},
				"required": []string{"agent_id"},
			},
		},
		{
			Name:        "ipc_update_status",
			Description: "Update your own status (running, blocked, idle)",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "description": "New status: running, blocked, idle"},
					"detail": map[string]any{"type": "string", "description": "Optional detail about current work"},
				},
				"required": []string{"status"},
			},
		},
		{
			Name:        "ipc_mark_complete",
			Description: "Mark your task as complete with a summary",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{"type": "string", "description": "Summary of what was accomplished"},
				},
				"required": []string{"summary"},
			},
		},
	}
}

// ToolHandler dispatches tool calls to the store.
type ToolHandler struct {
	store   *Store
	agentID string
}

// NewToolHandler creates a handler bound to a specific agent and store.
func NewToolHandler(store *Store, agentID string) *ToolHandler {
	return &ToolHandler{store: store, agentID: agentID}
}

// Call dispatches a tool call by name with the given arguments.
func (h *ToolHandler) Call(name string, args json.RawMessage) (any, error) {
	switch name {
	case "ipc_send_message":
		return h.sendMessage(args)
	case "ipc_check_messages":
		return h.checkMessages()
	case "ipc_respond":
		return h.respond(args)
	case "ipc_list_agents":
		return h.listAgents()
	case "ipc_get_agent_status":
		return h.getAgentStatus(args)
	case "ipc_update_status":
		return h.updateStatus(args)
	case "ipc_mark_complete":
		return h.markComplete(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (h *ToolHandler) sendMessage(args json.RawMessage) (any, error) {
	var params struct {
		ToAgent  string `json:"to_agent"`
		Message  string `json:"message"`
		Priority string `json:"priority"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.ToAgent == "" || params.Message == "" {
		return nil, fmt.Errorf("to_agent and message are required")
	}

	id, err := h.store.SendMessage(h.agentID, params.ToAgent, params.Message, params.Priority)
	if err != nil {
		return nil, err
	}
	return map[string]string{"message_id": id, "status": "sent"}, nil
}

func (h *ToolHandler) checkMessages() (any, error) {
	msgs, err := h.store.CheckMessages(h.agentID)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		msgs = []Message{}
	}
	return map[string]any{"messages": msgs, "count": len(msgs)}, nil
}

func (h *ToolHandler) respond(args json.RawMessage) (any, error) {
	var params struct {
		MessageID string `json:"message_id"`
		Response  string `json:"response"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.MessageID == "" || params.Response == "" {
		return nil, fmt.Errorf("message_id and response are required")
	}

	if err := h.store.RespondToMessage(params.MessageID, params.Response, params.Status); err != nil {
		return nil, err
	}
	return map[string]string{"status": "responded"}, nil
}

func (h *ToolHandler) listAgents() (any, error) {
	agents, err := h.store.ListAgents()
	if err != nil {
		return nil, err
	}
	if agents == nil {
		agents = []AgentRecord{}
	}
	return map[string]any{"agents": agents, "count": len(agents)}, nil
}

func (h *ToolHandler) getAgentStatus(args json.RawMessage) (any, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	agent, err := h.store.GetAgent(params.AgentID)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (h *ToolHandler) updateStatus(args json.RawMessage) (any, error) {
	var params struct {
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	if err := h.store.UpdateAgentStatus(h.agentID, params.Status, params.Detail); err != nil {
		return nil, err
	}
	return map[string]string{"status": params.Status}, nil
}

func (h *ToolHandler) markComplete(args json.RawMessage) (any, error) {
	var params struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	if err := h.store.MarkAgentComplete(h.agentID, params.Summary); err != nil {
		return nil, err
	}
	return map[string]string{"status": "complete", "summary": params.Summary}, nil
}
