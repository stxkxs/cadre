package server

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// Server is the cadre web UI HTTP server.
type Server struct {
	cfg      *config.Config
	stateMgr *state.Manager
	eventBus *event.Bus
	broker   *Broker
	chatMgr  *ChatManager
	logger   *telemetry.Logger
}

// New creates a new server instance.
func New(cfg *config.Config, stateMgr *state.Manager, eventBus *event.Bus, logger *telemetry.Logger) *Server {
	broker := NewBroker(logger)
	// Register the broker as an event hook so cadre events broadcast to SSE clients.
	eventBus.Register(broker)

	return &Server{
		cfg:      cfg,
		stateMgr: stateMgr,
		eventBus: eventBus,
		broker:   broker,
		chatMgr:  NewChatManager(cfg, logger),
		logger:   logger,
	}
}

// Start starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context, addr string) error {
	mux := s.setupRoutes()

	srv := &http.Server{
		Addr:              addr,
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("Starting cadre web UI", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		s.chatMgr.Close()
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Agents
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/agents/{name}", s.handleGetAgent)
	mux.HandleFunc("POST /api/agents", s.handleCreateAgent)
	mux.HandleFunc("PUT /api/agents/{name}", s.handleUpdateAgent)
	mux.HandleFunc("DELETE /api/agents/{name}", s.handleDeleteAgent)
	mux.HandleFunc("POST /api/agents/{name}/test", s.handleTestAgent)
	mux.HandleFunc("POST /api/agents/{name}/chat", s.handleAgentChat)
	mux.HandleFunc("DELETE /api/agents/{name}/chat", s.handleClearChat)

	// Tasks
	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("GET /api/tasks/{name}", s.handleGetTask)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("PUT /api/tasks/{name}", s.handleUpdateTask)
	mux.HandleFunc("DELETE /api/tasks/{name}", s.handleDeleteTask)

	// Crews
	mux.HandleFunc("GET /api/crews", s.handleListCrews)
	mux.HandleFunc("GET /api/crews/{name}", s.handleGetCrew)
	mux.HandleFunc("POST /api/crews", s.handleCreateCrew)
	mux.HandleFunc("PUT /api/crews/{name}", s.handleUpdateCrew)
	mux.HandleFunc("DELETE /api/crews/{name}", s.handleDeleteCrew)
	mux.HandleFunc("POST /api/crews/{name}/validate", s.handleValidateCrew)

	// Runs
	mux.HandleFunc("POST /api/runs", s.handleStartRun)
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)

	// Tools
	mux.HandleFunc("GET /api/tools", s.handleListTools)

	// Providers
	mux.HandleFunc("GET /api/providers", s.handleListProviders)
	mux.HandleFunc("GET /api/providers/claudecode/status", s.handleClaudeCodeStatus)

	// SSE events
	mux.HandleFunc("GET /api/events", s.handleSSEEvents)
	mux.HandleFunc("GET /api/events/{runID}", s.handleSSEEventsFiltered)

	// Templates
	mux.HandleFunc("GET /api/templates", s.handleListTemplates)
	mux.HandleFunc("GET /api/templates/agents", s.handleListTemplateAgents)
	mux.HandleFunc("GET /api/templates/agents/{name}", s.handleGetTemplateAgent)
	mux.HandleFunc("GET /api/templates/tasks", s.handleListTemplateTasks)
	mux.HandleFunc("GET /api/templates/tasks/{name}", s.handleGetTemplateTask)
	mux.HandleFunc("GET /api/templates/crews", s.handleListTemplateCrews)
	mux.HandleFunc("GET /api/templates/crews/{name}", s.handleGetTemplateCrew)
	mux.HandleFunc("POST /api/templates/import", s.handleImportTemplate)

	// Static frontend (SPA fallback)
	mux.Handle("/", staticHandler())

	return mux
}

// corsMiddleware adds CORS headers for development mode.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetStaticFS allows overriding the embedded filesystem (used by static.go).
var StaticFS embed.FS
