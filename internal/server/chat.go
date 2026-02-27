package server

import (
	"sync"
	"time"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/telemetry"
)

const chatSessionTimeout = 30 * time.Minute

type chatSession struct {
	runtime  *agent.Runtime
	lastUsed time.Time
}

// ChatManager keeps agent runtimes alive across chat messages so
// conversation memory persists within a session.
type ChatManager struct {
	mu       sync.Mutex
	sessions map[string]*chatSession
	cfg      *config.Config
	logger   *telemetry.Logger
	done     chan struct{}
}

// NewChatManager creates a chat manager that expires idle sessions.
func NewChatManager(cfg *config.Config, logger *telemetry.Logger) *ChatManager {
	cm := &ChatManager{
		sessions: make(map[string]*chatSession),
		cfg:      cfg,
		logger:   logger,
		done:     make(chan struct{}),
	}
	go cm.reapLoop()
	return cm
}

// GetOrCreate returns an existing runtime for the agent or creates a new one.
func (cm *ChatManager) GetOrCreate(agentName string) (*agent.Runtime, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if sess, ok := cm.sessions[agentName]; ok {
		sess.lastUsed = time.Now()
		return sess.runtime, nil
	}

	agentCfg, err := config.LoadAgent(agentName)
	if err != nil {
		return nil, err
	}

	runtime, err := crew.NewAgentRuntime(cm.cfg, agentCfg, nil, cm.logger)
	if err != nil {
		return nil, err
	}

	cm.sessions[agentName] = &chatSession{
		runtime:  runtime,
		lastUsed: time.Now(),
	}
	return runtime, nil
}

// Clear removes a chat session.
func (cm *ChatManager) Clear(agentName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if sess, ok := cm.sessions[agentName]; ok {
		sess.runtime.Close()
		delete(cm.sessions, agentName)
	}
}

// Close stops the reaper and cleans up all sessions.
func (cm *ChatManager) Close() {
	close(cm.done)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for name, sess := range cm.sessions {
		sess.runtime.Close()
		delete(cm.sessions, name)
	}
}

func (cm *ChatManager) reapLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cm.done:
			return
		case <-ticker.C:
			cm.mu.Lock()
			now := time.Now()
			for name, sess := range cm.sessions {
				if now.Sub(sess.lastUsed) > chatSessionTimeout {
					cm.logger.Debug("Reaping idle chat session", "agent", name)
					sess.runtime.Close()
					delete(cm.sessions, name)
				}
			}
			cm.mu.Unlock()
		}
	}
}
