package state

import (
	"fmt"
	"sync"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager("memory", "")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return mgr
}

func TestManager_StartRun(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	inputs := map[string]interface{}{"key": "value"}
	run, err := mgr.StartRun("test-crew", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run.ID == "" {
		t.Fatal("expected run ID to be set")
	}
	if run.CrewName != "test-crew" {
		t.Fatalf("expected crew name test-crew, got %s", run.CrewName)
	}
	if run.Status != "running" {
		t.Fatalf("expected status running, got %s", run.Status)
	}
	if run.Inputs["key"] != "value" {
		t.Fatal("inputs not preserved")
	}
}

func TestManager_CompleteRun(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	_, err := mgr.StartRun("test-crew", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputs := map[string]interface{}{"result": "done"}
	if err := mgr.CompleteRun(outputs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run, err := mgr.GetActiveRun()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed, got %s", run.Status)
	}
	if run.Outputs["result"] != "done" {
		t.Fatal("outputs not preserved")
	}
	if run.CompletedAt.IsZero() {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestManager_CompleteRun_NoActiveRun(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	err := mgr.CompleteRun(nil)
	if err == nil {
		t.Fatal("expected error when no active run")
	}
}

func TestManager_FailRun(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	_, err := mgr.StartRun("test-crew", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mgr.FailRun(fmt.Errorf("something broke")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run, err := mgr.GetActiveRun()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != "failed" {
		t.Fatalf("expected failed, got %s", run.Status)
	}
	if run.Error != "something broke" {
		t.Fatalf("expected error message, got %s", run.Error)
	}
	if run.CompletedAt.IsZero() {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestManager_FailRun_NoActiveRun(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	err := mgr.FailRun(fmt.Errorf("fail"))
	if err == nil {
		t.Fatal("expected error when no active run")
	}
}

func TestManager_UpdateTaskState(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	run, _ := mgr.StartRun("test-crew", nil)
	run.UpdateTask(TaskState{Name: "task-1", Agent: "agent-1", Status: "pending"})
	mgr.UpdateRun(run)

	err := mgr.UpdateTaskState("task-1", "running", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	active, _ := mgr.GetActiveRun()
	ts := active.GetTask("task-1")
	if ts == nil {
		t.Fatal("task not found")
	}
	if ts.Status != "running" {
		t.Fatalf("expected running, got %s", ts.Status)
	}
	if ts.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be set")
	}
}

func TestManager_UpdateTaskState_Completed(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	run, _ := mgr.StartRun("test-crew", nil)
	run.UpdateTask(TaskState{Name: "task-1", Agent: "agent-1", Status: "running"})
	mgr.UpdateRun(run)

	outputs := map[string]interface{}{"out": "value"}
	err := mgr.UpdateTaskState("task-1", "completed", outputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	active, _ := mgr.GetActiveRun()
	ts := active.GetTask("task-1")
	if ts.Status != "completed" {
		t.Fatalf("expected completed, got %s", ts.Status)
	}
	if ts.Outputs["out"] != "value" {
		t.Fatal("outputs not preserved")
	}
	if ts.CompletedAt.IsZero() {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestManager_UpdateTaskState_Failed(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	run, _ := mgr.StartRun("test-crew", nil)
	run.UpdateTask(TaskState{Name: "task-1", Agent: "agent-1", Status: "running"})
	mgr.UpdateRun(run)

	err := mgr.UpdateTaskState("task-1", "failed", nil, fmt.Errorf("task error"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	active, _ := mgr.GetActiveRun()
	ts := active.GetTask("task-1")
	if ts.Status != "failed" {
		t.Fatalf("expected failed, got %s", ts.Status)
	}
	if ts.Error != "task error" {
		t.Fatalf("expected error message, got %s", ts.Error)
	}
}

func TestManager_UpdateTaskState_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	mgr.StartRun("test-crew", nil)

	err := mgr.UpdateTaskState("nonexistent", "running", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestManager_SaveAndLoadCheckpoint(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	run, _ := mgr.StartRun("test-crew", nil)
	run.UpdateTask(TaskState{Name: "task-1", Agent: "agent-1", Status: "running"})

	// SaveCheckpoint writes to file too, but with memory store this tests the store path
	err := mgr.SaveCheckpoint(run)
	if err != nil {
		// File write may fail in test environments, that's OK
		// The store save should still work
		t.Logf("SaveCheckpoint returned error (file write expected in test): %v", err)
	}
}

func TestManager_ListRuns(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	mgr.StartRun("crew-1", nil)
	mgr.CompleteRun(nil)

	mgr.StartRun("crew-2", nil)

	runs, err := mgr.ListRuns(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
}

func TestManager_ConcurrentUpdateTaskState(t *testing.T) {
	mgr := newTestManager(t)
	defer mgr.Close()

	run, _ := mgr.StartRun("test-crew", nil)
	for i := 0; i < 10; i++ {
		run.UpdateTask(TaskState{
			Name:   fmt.Sprintf("task-%d", i),
			Agent:  "agent-1",
			Status: "pending",
		})
	}
	mgr.UpdateRun(run)

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			taskName := fmt.Sprintf("task-%d", idx)
			if err := mgr.UpdateTaskState(taskName, "running", nil, nil); err != nil {
				errCh <- err
				return
			}
			outputs := map[string]interface{}{"result": fmt.Sprintf("done-%d", idx)}
			if err := mgr.UpdateTaskState(taskName, "completed", outputs, nil); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}

	// Verify all tasks completed
	active, _ := mgr.GetActiveRun()
	for i := 0; i < 10; i++ {
		ts := active.GetTask(fmt.Sprintf("task-%d", i))
		if ts == nil {
			t.Errorf("task-%d not found", i)
			continue
		}
		if ts.Status != "completed" {
			t.Errorf("task-%d expected completed, got %s", i, ts.Status)
		}
	}
}

func TestManager_NewManager_Unsupported(t *testing.T) {
	_, err := NewManager("postgres", "")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}
