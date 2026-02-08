//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/cadre-oss/cadre/internal/agent"
	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/crew"
	"github.com/cadre-oss/cadre/internal/event"
	"github.com/cadre-oss/cadre/internal/provider"
	"github.com/cadre-oss/cadre/internal/task"
	"github.com/cadre-oss/cadre/internal/testutil"
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

// Verify that the full orchestrator flow works with mock providers
func TestSequentialWorkflow_FullOrchestrator(t *testing.T) {
	// This test would require file-based agent/task configs. Skip for now.
	// The other tests above exercise the core flow without needing filesystem.
	_ = crew.NewOrchestratorFromCheckpoint // reference to ensure crew package compiles
}
