package crew

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cadre-oss/cadre/internal/agent"
	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// InteractiveSession provides a chat interface with an agent
type InteractiveSession struct {
	config   *config.Config
	agentCfg *config.AgentConfig
	runtime  *agent.Runtime
	stateMgr *state.Manager
	logger   *telemetry.Logger
}

// NewInteractiveSession creates a new interactive session
func NewInteractiveSession(cfg *config.Config, agentCfg *config.AgentConfig, stateMgr *state.Manager, logger *telemetry.Logger) (*InteractiveSession, error) {
	runtime, err := agent.NewRuntime(cfg, agentCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runtime: %w", err)
	}

	return &InteractiveSession{
		config:   cfg,
		agentCfg: agentCfg,
		runtime:  runtime,
		stateMgr: stateMgr,
		logger:   logger,
	}, nil
}

// Run starts the interactive session
func (s *InteractiveSession) Run(ctx context.Context) error {
	fmt.Printf("Starting interactive session with %s\n", s.agentCfg.Name)
	fmt.Printf("Role: %s\n", s.agentCfg.Role)
	fmt.Printf("Goal: %s\n", s.agentCfg.Goal)
	fmt.Println()
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println("Type 'clear' to clear conversation history.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Handle special commands
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil
		case "clear":
			s.runtime.GetAgent().ClearMemory()
			fmt.Println("Conversation cleared.")
			continue
		case "":
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute agent with streaming output
		fmt.Printf("\n%s: ", s.agentCfg.Name)

		_, err = s.runtime.StreamExecute(ctx, input, func(chunk string) {
			fmt.Print(chunk)
		})
		if err != nil {
			fmt.Printf("\nError: %v\n\n", err)
			continue
		}

		fmt.Println()
		fmt.Println()
	}
}
