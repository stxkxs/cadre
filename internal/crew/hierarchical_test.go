package crew

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/provider"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/telemetry"
	"github.com/stxkxs/cadre/internal/testutil"
)

// makeHierarchicalSetup creates a standard test setup for hierarchical tests.
func makeHierarchicalSetup(t *testing.T, dag dagEntry, deps ...dagEntry) (*hierarchicalCoordinator, *mockExecutor) {
	t.Helper()

	entries := append([]dagEntry{dag}, deps...)
	return makeHierarchicalSetupFromEntries(t, entries...)
}

func makeHierarchicalSetupFromEntries(t *testing.T, entries ...dagEntry) (*hierarchicalCoordinator, *mockExecutor) {
	t.Helper()

	d := makeTestDAG(t, entries...)
	exec := newMockExecutor()
	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range d.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	// Collect unique agent names.
	agentNames := make(map[string]bool)
	for _, e := range entries {
		agentNames[e.agent] = true
	}
	agentNames["manager"] = true

	agents := make(map[string]*agent.Runtime)
	for name := range agentNames {
		// We need real runtimes for the manager since it calls Execute.
		// Use a MockProvider that we'll configure per test.
		agents[name] = nil // placeholder — tests set the manager runtime
	}

	logger := telemetry.NewLogger(true)
	coord := &hierarchicalCoordinator{
		dag:         d,
		agents:      agents,
		executor:    exec,
		stateMgr:    mgr,
		logger:      logger,
		eventBus:    nil, // nil bus is safe
		managerName: "manager",
	}

	return coord, exec
}

// makeManagerRuntime creates a Runtime for the manager with a mock provider.
func makeManagerRuntime(t *testing.T, responses []*provider.Response) *agent.Runtime {
	t.Helper()
	mock := &testutil.MockProvider{Responses: responses}
	cfg := testutil.TestConfig()
	agentCfg := &config.AgentConfig{
		Name:  "manager",
		Role:  "Manager",
		Goal:  "Oversee task execution",
		Tools: []string{},
		Memory: config.MemoryConfig{
			Type:      "conversation",
			MaxTokens: 50000,
		},
	}
	rt, err := agent.NewRuntimeWithProvider(cfg, agentCfg, mock, testutil.TestLogger())
	if err != nil {
		t.Fatalf("failed to create manager runtime: %v", err)
	}
	return rt
}

// delegateToolCall creates a provider.Response with a delegate_task tool call.
func delegateToolCall(id, taskName, agentName string) *provider.Response {
	input, _ := json.Marshal(map[string]string{
		"task_name":  taskName,
		"agent_name": agentName,
	})
	return &provider.Response{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []provider.ToolCall{
			{ID: id, Name: "delegate_task", Input: string(input)},
		},
	}
}

// checkStatusToolCall creates a provider.Response with a check_status tool call.
func checkStatusToolCall(id string) *provider.Response {
	return &provider.Response{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []provider.ToolCall{
			{ID: id, Name: "check_status", Input: "{}"},
		},
	}
}

// feedbackToolCall creates a provider.Response with a provide_feedback tool call.
func feedbackToolCall(id, taskName, feedback string) *provider.Response {
	input, _ := json.Marshal(map[string]string{
		"task_name": taskName,
		"feedback":  feedback,
	})
	return &provider.Response{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []provider.ToolCall{
			{ID: id, Name: "provide_feedback", Input: string(input)},
		},
	}
}

// finalResponse creates a provider.Response that ends the manager's turn.
func finalResponse(content string) *provider.Response {
	return &provider.Response{
		Content:    content,
		StopReason: "end_turn",
	}
}

func TestHierarchical_BasicDelegation(t *testing.T) {
	coord, exec := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
		dagEntry{"task-b", "developer", []string{"task-a"}},
	)

	// Manager will: delegate task-a, then delegate task-b, then finish.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "task-a", "developer"),
		delegateToolCall("call-2", "task-b", "developer"),
		finalResponse("All tasks completed successfully."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil // mock executor doesn't use runtime

	outputs, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := exec.executedNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 tasks executed, got %d: %v", len(names), names)
	}

	if outputs == nil {
		t.Fatal("expected non-nil outputs")
	}
}

func TestHierarchical_DependencyOrder(t *testing.T) {
	coord, exec := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
		dagEntry{"task-b", "developer", []string{"task-a"}},
	)

	// Manager tries to delegate task-b first (should fail — dependency not met),
	// then delegates task-a, then task-b.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "task-b", "developer"), // should return error
		delegateToolCall("call-2", "task-a", "developer"),
		delegateToolCall("call-3", "task-b", "developer"),
		finalResponse("Done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	outputs, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := exec.executedNames()
	// Only task-a and task-b should have been executed (first attempt at task-b was rejected)
	if len(names) != 2 {
		t.Fatalf("expected 2 tasks executed, got %d: %v", len(names), names)
	}

	// Verify order: a before b
	if names[0] != "task-a" || names[1] != "task-b" {
		t.Errorf("expected [task-a, task-b], got %v", names)
	}

	if outputs == nil {
		t.Fatal("expected non-nil outputs")
	}
}

