package task

import (
	"sort"
	"testing"

	"github.com/stxkxs/cadre/internal/config"
)

func makeTask(name string, deps ...string) *Task {
	return NewTask(&config.TaskConfig{
		Name:         name,
		Description:  "test task " + name,
		Agent:        "test-agent",
		Dependencies: deps,
		Timeout:      "5m",
	})
}

func TestDAG_AddTask(t *testing.T) {
	dag := NewDAG()

	if err := dag.AddTask(makeTask("a")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate should fail
	err := dag.AddTask(makeTask("a"))
	if err == nil {
		t.Fatal("expected error for duplicate task")
	}
}

func TestDAG_GetTask(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))

	task, ok := dag.GetTask("a")
	if !ok || task.Name() != "a" {
		t.Fatal("expected to find task a")
	}

	_, ok = dag.GetTask("missing")
	if ok {
		t.Fatal("expected not to find missing task")
	}
}

func TestDAG_Validate_Valid(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	if err := dag.Validate(); err != nil {
		t.Fatalf("expected valid DAG: %v", err)
	}
}

func TestDAG_Validate_MissingDep(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "nonexistent"))

	err := dag.Validate()
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestDAG_Validate_Cycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "c"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	err := dag.Validate()
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestDAG_Validate_SelfCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "a"))

	err := dag.Validate()
	if err == nil {
		t.Fatal("expected error for self-referencing dependency")
	}
}

func TestDAG_TopologicalSort_Linear(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(sorted))
	}

	names := make([]string, len(sorted))
	for i, task := range sorted {
		names[i] = task.Name()
	}

	// a must come before b, b before c
	posA, posB, posC := indexOf(names, "a"), indexOf(names, "b"), indexOf(names, "c")
	if posA >= posB || posB >= posC {
		t.Fatalf("wrong order: %v", names)
	}
}

func TestDAG_TopologicalSort_Diamond(t *testing.T) {
	// a -> b, a -> c, b -> d, c -> d
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "a"))
	dag.AddTask(makeTask("d", "b", "c"))

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(sorted))
	}

	names := make([]string, len(sorted))
	for i, task := range sorted {
		names[i] = task.Name()
	}

	posA := indexOf(names, "a")
	posB := indexOf(names, "b")
	posC := indexOf(names, "c")
	posD := indexOf(names, "d")

	if posA >= posB || posA >= posC {
		t.Fatalf("a should come before b and c: %v", names)
	}
	if posB >= posD || posC >= posD {
		t.Fatalf("b and c should come before d: %v", names)
	}
}

func TestDAG_TopologicalSort_Independent(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))
	dag.AddTask(makeTask("c"))

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(sorted))
	}
}

func TestDAG_GetReady_InitiallyReady(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))
	dag.AddTask(makeTask("c", "a"))

	ready := dag.GetReady()
	names := taskNames(ready)
	sort.Strings(names)

	if len(names) != 2 {
		t.Fatalf("expected 2 ready tasks, got %d: %v", len(names), names)
	}
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("expected [a b], got %v", names)
	}
}

func TestDAG_GetReady_AfterCompletion(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))

	// Initially only a is ready
	ready := dag.GetReady()
	if len(ready) != 1 || ready[0].Name() != "a" {
		t.Fatalf("expected only a ready, got %v", taskNames(ready))
	}

	// Complete a
	task, _ := dag.GetTask("a")
	task.Complete(nil)

	// Now b should be ready
	ready = dag.GetReady()
	if len(ready) != 1 || ready[0].Name() != "b" {
		t.Fatalf("expected only b ready, got %v", taskNames(ready))
	}
}

func TestDAG_GetReady_PartialDeps(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))
	dag.AddTask(makeTask("c", "a", "b"))

	// Complete only a
	task, _ := dag.GetTask("a")
	task.Complete(nil)

	ready := dag.GetReady()
	names := taskNames(ready)

	// c should NOT be ready (b not completed), but b should be ready
	if len(names) != 1 || names[0] != "b" {
		t.Fatalf("expected only b ready, got %v", names)
	}
}

