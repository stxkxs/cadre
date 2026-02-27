package crew

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/task"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// taskResult is sent from workers back to the scheduler.
type taskResult struct {
	task *task.Task
	err  error
}

// parallelCoordinator manages concurrent task execution.
type parallelCoordinator struct {
	dag           *task.DAG
	agents        map[string]*agent.Runtime
	executor      task.TaskExecutor
	stateMgr      *state.Manager
	crewCfg       *config.CrewConfig
	logger        *telemetry.Logger
	eventBus      *event.Bus
	concurrency   int
	errorStrategy string
}

func newParallelCoordinator(
	dag *task.DAG,
	agents map[string]*agent.Runtime,
	executor task.TaskExecutor,
	stateMgr *state.Manager,
	crewCfg *config.CrewConfig,
	logger *telemetry.Logger,
	eventBus *event.Bus,
) *parallelCoordinator {
	concurrency := crewCfg.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	// Don't spin up more workers than tasks
	taskCount := len(dag.GetTasks())
	if concurrency > taskCount {
		concurrency = taskCount
	}
	if concurrency < 1 {
		concurrency = 1
	}

	errorStrategy := crewCfg.ErrorStrategy
	if errorStrategy == "" {
		errorStrategy = "fail-fast"
	}

	return &parallelCoordinator{
		dag:           dag,
		agents:        agents,
		executor:      executor,
		stateMgr:      stateMgr,
		crewCfg:       crewCfg,
		logger:        logger,
		eventBus:      eventBus,
		concurrency:   concurrency,
		errorStrategy: errorStrategy,
	}
}

func (pc *parallelCoordinator) execute(ctx context.Context) (map[string]interface{}, error) {
	taskQueue := make(chan *task.Task, pc.concurrency)
	resultChan := make(chan taskResult, pc.concurrency)

	// Track which tasks have been submitted to avoid double-scheduling.
	queued := make(map[string]bool)

	// Use a cancellable context so we can stop workers on failure.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start workers.
	var wg sync.WaitGroup
	for i := 0; i < pc.concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pc.worker(ctx, id, taskQueue, resultChan)
		}(i)
	}

	// Close result channel when all workers exit.
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Scheduling loop.
	var firstErr error
	active := 0 // number of tasks currently in-flight

	// Seed the queue with initially ready tasks.
	for _, t := range pc.dag.GetReady() {
		if !queued[t.Name()] {
			queued[t.Name()] = true
			active++
			taskQueue <- t
		}
	}

	for !pc.dag.IsComplete() {
		if active == 0 {
			// No tasks in-flight and DAG not complete — means remaining tasks
			// are blocked by failures.
			break
		}

		result, ok := <-resultChan
		if !ok {
			break
		}
		active--

		if result.err != nil {
			pc.logger.Error("Task failed", "task", result.task.Name(), "error", result.err)

			switch pc.errorStrategy {
			case "fail-fast":
				cancel()
				// Drain remaining results.
				for active > 0 {
					if _, ok := <-resultChan; !ok {
						break
					}
					active--
				}
				close(taskQueue)
				return nil, result.err

			case "complete-running":
				if firstErr == nil {
					firstErr = result.err
				}
				// Don't schedule new tasks, just wait for active ones.
				for active > 0 {
					if r, ok := <-resultChan; !ok {
						break
					} else {
						active--
						if r.err != nil {
							pc.logger.Error("Task failed", "task", r.task.Name(), "error", r.err)
						}
					}
				}
				close(taskQueue)
				return nil, firstErr

			case "continue-all":
				if firstErr == nil {
					firstErr = result.err
				}
				// Keep scheduling — fall through to enqueue newly ready tasks.
			}
		}

		// Enqueue newly ready tasks.
		for _, t := range pc.dag.GetReady() {
			if !queued[t.Name()] {
				queued[t.Name()] = true
				active++
				taskQueue <- t
			}
		}
	}

	// Signal workers to stop.
	close(taskQueue)

	// Wait for workers to finish.
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Collect outputs from leaf tasks.
	outputs := make(map[string]interface{})
	for _, t := range pc.dag.GetTasks() {
		if len(pc.dag.GetChildren(t.Name())) == 0 && t.GetStatus() == "completed" {
			for k, v := range t.GetOutputs() {
				outputs[k] = v
			}
		}
	}

	return outputs, nil
}

// worker pulls tasks from taskQueue, executes them, and sends results.
func (pc *parallelCoordinator) worker(ctx context.Context, id int, taskQueue <-chan *task.Task, resultChan chan<- taskResult) {
	for t := range taskQueue {
		pc.executeOneTask(ctx, id, t, resultChan)
	}
}

// executeOneTask runs a single task with panic recovery.
func (pc *parallelCoordinator) executeOneTask(ctx context.Context, workerID int, t *task.Task, resultChan chan<- taskResult) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in worker %d executing task %s: %v", workerID, t.Name(), r)
			t.Fail(err)
			pc.stateMgr.UpdateTaskState(t.Name(), "failed", nil, err)
			pc.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
				"task":  t.Name(),
				"error": err.Error(),
			}))
			resultChan <- taskResult{task: t, err: err}
		}
	}()

	// Check context before executing.
	select {
	case <-ctx.Done():
		resultChan <- taskResult{task: t, err: ctx.Err()}
		return
	default:
	}

	pc.logger.Debug("Worker executing task", "worker", workerID, "task", t.Name())

	// Propagate dependency outputs as inputs.
	for _, depName := range pc.dag.GetDependencies(t.Name()) {
		depTask, _ := pc.dag.GetTask(depName)
		if depTask != nil {
			for k, v := range depTask.GetOutputs() {
				if k != "_response" {
					t.SetInput(k, v)
				}
			}
		}
	}

	// Update state → running.
	pc.stateMgr.UpdateTaskState(t.Name(), "running", nil, nil)
	pc.eventBus.Emit(event.NewEvent(event.TaskStarted, map[string]interface{}{
		"task":  t.Name(),
		"agent": t.Agent(),
	}))

	// Get agent runtime.
	rt, ok := pc.agents[t.Agent()]
	if !ok {
		err := fmt.Errorf("agent not found for task %s: %s", t.Name(), t.Agent())
		t.Fail(err)
		pc.stateMgr.UpdateTaskState(t.Name(), "failed", nil, err)
		pc.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
			"task":  t.Name(),
			"error": err.Error(),
		}))
		resultChan <- taskResult{task: t, err: err}
		return
	}

	// Execute task.
	err := pc.executor.Execute(ctx, t, rt)
	if err != nil {
		pc.stateMgr.UpdateTaskState(t.Name(), "failed", nil, err)
		pc.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
			"task":  t.Name(),
			"error": err.Error(),
		}))
		resultChan <- taskResult{task: t, err: err}
	} else {
		pc.stateMgr.UpdateTaskState(t.Name(), "completed", t.GetOutputs(), nil)
		pc.eventBus.Emit(event.NewEvent(event.TaskCompleted, map[string]interface{}{
			"task": t.Name(),
		}))
		resultChan <- taskResult{task: t, err: nil}
	}
}
