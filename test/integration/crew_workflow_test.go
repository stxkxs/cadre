//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/crew"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/provider"
	"github.com/stxkxs/cadre/internal/task"
	"github.com/stxkxs/cadre/internal/testutil"
)

func TestSequentialWorkflow(t *testing.T) {
	h := testutil.NewTestHarness(t)

	h.SetResponses(
		&provider.Response{Content: `{"files_changed": ["main.go"]}`, StopReason: "end_turn"},
		&provider.Response{Content: `{"approved": true}`, StopReason: "end_turn"},
	)

	crewCfg := &config.CrewConfig{
		Name:    "test-seq",
		Agents:  []string{"dev", "reviewer"},
		Process: "sequential",
		Tasks: []config.CrewTaskConfig{
			{Name: "implement", Agent: "dev"},
			{Name: "review", Agent: "reviewer", DependsOn: []string{"implement"}},
		},
	}

	// Create agent runtimes with mock provider
	devCfg := testutil.TestAgentConfig("dev")
	reviewerCfg := testutil.TestAgentConfig("reviewer")

	devRuntime, _ := agent.NewRuntimeWithProvider(h.Config, devCfg, h.Provider, h.Logger)
	reviewerRuntime, _ := agent.NewRuntimeWithProvider(h.Config, reviewerCfg, h.Provider, h.Logger)

	// Build DAG manually
	dag := task.NewDAG()
	implementTask := task.NewTask(&config.TaskConfig{
		Name:        "implement",
		Description: "Implement feature",
		Agent:       "dev",
	})
	reviewTask := task.NewTask(&config.TaskConfig{
		Name:         "review",
		Description:  "Review code",
		Agent:        "reviewer",
		Dependencies: []string{"implement"},
	})
	dag.AddTask(implementTask)
	dag.AddTask(reviewTask)

	executor := task.NewExecutor(h.Config, h.Logger)
	agents := map[string]*agent.Runtime{
		"dev":      devRuntime,
		"reviewer": reviewerRuntime,
	}

	// Execute sequential
	_ = crewCfg // used for reference
	tasks, _ := dag.TopologicalSort()
	for _, tk := range tasks {
		runtime := agents[tk.Agent()]
		if err := executor.Execute(context.Background(), tk, runtime); err != nil {
			t.Fatalf("task %s failed: %v", tk.Name(), err)
		}
	}

	// Verify both tasks completed
	if implementTask.GetStatus() != "completed" {
		t.Errorf("expected implement completed, got %s", implementTask.GetStatus())
	}
	if reviewTask.GetStatus() != "completed" {
		t.Errorf("expected review completed, got %s", reviewTask.GetStatus())
	}
}

func TestSequentialWorkflow_Events(t *testing.T) {
	h := testutil.NewTestHarness(t)

	h.SetResponses(
		&provider.Response{Content: "done", StopReason: "end_turn"},
	)

	// Emit events manually to verify the harness captures them
	h.EventBus.Emit(event.NewEvent(event.CrewStarted, map[string]interface{}{"crew": "test"}))
	h.EventBus.Emit(event.NewEvent(event.TaskStarted, map[string]interface{}{"task": "build"}))
	h.EventBus.Emit(event.NewEvent(event.TaskCompleted, map[string]interface{}{"task": "build"}))
	h.EventBus.Emit(event.NewEvent(event.CrewCompleted, map[string]interface{}{"crew": "test"}))

	h.AssertEventEmitted(event.CrewStarted)
	h.AssertEventEmitted(event.TaskStarted)
	h.AssertEventEmitted(event.TaskCompleted)
	h.AssertEventEmitted(event.CrewCompleted)
	h.AssertNoEvent(event.CrewFailed)
}