func TestDAG_IsComplete(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))

	if dag.IsComplete() {
		t.Fatal("should not be complete with pending tasks")
	}

	a, _ := dag.GetTask("a")
	a.Complete(nil)

	if dag.IsComplete() {
		t.Fatal("should not be complete with one pending task")
	}

	b, _ := dag.GetTask("b")
	b.Complete(nil)

	if !dag.IsComplete() {
		t.Fatal("should be complete when all tasks completed")
	}
}

func TestDAG_IsComplete_WithFailures(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))

	a, _ := dag.GetTask("a")
	a.Fail(nil)

	if !dag.IsComplete() {
		t.Fatal("should be complete when all tasks are failed")
	}
}

func TestDAG_HasFailures(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))

	if dag.HasFailures() {
		t.Fatal("no failures yet")
	}

	a, _ := dag.GetTask("a")
	a.Fail(nil)

	if !dag.HasFailures() {
		t.Fatal("should report failures")
	}
}

func TestDAG_Reset(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))

	a, _ := dag.GetTask("a")
	a.Complete(map[string]interface{}{"key": "val"})
	b, _ := dag.GetTask("b")
	b.Fail(nil)

	dag.Reset()

	for _, task := range dag.GetTasks() {
		if task.GetStatus() != "pending" {
			t.Fatalf("expected pending after reset, got %s for task %s", task.GetStatus(), task.Name())
		}
		if task.Attempts != 0 {
			t.Fatalf("expected 0 attempts after reset, got %d", task.Attempts)
		}
	}
}

func TestDAG_GetTasks(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b"))
	dag.AddTask(makeTask("c"))

	tasks := dag.GetTasks()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestDAG_GetDependencies(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))

	deps := dag.GetDependencies("b")
	if len(deps) != 1 || deps[0] != "a" {
		t.Fatalf("expected [a], got %v", deps)
	}

	deps = dag.GetDependencies("a")
	if len(deps) != 0 {
		t.Fatalf("expected no deps for a, got %v", deps)
	}
}

func TestDAG_GetChildren(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "a"))

	children := dag.GetChildren("a")
	sort.Strings(children)
	if len(children) != 2 || children[0] != "b" || children[1] != "c" {
		t.Fatalf("expected [b c], got %v", children)
	}
}

func TestDAG_HasCycles_WithCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "b"))
	dag.AddTask(makeTask("b", "a"))

	if !dag.HasCycles() {
		t.Fatal("expected cycle to be detected")
	}
}

func TestDAG_HasCycles_NoCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	if dag.HasCycles() {
		t.Fatal("expected no cycle")
	}
}

func TestDAG_ValidateDeps_Valid(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "b"))
	dag.AddTask(makeTask("b", "a"))

	// ValidateDeps should not error on cycles — only checks refs exist
	if err := dag.ValidateDeps(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDAG_ValidateDeps_MissingRef(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "nonexistent"))

	err := dag.ValidateDeps()
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestDAG_Linearize_WithCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "b"))
	dag.AddTask(makeTask("b", "a"))

	tasks, err := dag.Linearize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Both tasks should appear in the result
	names := taskNames(tasks)
	sort.Strings(names)
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("expected [a b], got %v", names)
	}
}

func TestDAG_Linearize_NoCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	tasks, err := dag.Linearize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	names := taskNames(tasks)
	// Without cycles, should fall back to TopologicalSort: a before b before c
	posA, posB, posC := indexOf(names, "a"), indexOf(names, "b"), indexOf(names, "c")
	if posA >= posB || posB >= posC {
		t.Fatalf("wrong order: %v", names)
	}
}

func TestDAG_Linearize_ThreeWayCycle(t *testing.T) {
	dag := NewDAG()
	dag.AddTask(makeTask("a", "c"))
	dag.AddTask(makeTask("b", "a"))
	dag.AddTask(makeTask("c", "b"))

	tasks, err := dag.Linearize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

// helpers

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func taskNames(tasks []*Task) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.Name()
	}
	return names
}
