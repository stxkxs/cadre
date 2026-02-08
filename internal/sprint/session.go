package sprint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/cadre-oss/cadre/internal/mcp"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// Session manages a sprint execution lifecycle.
type Session struct {
	config *SprintConfig
	store  *mcp.Store
	agents map[string]*AgentProcess
	logger *telemetry.Logger
	dbPath string
	mu     sync.Mutex
}

// AgentStatus reports an agent's current state.
type AgentStatus struct {
	AgentID   string `json:"agent_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	PID       int    `json:"pid,omitempty"`
	Task      string `json:"current_task,omitempty"`
	Summary   string `json:"summary,omitempty"`
	IsRunning bool   `json:"is_running"`
}

// NewSession initializes a sprint session.
func NewSession(cfg *SprintConfig, logger *telemetry.Logger) (*Session, error) {
	dbDir := filepath.Join(".cadre", "sessions")
	dbPath := filepath.Join(dbDir, cfg.Name+".db")

	store, err := mcp.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open session store: %w", err)
	}

	return &Session{
		config: cfg,
		store:  store,
		agents: make(map[string]*AgentProcess),
		logger: logger,
		dbPath: dbPath,
	}, nil
}

// Start launches all agents following the phase order.
func (s *Session) Start(ctx context.Context) error {
	crewBin, err := CadreBinaryPath()
	if err != nil {
		return fmt.Errorf("resolve cadre binary: %w", err)
	}

	if len(s.config.Phases) == 0 {
		// No phases defined — run all workstreams in parallel
		return s.launchWorkstreams(ctx, s.config.Workstreams, true, crewBin)
	}

	for _, phase := range s.config.Phases {
		s.logger.Info("Starting phase", "name", phase.Name)

		// Collect workstreams for this phase
		var ws []WorkstreamConfig
		for _, name := range phase.Workstreams {
			for _, w := range s.config.Workstreams {
				if w.Name == name {
					ws = append(ws, w)
					break
				}
			}
		}

		if err := s.launchWorkstreams(ctx, ws, phase.Parallel, crewBin); err != nil {
			return fmt.Errorf("phase %s: %w", phase.Name, err)
		}

		// Wait for all agents in this phase to complete
		if err := s.waitForAgents(ctx, ws); err != nil {
			return fmt.Errorf("phase %s: %w", phase.Name, err)
		}

		s.logger.Info("Phase completed", "name", phase.Name)
	}

	return nil
}

func (s *Session) launchWorkstreams(ctx context.Context, workstreams []WorkstreamConfig, parallel bool, crewBin string) error {
	if parallel {
		var wg sync.WaitGroup
		errCh := make(chan error, len(workstreams))

		for i := range workstreams {
			ws := &workstreams[i]
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errCh <- fmt.Errorf("workstream %s panicked: %v", ws.Name, r)
					}
				}()
				if err := s.launchOne(ctx, ws, crewBin); err != nil {
					errCh <- fmt.Errorf("workstream %s: %w", ws.Name, err)
				}
			}()
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			return err
		}
		return nil
	}

	// Sequential
	for i := range workstreams {
		if err := s.launchOne(ctx, &workstreams[i], crewBin); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) launchOne(ctx context.Context, ws *WorkstreamConfig, crewBin string) error {
	sessionID := uuid.New().String()

	// Register agent in DB before launching
	if err := s.store.RegisterAgent(ws.Name, ws.Name, 0, sessionID); err != nil {
		return fmt.Errorf("register agent %s: %w", ws.Name, err)
	}

	ap, err := LaunchAgent(ctx, ws, s.config.Name, s.dbPath, crewBin, sessionID)
	if err != nil {
		return fmt.Errorf("launch agent %s: %w", ws.Name, err)
	}

	// Update PID in DB
	if err := s.store.RegisterAgent(ws.Name, ws.Name, ap.PID, sessionID); err != nil {
		s.logger.Error("Failed to update agent PID", "agent", ws.Name, "error", err)
	}

	s.mu.Lock()
	s.agents[ws.Name] = ap
	s.mu.Unlock()

	s.logger.Info("Agent launched", "agent", ws.Name, "pid", ap.PID, "session", sessionID)
	return nil
}

func (s *Session) waitForAgents(ctx context.Context, workstreams []WorkstreamConfig) error {
	for _, ws := range workstreams {
		s.mu.Lock()
		ap, ok := s.agents[ws.Name]
		s.mu.Unlock()
		if !ok {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ap.Done:
			// Agent process exited — check if it completed successfully
			agent, err := s.store.GetAgent(ws.Name)
			if err != nil {
				s.logger.Error("Failed to get agent status", "agent", ws.Name, "error", err)
				continue
			}
			if agent.Status != "complete" {
				s.logger.Warn("Agent exited without completing", "agent", ws.Name, "status", agent.Status)
			}
		}
	}
	return nil
}

// Status returns the current status of all agents in the session.
func (s *Session) Status() (map[string]AgentStatus, error) {
	agents, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}

	result := make(map[string]AgentStatus)
	for _, a := range agents {
		isRunning := false
		s.mu.Lock()
		if ap, ok := s.agents[a.ID]; ok {
			select {
			case <-ap.Done:
			default:
				isRunning = true
			}
		}
		s.mu.Unlock()

		result[a.ID] = AgentStatus{
			AgentID:   a.ID,
			Name:      a.Name,
			Status:    a.Status,
			PID:       a.PID,
			Task:      a.CurrentTask,
			Summary:   a.Summary,
			IsRunning: isRunning,
		}
	}
	return result, nil
}

// Stop gracefully terminates all running agent processes.
func (s *Session) Stop() error {
	s.mu.Lock()
	agents := make(map[string]*AgentProcess)
	for k, v := range s.agents {
		agents[k] = v
	}
	s.mu.Unlock()

	for name, ap := range agents {
		select {
		case <-ap.Done:
			continue // already exited
		default:
		}

		s.logger.Info("Stopping agent", "agent", name, "pid", ap.PID)
		if ap.Cmd.Process != nil {
			_ = ap.Cmd.Process.Signal(os.Interrupt)
		}

		// Give it 10 seconds to clean up
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-ap.Done:
			timer.Stop()
		case <-timer.C:
			s.logger.Warn("Force-killing agent", "agent", name)
			if ap.Cmd.Process != nil {
				_ = ap.Cmd.Process.Kill()
			}
		}
	}

	return s.store.Close()
}

// Wait blocks until all agents have exited.
func (s *Session) Wait(ctx context.Context) error {
	s.mu.Lock()
	agents := make(map[string]*AgentProcess)
	for k, v := range s.agents {
		agents[k] = v
	}
	s.mu.Unlock()

	for name, ap := range agents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ap.Done:
			s.logger.Info("Agent finished", "agent", name)
		}
	}
	return nil
}

// RunGate executes the gate checks. Returns nil if all pass.
func (s *Session) RunGate(ctx context.Context) error {
	if len(s.config.Gate.Checks) == 0 {
		return nil
	}

	s.logger.Info("Running gate checks", "count", len(s.config.Gate.Checks))

	for _, check := range s.config.Gate.Checks {
		s.logger.Info("Gate check", "command", check)
		cmd := exec.CommandContext(ctx, "sh", "-c", check)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gate check failed: %s: %w", check, err)
		}
	}

	s.logger.Info("All gate checks passed")
	return nil
}
