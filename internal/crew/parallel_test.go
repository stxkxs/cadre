package crew

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cadre-oss/cadre/internal/agent"
	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/task"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// mockExecutor for parallel tests — implements task.TaskExecutor.
type mockExecutor struct {
	mu         sync.Mutex
	executed   []string
	failTasks  map[string]bool
	delay      time.Duration
	onExecute  func(t *task.Task) // optional callback for timing tests
	maxActive  int64
	curActive  int64
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{failTasks: make(map[string]bool)}
}

func (e *mockExecutor) Execute(ctx context.Context, t *task.Task, runtime *agent.Runtime) error {
	atomic.AddInt64(&e.curActive, 1)
	defer atomic.AddInt64(&e.curActive, -1)

	// Track peak concurrency.
	cur := atomic.LoadInt64(&e.curActive)
	for {
		old := atomic.LoadInt64(&e.maxActive)
		if cur <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&e.maxActive, old, cur) {
			break
		}
	}

	if e.onExecute != nil {
		e.onExecute(t)
	}

	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-ctx.Done():
			t.Fail(ctx.Err())
			return ctx.Err()
		}
	}

	e.mu.Lock()
	shouldFail := e.failTasks[t.Name()]
	e.executed = append(e.executed, t.Name())
	e.mu.Unlock()

	if shouldFail {
		err := fmt.Errorf("task %s failed", t.Name())
		t.Fail(err)
		return err
	}

	t.Complete(map[string]interface{}{
		"result":    fmt.Sprintf("output from %s", t.Name()),
		"_response": fmt.Sprintf("mock response for %s", t.Name()),
	})
	return nil
}

func (e *mockExecutor) executedNames() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	names := make([]string, len(e.executed))
	copy(names, e.executed)
	return names
}

func makeTestDAG(t *testing.T, tasks ...dagEntry) *task.DAG {
	t.Helper()
	dag := task.NewDAG()
	for _, e := range tasks {
		tsk := task.NewTask(&config.TaskConfig{
			Name:         e.name,
			Description:  "test " + e.name,
			Agent:        e.agent,
			Dependencies: e.deps,
			Timeout:      "5m",
		})
		if err := dag.AddTask(tsk); err != nil {
			t.Fatalf("failed to add task %s: %v", e.name, err)
		}
	}
	return dag
}

type dagEntry struct {
	name  string
	agent string
	deps  []string
}

func makeTestAgents(names ...string) map[string]*agent.Runtime {
	// We don't actually call the agent runtime in parallel tests since
	// the mock executor intercepts execution. Return nil runtimes.
	agents := make(map[string]*agent.Runtime)
	for _, name := range names {
		agents[name] = nil
	}
	return agents
}

func makeTestStateMgr(t *testing.T) *state.Manager {
	t.Helper()
	mgr, err := state.NewManager("memory", "")
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}
	return mgr
}

func TestParallel_IndependentTasksRunConcurrently(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", nil},
		dagEntry{"c", "agent1", nil},
	)

	exec := newMockExecutor()
	exec.delay = 50 * time.Millisecond

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   3,
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)

	start := time.Now()
	outputs, err := coord.execute(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outputs == nil {
		t.Fatal("expected non-nil outputs")
	}

	// With 3 concurrent workers and 50ms each, should take ~50ms, not ~150ms.
	if elapsed > 200*time.Millisecond {
		t.Errorf("tasks did not run concurrently; elapsed=%v (expected ~50ms)", elapsed)
	}

	if len(exec.executedNames()) != 3 {
		t.Errorf("expected 3 tasks executed, got %d", len(exec.executedNames()))
	}
}

func TestParallel_DependenciesRespected(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", []string{"a"}},
	)

	exec := newMockExecutor()

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   4,
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := exec.executedNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 executed, got %d", len(names))
	}

	// "a" must come before "b"
	idxA, idxB := -1, -1
	for i, n := range names {
		if n == "a" {
			idxA = i
		}
		if n == "b" {
			idxB = i
		}
	}
	if idxA >= idxB {
		t.Fatalf("expected a before b, got order: %v", names)
	}
}

func TestParallel_ConcurrencyLimitEnforced(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", nil},
		dagEntry{"c", "agent1", nil},
		dagEntry{"d", "agent1", nil},
	)

	exec := newMockExecutor()
	exec.delay = 50 * time.Millisecond

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   2, // Only 2 workers
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	peak := atomic.LoadInt64(&exec.maxActive)
	if peak > 2 {
		t.Errorf("concurrency limit violated: peak active tasks = %d (limit 2)", peak)
	}
}