func TestHierarchical_FailureHandling(t *testing.T) {
	coord, exec := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
		dagEntry{"task-b", "developer", nil},
	)

	exec.failTasks["task-a"] = true

	// Manager delegates task-a (fails), then task-b (succeeds).
	// Manager finishes — but task-a is failed, so we expect an error.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "task-a", "developer"),
		delegateToolCall("call-2", "task-b", "developer"),
		finalResponse("Task-a failed, task-b completed."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err == nil {
		t.Fatal("expected error when task-a failed and was not retried")
	}
}

func TestHierarchical_AgentReassignment(t *testing.T) {
	coord, exec := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil}, // suggested: developer
	)

	// Manager delegates task-a to reviewer instead of developer.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "task-a", "reviewer"),
		finalResponse("Done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil
	coord.agents["reviewer"] = nil

	outputs, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := exec.executedNames()
	if len(names) != 1 || names[0] != "task-a" {
		t.Errorf("expected [task-a], got %v", names)
	}

	if outputs == nil {
		t.Fatal("expected non-nil outputs")
	}
}

func TestHierarchical_OutputPropagation(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"producer", "developer", nil},
		dagEntry{"consumer", "developer", []string{"producer"}},
	)

	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "producer", "developer"),
		delegateToolCall("call-2", "consumer", "developer"),
		finalResponse("Done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Consumer should have received producer's outputs as inputs.
	consumer, _ := coord.dag.GetTask("consumer")
	val, ok := consumer.GetInput("result")
	if !ok {
		t.Fatal("consumer should have received 'result' input from producer")
	}
	if val != "output from producer" {
		t.Errorf("expected 'output from producer', got %v", val)
	}
}

func TestHierarchical_ManagerFailure(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
	)

	// Manager provider fails.
	failingMock := &testutil.MockProvider{
		ShouldFail: true,
		FailErr:    fmt.Errorf("API error (status 500): internal server error"),
	}
	cfg := testutil.TestConfig()
	agentCfg := &config.AgentConfig{
		Name:  "manager",
		Role:  "Manager",
		Goal:  "Oversee",
		Tools: []string{},
		Memory: config.MemoryConfig{
			Type:      "conversation",
			MaxTokens: 10000,
		},
	}
	managerRT, _ := agent.NewRuntimeWithProvider(cfg, agentCfg, failingMock, testutil.TestLogger())

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err == nil {
		t.Fatal("expected error when manager LLM fails")
	}
}

func TestHierarchical_IncompleteDAG(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
		dagEntry{"task-b", "developer", nil},
	)

	// Manager only delegates task-a and finishes without completing task-b.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		delegateToolCall("call-1", "task-a", "developer"),
		finalResponse("I'm done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err == nil {
		t.Fatal("expected error when manager finishes without completing all tasks")
	}
}

func TestHierarchical_CheckStatusTool(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
	)

	// Manager checks status, then delegates, then finishes.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		checkStatusToolCall("call-1"),
		delegateToolCall("call-2", "task-a", "developer"),
		finalResponse("Done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHierarchical_ProvideFeedbackTool(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
	)

	// Manager provides feedback, then delegates, then finishes.
	managerRT := makeManagerRuntime(t, []*provider.Response{
		feedbackToolCall("call-1", "task-a", "Focus on error handling"),
		delegateToolCall("call-2", "task-a", "developer"),
		finalResponse("Done."),
	})

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil

	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify feedback was set.
	taskA, _ := coord.dag.GetTask("task-a")
	fb, ok := taskA.GetInput("_manager_feedback")
	if !ok {
		t.Fatal("expected _manager_feedback to be set")
	}
	if fb != "Focus on error handling" {
		t.Errorf("expected feedback 'Focus on error handling', got %v", fb)
	}
}

func TestHierarchical_Timeout(t *testing.T) {
	coord, _ := makeHierarchicalSetupFromEntries(t,
		dagEntry{"task-a", "developer", nil},
	)

	// Use a slow provider that exceeds the timeout.
	slowMock := &testutil.MockProvider{
		Delay: 5 * time.Second,
		Responses: []*provider.Response{
			finalResponse("done"),
		},
	}
	cfg := testutil.TestConfig()
	agentCfg := &config.AgentConfig{
		Name:  "manager",
		Role:  "Manager",
		Goal:  "Oversee",
		Tools: []string{},
		Memory: config.MemoryConfig{
			Type:      "conversation",
			MaxTokens: 10000,
		},
	}
	managerRT, _ := agent.NewRuntimeWithProvider(cfg, agentCfg, slowMock, testutil.TestLogger())

	coord.managerRuntime = managerRT
	coord.agents["manager"] = managerRT
	coord.agents["developer"] = nil
	coord.timeout = 50 * time.Millisecond

	_, err := coord.execute(context.Background())
	if err == nil {
		t.Fatal("expected error when manager exceeds timeout")
	}
	// The error should originate from context deadline exceeded.
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline exceeded error, got: %v", err)
	}
}

func TestNewHierarchicalCoordinator_MissingManager(t *testing.T) {
	dag := makeTestDAG(t, dagEntry{"task-a", "developer", nil})
	mgr := makeTestStateMgr(t)
	logger := telemetry.NewLogger(true)

	agents := map[string]*agent.Runtime{
		"developer": nil,
	}

	_, err := newHierarchicalCoordinator(dag, agents, newMockExecutor(), mgr, "nonexistent", logger, nil)
	if err == nil {
		t.Fatal("expected error for missing manager agent")
	}
}
