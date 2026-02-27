package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// --- Helpers ---

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": s.cfg.Version,
		"name":    s.cfg.Name,
	})
}

// --- Agents ---

func (s *Server) handleListAgents(w http.ResponseWriter, _ *http.Request) {
	names, err := config.LoadAgentList()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	agents := make([]interface{}, 0, len(names))
	for _, name := range names {
		cfg, err := config.LoadAgent(name)
		if err != nil {
			s.logger.Warn("Failed to load agent", "name", name, "error", err)
			continue
		}
		agents = append(agents, cfg)
	}
	jsonResponse(w, http.StatusOK, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := config.LoadAgent(name)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("agent not found: %s", name))
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var cfg config.AgentConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if cfg.Name == "" {
		jsonError(w, http.StatusBadRequest, "agent name is required")
		return
	}
	if err := writeYAML("agents", cfg.Name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, cfg)
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var cfg config.AgentConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg.Name = name
	if err := writeYAML("agents", name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := deleteYAML("agents", name); err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Message == "" {
		jsonError(w, http.StatusBadRequest, "message is required")
		return
	}

	agentCfg, err := config.LoadAgent(name)
	if err != nil {
		jsonError(w, http.StatusNotFound, "agent not found: "+name)
		return
	}

	runtime, err := crew.NewAgentRuntime(s.cfg, agentCfg, nil, s.logger)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer runtime.Close()

	s.streamResponse(w, r, runtime, body.Message)
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Message == "" {
		jsonError(w, http.StatusBadRequest, "message is required")
		return
	}

	runtime, err := s.chatMgr.GetOrCreate(name)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	s.streamResponse(w, r, runtime, body.Message)
}

func (s *Server) handleClearChat(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.chatMgr.Clear(name)
	w.WriteHeader(http.StatusNoContent)
}

// streamResponse sends an SSE stream from an agent runtime execution.
func (s *Server) streamResponse(w http.ResponseWriter, r *http.Request, runtime interface {
	StreamExecute(context.Context, string, agent.StreamCallback) (string, error)
}, message string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()

	callback := func(chunk string) {
		data, _ := json.Marshal(map[string]string{"t": "c", "c": chunk})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	_, err := runtime.StreamExecute(ctx, message, callback)
	if err != nil {
		data, _ := json.Marshal(map[string]string{"t": "error", "error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	data, _ := json.Marshal(map[string]string{"t": "done"})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// --- Tasks ---

func (s *Server) handleListTasks(w http.ResponseWriter, _ *http.Request) {
	names, err := config.LoadTaskList()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]interface{}, 0, len(names))
	for _, name := range names {
		cfg, err := config.LoadTask(name)
		if err != nil {
			s.logger.Warn("Failed to load task", "name", name, "error", err)
			continue
		}
		tasks = append(tasks, cfg)
	}
	jsonResponse(w, http.StatusOK, tasks)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := config.LoadTask(name)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("task not found: %s", name))
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var cfg config.TaskConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if cfg.Name == "" {
		jsonError(w, http.StatusBadRequest, "task name is required")
		return
	}
	if err := writeYAML("tasks", cfg.Name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, cfg)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var cfg config.TaskConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg.Name = name
	if err := writeYAML("tasks", name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := deleteYAML("tasks", name); err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Crews ---

func (s *Server) handleListCrews(w http.ResponseWriter, _ *http.Request) {
	names, err := config.LoadCrewList()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	crews := make([]interface{}, 0, len(names))
	for _, name := range names {
		cfg, err := config.LoadCrew(name)
		if err != nil {
			s.logger.Warn("Failed to load crew", "name", name, "error", err)
			continue
		}
		crews = append(crews, cfg)
	}
	jsonResponse(w, http.StatusOK, crews)
}

func (s *Server) handleGetCrew(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := config.LoadCrew(name)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("crew not found: %s", name))
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleCreateCrew(w http.ResponseWriter, r *http.Request) {
	var cfg config.CrewConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if cfg.Name == "" {
		jsonError(w, http.StatusBadRequest, "crew name is required")
		return
	}
	if err := config.ValidateCrew(&cfg); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := writeYAML("crews", cfg.Name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, cfg)
}

func (s *Server) handleUpdateCrew(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var cfg config.CrewConfig
	if err := decodeJSON(r, &cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg.Name = name
	if err := config.ValidateCrew(&cfg); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := writeYAML("crews", name, cfg); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

func (s *Server) handleDeleteCrew(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := deleteYAML("crews", name); err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleValidateCrew(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := config.LoadCrew(name)
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}
	if err := config.ValidateCrew(cfg); err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	})
}

// --- Runs ---

// activeRuns tracks in-progress crew runs with their cancel functions.
var activeRuns = struct {
	sync.Mutex
	runs map[string]context.CancelFunc
}{runs: make(map[string]context.CancelFunc)}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Crew   string                 `json:"crew"`
		Inputs map[string]interface{} `json:"inputs"`
	}
	if err := decodeJSON(r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Crew == "" {
		jsonError(w, http.StatusBadRequest, "crew name is required")
		return
	}

	crewCfg, err := config.LoadCrew(body.Crew)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	runID := uuid.New().String()
	runState, err := s.stateMgr.StartRun(body.Crew, body.Inputs)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	activeRuns.Lock()
	activeRuns.runs[runState.ID] = cancel
	activeRuns.Unlock()

	// Execute crew asynchronously.
	go func() {
		defer func() {
			activeRuns.Lock()
			delete(activeRuns.runs, runState.ID)
			activeRuns.Unlock()
		}()

		s.eventBus.Emit(event.NewEvent(event.CrewStarted, map[string]interface{}{
			"run_id": runState.ID,
			"crew":   body.Crew,
		}))

		orchestrator, err := crew.NewOrchestrator(s.cfg, crewCfg, s.stateMgr, s.logger, s.eventBus)
		if err != nil {
			s.stateMgr.FailRun(err)
			s.eventBus.Emit(event.NewEvent(event.CrewFailed, map[string]interface{}{
				"run_id": runState.ID,
				"error":  err.Error(),
			}))
			return
		}

		outputs, err := orchestrator.Execute(ctx, body.Inputs)
		if err != nil {
			s.stateMgr.FailRun(err)
			s.eventBus.Emit(event.NewEvent(event.CrewFailed, map[string]interface{}{
				"run_id": runState.ID,
				"error":  err.Error(),
			}))
			return
		}

		s.stateMgr.CompleteRun(outputs)
		s.eventBus.Emit(event.NewEvent(event.CrewCompleted, map[string]interface{}{
			"run_id":  runState.ID,
			"outputs": outputs,
		}))
	}()

	_ = runID // runState.ID is used from state manager
	jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"id":     runState.ID,
		"status": "running",
		"crew":   body.Crew,
	})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.stateMgr.ListRuns(50)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runs == nil {
		runs = []*state.RunState{}
	}
	jsonResponse(w, http.StatusOK, runs)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	runs, err := s.stateMgr.ListRuns(100)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, run := range runs {
		if run.ID == id {
			jsonResponse(w, http.StatusOK, run)
			return
		}
	}
	jsonError(w, http.StatusNotFound, "run not found")
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	activeRuns.Lock()
	cancel, ok := activeRuns.runs[id]
	activeRuns.Unlock()

	if !ok {
		jsonError(w, http.StatusNotFound, "run not found or already completed")
		return
	}
	cancel()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// --- Tools ---

func (s *Server) handleListTools(w http.ResponseWriter, _ *http.Request) {
	names, err := config.LoadToolList()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tools := make([]interface{}, 0, len(names))
	for _, name := range names {
		cfg, err := config.LoadTool(name)
		if err != nil {
			continue
		}
		tools = append(tools, cfg)
	}

	// Also include built-in tools.
	builtins := []map[string]string{
		{"name": "file_read", "description": "Read file contents", "provider": "builtin"},
		{"name": "file_write", "description": "Write file contents", "provider": "builtin"},
		{"name": "bash", "description": "Execute shell commands", "provider": "builtin"},
		{"name": "grep", "description": "Search file contents", "provider": "builtin"},
	}
	for _, b := range builtins {
		tools = append(tools, b)
	}

	jsonResponse(w, http.StatusOK, tools)
}

// --- SSE Events ---

func (s *Server) handleSSEEvents(w http.ResponseWriter, r *http.Request) {
	s.serveSSE(w, r, "")
}

func (s *Server) handleSSEEventsFiltered(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runID")
	s.serveSSE(w, r, runID)
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request, runID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	clientID := uuid.New().String()
	client := s.broker.Subscribe(r.Context(), clientID, runID)

	// Send initial connected event.
	data, _ := json.Marshal(map[string]string{"type": "connected", "client_id": clientID})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	for ev := range client.Events {
		data, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// --- Providers ---

func (s *Server) handleListProviders(w http.ResponseWriter, _ *http.Request) {
	providers := []map[string]interface{}{
		{"name": "anthropic", "label": "Anthropic API", "needs_key": true},
		{"name": "claudecode", "label": "Claude Code CLI", "needs_key": false},
	}
	jsonResponse(w, http.StatusOK, providers)
}

func (s *Server) handleClaudeCodeStatus(w http.ResponseWriter, _ *http.Request) {
	path, err := exec.LookPath("claude")
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"available": false,
			"error":     "claude binary not found in PATH",
		})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"available": true,
		"path":      path,
	})
}

// --- YAML file helpers ---

func writeYAML(dir, name string, data interface{}) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, name+".yaml"), content, 0644)
}

func deleteYAML(dir, name string) error {
	path := filepath.Join(dir, name+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete %s: %w", path, err)
	}
	return nil
}