func TestParallel_FailFast(t *testing.T) {
	// a (fast, fails) -> triggers fail-fast before c (slow) and d (blocked) run
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", nil},
		dagEntry{"c", "agent1", []string{"b"}},
		dagEntry{"d", "agent1", []string{"b"}},
	)

	exec := newMockExecutor()
	exec.failTasks["a"] = true
	// b takes time so c and d haven't been queued yet when a fails

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   4,
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())

	if err == nil {
		t.Fatal("expected error from fail-fast strategy")
	}
}

func TestParallel_CompleteRunning(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"fast-fail", "agent1", nil},
		dagEntry{"slow-ok", "agent1", nil},
		dagEntry{"not-started", "agent1", []string{"fast-fail", "slow-ok"}},
	)

	exec := newMockExecutor()
	exec.failTasks["fast-fail"] = true

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   3,
		ErrorStrategy: "complete-running",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())

	if err == nil {
		t.Fatal("expected error from complete-running strategy")
	}

	names := exec.executedNames()
	// fast-fail and slow-ok should both execute; not-started should NOT
	for _, n := range names {
		if n == "not-started" {
			t.Fatal("not-started should not have been executed")
		}
	}
}

func TestParallel_ContinueAll(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", nil},
		dagEntry{"c", "agent1", []string{"b"}}, // depends on b, not a
	)

	exec := newMockExecutor()
	exec.failTasks["a"] = true // a fails, but b and c should still run

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   3,
		ErrorStrategy: "continue-all",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())

	if err == nil {
		t.Fatal("expected error (a failed)")
	}

	names := exec.executedNames()
	// b and c should still execute despite a failing
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["b"] || !found["c"] {
		t.Errorf("expected b and c to run despite a's failure, executed: %v", names)
	}
}

func TestParallel_OutputPropagation(t *testing.T) {
	dag := makeTestDAG(t,
		dagEntry{"producer", "agent1", nil},
		dagEntry{"consumer", "agent1", []string{"producer"}},
	)

	exec := newMockExecutor()

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   2,
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Consumer should have received producer's outputs as inputs.
	consumer, _ := dag.GetTask("consumer")
	val, ok := consumer.GetInput("result")
	if !ok {
		t.Fatal("consumer should have received 'result' input from producer")
	}
	if val != "output from producer" {
		t.Errorf("expected 'output from producer', got '%v'", val)
	}
}

func TestParallel_DiamondDAG(t *testing.T) {
	// a -> b, a -> c, b -> d, c -> d
	dag := makeTestDAG(t,
		dagEntry{"a", "agent1", nil},
		dagEntry{"b", "agent1", []string{"a"}},
		dagEntry{"c", "agent1", []string{"a"}},
		dagEntry{"d", "agent1", []string{"b", "c"}},
	)

	var order []string
	var orderMu sync.Mutex

	exec := newMockExecutor()
	exec.onExecute = func(tsk *task.Task) {
		orderMu.Lock()
		order = append(order, tsk.Name())
		orderMu.Unlock()
	}

	mgr := makeTestStateMgr(t)
	run, _ := mgr.StartRun("test-crew", nil)
	for _, tsk := range dag.GetTasks() {
		run.UpdateTask(state.TaskState{Name: tsk.Name(), Agent: tsk.Agent(), Status: "pending"})
	}
	mgr.UpdateRun(run)

	crewCfg := &config.CrewConfig{
		Name:          "test",
		Concurrency:   4,
		ErrorStrategy: "fail-fast",
	}
	logger := telemetry.NewLogger(true)

	coord := newParallelCoordinator(dag, makeTestAgents("agent1"), exec, mgr, crewCfg, logger, nil)
	_, err := coord.execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 tasks, got %d: %v", len(order), order)
	}

	// a must come before b and c; b and c must come before d
	pos := make(map[string]int)
	for i, n := range order {
		pos[n] = i
	}
	if pos["a"] >= pos["b"] || pos["a"] >= pos["c"] {
		t.Errorf("a should come before b and c: %v", order)
	}
	if pos["b"] >= pos["d"] || pos["c"] >= pos["d"] {
		t.Errorf("b and c should come before d: %v", order)
	}
}
