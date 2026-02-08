package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "cadre-ipc"
	serverVersion   = "0.1.0"
)

// Server is a minimal MCP server that speaks JSON-RPC 2.0 over stdin/stdout.
// It only implements initialize, tools/list, and tools/call.
type Server struct {
	store   *Store
	agentID string
	handler *ToolHandler
	in      io.Reader
	out     io.Writer
}

// NewServer creates a new MCP server for the given agent.
func NewServer(store *Store, agentID string) *Server {
	return &Server{
		store:   store,
		agentID: agentID,
		handler: NewToolHandler(store, agentID),
		in:      os.Stdin,
		out:     os.Stdout,
	}
}

// jsonrpcRequest is a JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *jsonrpcError  `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Run reads JSON-RPC requests from stdin and writes responses to stdout.
// It blocks until the context is cancelled or stdin is closed.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// MCP messages can be large
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, -32700, "parse error")
			continue
		}

		// Notifications (no ID) don't get responses
		if req.ID == nil {
			s.handleNotification(req)
			continue
		}

		result, err := s.dispatch(req)
		if err != nil {
			s.writeError(req.ID, -32603, err.Error())
			continue
		}

		s.writeResult(req.ID, result)
	}

	return scanner.Err()
}

func (s *Server) dispatch(req jsonrpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "tools/list":
		return s.handleToolsList(req.Params)
	case "tools/call":
		return s.handleToolsCall(req.Params)
	case "ping":
		return map[string]any{}, nil
	default:
		return nil, fmt.Errorf("method not found: %s", req.Method)
	}
}

func (s *Server) handleNotification(req jsonrpcRequest) {
	switch req.Method {
	case "notifications/initialized":
		// Client confirmed initialization — update agent heartbeat
		_ = s.store.Heartbeat(s.agentID)
	}
}

func (s *Server) handleInitialize(_ json.RawMessage) (any, error) {
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": serverVersion,
		},
	}, nil
}

func (s *Server) handleToolsList(_ json.RawMessage) (any, error) {
	return map[string]any{
		"tools": AllTools(),
	}, nil
}

func (s *Server) handleToolsCall(params json.RawMessage) (any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("parse tool call params: %w", err)
	}

	// Update heartbeat on each tool call
	_ = s.store.Heartbeat(s.agentID)

	result, err := s.handler.Call(call.Name, call.Arguments)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": fmt.Sprintf("Error: %s", err.Error())},
			},
			"isError": true,
		}, nil
	}

	text, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(text)},
		},
	}, nil
}

func (s *Server) writeResult(id json.RawMessage, result any) {
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(resp)
}

func (s *Server) writeError(id json.RawMessage, code int, message string) {
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonrpcError{Code: code, Message: message},
	}
	s.writeJSON(resp)
}

func (s *Server) writeJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = s.out.Write(data)
}