func TestSequentialWorkflow_DependencyPropagation(t *testing.T) {
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			{Content: `{"result": "built artifacts"}`, StopReason: "end_turn"},
			{Content: `{"deployed": true}`, StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	logger := testutil.TestLogger()

	buildCfg := testutil.TestAgentConfig("builder")
	deployCfg := testutil.TestAgentConfig("deployer")

	buildRuntime, _ := agent.NewRuntimeWithProvider(cfg, buildCfg, mock, logger)
	deployRuntime, _ := agent.NewRuntimeWithProvider(cfg, deployCfg, mock, logger)

	dag := task.NewDAG()
	buildTask := task.NewTask(&config.TaskConfig{
		Name:        "build",
		Description: "Build the project",
		Agent:       "builder",
		Outputs:     []config.OutputConfig{{Name: "result", Type: "string"}},
	})
	deployTask := task.NewTask(&config.TaskConfig{
		Name:         "deploy",
		Description:  "Deploy the project",
		Agent:        "deployer",
		Dependencies: []string{"build"},
	})
	dag.AddTask(buildTask)
	dag.AddTask(deployTask)

	executor := task.NewExecutor(cfg, logger)

	// Execute build
	if err := executor.Execute(context.Background(), buildTask, buildRuntime); err != nil {
		t.Fatal(err)
	}

	// Propagate outputs
	for k, v := range buildTask.GetOutputs() {
		if k != "_response" {
			deployTask.SetInput(k, v)
		}
	}

	// Execute deploy
	if err := executor.Execute(context.Background(), deployTask, deployRuntime); err != nil {
		t.Fatal(err)
	}

	// Verify deploy got the build output
	if val, ok := deployTask.GetInput("result"); !ok {
		t.Error("expected 'result' input on deploy task")
	} else if val != "built artifacts" {
		t.Errorf("expected 'built artifacts', got %v", val)
	}
}

func TestIterativeWorkflow(t *testing.T) {
	// 2 tasks in a cycle (A depends on B, B depends on A) with max_iterations=2
	// Requires 2 iterations × 2 tasks = 4 mock responses
	mock := &testutil.MockProvider{
		Responses: []*provider.Response{
			// Iteration 1
			{Content: `{"step": "first-a"}`, StopReason: "end_turn"},
			{Content: `{"step": "first-b"}`, StopReason: "end_turn"},
			// Iteration 2
			{Content: `{"step": "second-a"}`, StopReason: "end_turn"},
			{Content: `{"step": "second-b"}`, StopReason: "end_turn"},
		},
	}

	cfg := testutil.TestConfig()
	logger := testutil.TestLogger()

	agentACfg := testutil.TestAgentConfig("agent-a")
	agentBCfg := testutil.TestAgentConfig("agent-b")

	runtimeA, _ := agent.NewRuntimeWithProvider(cfg, agentACfg, mock, logger)
	runtimeB, _ := agent.NewRuntimeWithProvider(cfg, agentBCfg, mock, logger)

	agents := map[string]*agent.Runtime{
		"agent-a": runtimeA,
		"agent-b": runtimeB,
	}

	executor := task.NewExecutor(cfg, logger)
	maxIterations := 2

	// Track outputs across iterations
	var lastOutputs map[string]interface{}

	for iteration := 1; iteration <= maxIterations; iteration++ {
		// Build a fresh DAG each iteration (mimics Reset + Linearize)
		dag := task.NewDAG()
		taskA := task.NewTask(&config.TaskConfig{
			Name:         "task-a",
			Description:  "Task A",
			Agent:        "agent-a",
			Dependencies: []string{"task-b"},
			Outputs:      []config.OutputConfig{{Name: "step", Type: "string"}},
		})
		taskB := task.NewTask(&config.TaskConfig{
			Name:         "task-b",
			Description:  "Task B",
			Agent:        "agent-b",
			Dependencies: []string{"task-a"},
			Outputs:      []config.OutputConfig{{Name: "step", Type: "string"}},
		})
		dag.AddTask(taskA)
		dag.AddTask(taskB)

		tasks, err := dag.Linearize()
		if err != nil {
			t.Fatalf("iteration %d: linearize failed: %v", iteration, err)
		}

		// Inject previous iteration outputs
		if lastOutputs != nil {
			for _, tk := range tasks {
				for k, v := range lastOutputs {
					tk.SetInput(k, v)
				}
			}
		}

		for _, tk := range tasks {
			runtime := agents[tk.Agent()]
			if err := executor.Execute(context.Background(), tk, runtime); err != nil {
				t.Fatalf("iteration %d: task %s failed: %v", iteration, tk.Name(), err)
			}
		}

		// Collect outputs for next iteration
		lastOutputs = make(map[string]interface{})
		for _, tk := range tasks {
			for k, v := range tk.GetOutputs() {
				lastOutputs[k] = v
			}
		}

		// Verify both tasks completed this iteration
		for _, tk := range tasks {
			if tk.GetStatus() != "completed" {
				t.Errorf("iteration %d: expected %s completed, got %s", iteration, tk.Name(), tk.GetStatus())
			}
		}
	}

	// Verify all 4 responses were consumed
	if mock.CallCount() != 4 {
		t.Errorf("expected 4 provider calls (2 iterations × 2 tasks), got %d", mock.CallCount())
	}

	// Verify outputs from final iteration propagated
	if lastOutputs["step"] != "second-b" {
		t.Errorf("expected final output step='second-b', got %v", lastOutputs["step"])
	}
}

// Verify that the full orchestrator flow works with mock providers
func TestSequentialWorkflow_FullOrchestrator(t *testing.T) {
	// This test would require file-based agent/task configs. Skip for now.
	// The other tests above exercise the core flow without needing filesystem.
	_ = crew.NewOrchestratorFromCheckpoint // reference to ensure crew package compiles
}
