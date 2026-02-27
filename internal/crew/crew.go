package crew

import (
	"context"
	"fmt"
	"strings"

	"github.com/stxkxs/cadre/internal/agent"
	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/event"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/task"
	"github.com/stxkxs/cadre/internal/telemetry"
)

// Orchestrator coordinates crew execution
type Orchestrator struct {
	config     *config.Config
	crewConfig *config.CrewConfig
	stateMgr   *state.Manager
	logger     *telemetry.Logger
	eventBus   *event.Bus

	agents       map[string]*agent.Runtime
	dag          *task.DAG
	executor     *task.Executor
	runState     *state.RunState
	sharedMemory *agent.SharedMemory // non-nil when agents use shared memory
}

// NewOrchestrator creates a new crew orchestrator.
// eventBus is optional — pass nil to disable event emission.
func NewOrchestrator(cfg *config.Config, crewCfg *config.CrewConfig, stateMgr *state.Manager, logger *telemetry.Logger, eventBus *event.Bus) (*Orchestrator, error) {
	o := &Orchestrator{
		config:     cfg,
		crewConfig: crewCfg,
		stateMgr:   stateMgr,
		logger:     logger,
		eventBus:   eventBus,
		agents:     make(map[string]*agent.Runtime),
		dag:        task.NewDAG(),
		executor:   task.NewExecutor(cfg, logger),
	}

	// Check if any agent uses shared memory — if so, create a shared store.
	var sharedStore agent.MemoryStore
	for _, agentName := range crewCfg.Agents {
		agentCfg, err := config.LoadAgent(agentName)
		if err != nil {
			continue
		}
		if agentCfg.Memory.Type == "shared" {
			if sharedStore == nil {
				dbPath := ".cadre/memory.db"
				if cfg.State.Path != "" {
					dbPath = cfg.State.Path + ".memory"
				}
				store, err := agent.NewSQLiteMemoryStore(dbPath)
				if err != nil {
					logger.Warn("Failed to create shared memory store", "error", err)
				} else {
					sharedStore = store
					o.sharedMemory = agent.NewSharedMemory(store, crewCfg.Name)
				}
			}
			break
		}
	}

	// Initialize agents
	for _, agentName := range crewCfg.Agents {
		agentCfg, err := config.LoadAgent(agentName)
		if err != nil {
			return nil, fmt.Errorf("failed to load agent %s: %w", agentName, err)
		}

		runtime, err := agent.NewRuntime(cfg, agentCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent runtime for %s: %w", agentName, err)
		}

		// Inject shared memory for agents configured with shared memory type.
		if agentCfg.Memory.Type == "shared" && o.sharedMemory != nil {
			if err := runtime.SetMemoryStore(o.sharedMemory.Store(), o.sharedMemory.Namespace()); err != nil {
				logger.Warn("Failed to set shared memory", "agent", agentName, "error", err)
			}
		}

		o.agents[agentName] = runtime
	}

	// Build task DAG
	for _, taskCfg := range crewCfg.Tasks {
		fullTaskCfg, err := config.LoadTask(taskCfg.Name)
		if err != nil {
			// Create minimal task config from crew task config
			fullTaskCfg = &config.TaskConfig{
				Name:         taskCfg.Name,
				Description:  taskCfg.Name,
				Agent:        taskCfg.Agent,
				Dependencies: taskCfg.DependsOn,
				Timeout:      "30m",
			}
		}

		t := task.NewTask(fullTaskCfg)
		if err := o.dag.AddTask(t); err != nil {
			return nil, fmt.Errorf("failed to add task %s: %w", taskCfg.Name, err)
		}
	}

	// Validate DAG — when max_iterations is set, allow cycles.
	if crewCfg.MaxIterations > 0 {
		if err := o.dag.ValidateDeps(); err != nil {
			return nil, fmt.Errorf("invalid task graph: %w", err)
		}
	} else {
		if err := o.dag.Validate(); err != nil {
			return nil, fmt.Errorf("invalid task graph: %w", err)
		}
	}

	return o, nil
}

// Execute runs the crew workflow
func (o *Orchestrator) Execute(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	o.logger.Info("Starting crew execution", "crew", o.crewConfig.Name, "process", o.crewConfig.Process)

	// Initialize run state
	runState, err := o.stateMgr.StartRun(o.crewConfig.Name, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to start run: %w", err)
	}
	o.runState = runState

	// Create root trace context and propagate through ctx.
	tc := telemetry.NewTraceContext(runState.ID)
	ctx = telemetry.ContextWithTrace(ctx, tc)

	// Initialize task states
	for _, t := range o.dag.GetTasks() {
		o.runState.UpdateTask(state.TaskState{
			Name:   t.Name(),
			Agent:  t.Agent(),
			Status: "pending",
		})
	}

	// Set initial inputs on root tasks
	for _, t := range o.dag.GetTasks() {
		if len(o.dag.GetDependencies(t.Name())) == 0 {
			for k, v := range inputs {
				t.SetInput(k, v)
			}
		}
	}

	// Emit crew.started
	o.eventBus.Emit(event.NewEvent(event.CrewStarted, map[string]interface{}{
		"crew":    o.crewConfig.Name,
		"process": o.crewConfig.Process,
	}))

	// Execute based on process type (iterative mode overrides when max_iterations > 0)
	var result map[string]interface{}
	if o.crewConfig.MaxIterations > 0 {
		result, err = o.executeIterative(ctx)
	} else {
		switch o.crewConfig.Process {
		case "sequential", "":
			result, err = o.executeSequential(ctx)
		case "parallel":
			result, err = o.executeParallel(ctx)
		case "hierarchical":
			result, err = o.executeHierarchical(ctx)
		default:
			return nil, fmt.Errorf("unknown process type: %s", o.crewConfig.Process)
		}
	}

	if err != nil {
		o.eventBus.Emit(event.NewEvent(event.CrewFailed, map[string]interface{}{
			"crew":  o.crewConfig.Name,
			"error": err.Error(),
		}))
		o.stateMgr.FailRun(err)
		return nil, err
	}

	o.eventBus.Emit(event.NewEvent(event.CrewCompleted, map[string]interface{}{
		"crew": o.crewConfig.Name,
	}))
	o.stateMgr.CompleteRun(result)
	return result, nil
}

// executeSequential runs tasks in topological order
func (o *Orchestrator) executeSequential(ctx context.Context) (map[string]interface{}, error) {
	// Get execution order
	tasks, err := o.dag.TopologicalSort()
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Get agent runtime for this task
		runtime, ok := o.agents[t.Agent()]
		if !ok {
			return nil, fmt.Errorf("agent not found for task %s: %s", t.Name(), t.Agent())
		}

		// Propagate outputs from dependencies as inputs
		for _, depName := range o.dag.GetDependencies(t.Name()) {
			depTask, _ := o.dag.GetTask(depName)
			for k, v := range depTask.GetOutputs() {
				if k != "_response" {
					t.SetInput(k, v)
				}
			}
		}

		// Update state
		o.stateMgr.UpdateTaskState(t.Name(), "running", nil, nil)
		o.eventBus.Emit(event.NewEvent(event.TaskStarted, map[string]interface{}{
			"task":  t.Name(),
			"agent": t.Agent(),
		}))

		// Execute task
		err := o.executor.Execute(ctx, t, runtime)
		if err != nil {
			o.stateMgr.UpdateTaskState(t.Name(), "failed", nil, err)
			o.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
				"task":  t.Name(),
				"error": err.Error(),
			}))
			return nil, fmt.Errorf("task %s failed: %w", t.Name(), err)
		}

		o.stateMgr.UpdateTaskState(t.Name(), "completed", t.GetOutputs(), nil)
		o.eventBus.Emit(event.NewEvent(event.TaskCompleted, map[string]interface{}{
			"task": t.Name(),
		}))
	}

	// Collect final outputs from leaf tasks
	outputs := make(map[string]interface{})
	for _, t := range tasks {
		if len(o.dag.GetChildren(t.Name())) == 0 {
			for k, v := range t.GetOutputs() {
				outputs[k] = v
			}
		}
	}

	return outputs, nil
}

// executeIterative runs tasks in a loop for cyclic workflows.
// Each iteration executes all tasks in linearized order, carrying outputs
// from previous iterations as inputs to the next.
func (o *Orchestrator) executeIterative(ctx context.Context) (map[string]interface{}, error) {
	maxIter := o.crewConfig.MaxIterations
	errorStrategy := o.crewConfig.ErrorStrategy

	// Track iteration metadata in run state
	o.stateMgr.SetMetadata("max_iterations", maxIter)

	var lastOutputs map[string]interface{}

	for iteration := 1; iteration <= maxIter; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		o.stateMgr.SetMetadata("current_iteration", iteration)
		o.logger.Info("Starting iteration", "crew", o.crewConfig.Name, "iteration", iteration, "max", maxIter)

		o.eventBus.Emit(event.NewEvent("crew.iteration.started", map[string]interface{}{
			"crew":      o.crewConfig.Name,
			"iteration": iteration,
			"max":       maxIter,
		}))

		// Get execution order (handles cycles)
		tasks, err := o.dag.Linearize()
		if err != nil {
			return nil, fmt.Errorf("failed to linearize task graph: %w", err)
		}

		// Inject previous iteration outputs as inputs
		if lastOutputs != nil {
			for _, t := range tasks {
				for k, v := range lastOutputs {
					t.SetInput(k, v)
				}
			}
		}

		var iterationErrors []error
		for _, t := range tasks {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			runtime, ok := o.agents[t.Agent()]
			if !ok {
				return nil, fmt.Errorf("agent not found for task %s: %s", t.Name(), t.Agent())
			}

			// Propagate outputs from dependencies within this iteration
			for _, depName := range o.dag.GetDependencies(t.Name()) {
				depTask, _ := o.dag.GetTask(depName)
				if depTask.GetStatus() == "completed" {
					for k, v := range depTask.GetOutputs() {
						if k != "_response" {
							t.SetInput(k, v)
						}
					}
				}
			}

			o.stateMgr.UpdateTaskState(t.Name(), "running", nil, nil)
			o.eventBus.Emit(event.NewEvent(event.TaskStarted, map[string]interface{}{
				"task":      t.Name(),
				"agent":     t.Agent(),
				"iteration": iteration,
			}))

			err := o.executor.Execute(ctx, t, runtime)
			if err != nil {
				o.stateMgr.UpdateTaskState(t.Name(), "failed", nil, err)
				o.eventBus.Emit(event.NewEvent(event.TaskFailed, map[string]interface{}{
					"task":      t.Name(),
					"error":     err.Error(),
					"iteration": iteration,
				}))
				taskErr := fmt.Errorf("task %s failed on iteration %d: %w", t.Name(), iteration, err)

				if errorStrategy == "continue-all" {
					iterationErrors = append(iterationErrors, taskErr)
					continue
				}
				// fail-fast (default) and complete-running (same as fail-fast for sequential)
				return nil, taskErr
			}

			o.stateMgr.UpdateTaskState(t.Name(), "completed", t.GetOutputs(), nil)
			o.eventBus.Emit(event.NewEvent(event.TaskCompleted, map[string]interface{}{
				"task":      t.Name(),
				"iteration": iteration,
			}))
		}

		if len(iterationErrors) > 0 {
			msgs := make([]string, len(iterationErrors))
			for i, e := range iterationErrors {
				msgs[i] = e.Error()
			}
			return nil, fmt.Errorf("iteration %d had %d errors: %s", iteration, len(iterationErrors), strings.Join(msgs, "; "))
		}

		o.eventBus.Emit(event.NewEvent("crew.iteration.completed", map[string]interface{}{
			"crew":      o.crewConfig.Name,
			"iteration": iteration,
		}))

		// Save outputs from all tasks before reset
		lastOutputs = make(map[string]interface{})
		for _, t := range tasks {
			for k, v := range t.GetOutputs() {
				lastOutputs[k] = v
			}
		}

		// Reset DAG for next iteration (unless this is the last one)
		if iteration < maxIter {
			o.dag.Reset()
		}
	}

	return lastOutputs, nil
}

// executeParallel runs independent tasks concurrently
func (o *Orchestrator) executeParallel(ctx context.Context) (map[string]interface{}, error) {
	coord := newParallelCoordinator(
		o.dag,
		o.agents,
		o.executor,
		o.stateMgr,
		o.crewConfig,
		o.logger,
		o.eventBus,
	)
	return coord.execute(ctx)
}

// executeHierarchical uses a manager agent to delegate tasks
func (o *Orchestrator) executeHierarchical(ctx context.Context) (map[string]interface{}, error) {
	if o.crewConfig.Manager == "" {
		return nil, fmt.Errorf("hierarchical process requires a manager agent (set 'manager' in crew config)")
	}

	coord, err := newHierarchicalCoordinator(
		o.dag,
		o.agents,
		o.executor,
		o.stateMgr,
		o.crewConfig.Manager,
		o.logger,
		o.eventBus,
	)
	if err != nil {
		return nil, err
	}

	// Apply crew-level timeout to manager execution.
	if timeout, tErr := o.config.Defaults.ParsedTimeout(); tErr == nil {
		coord.timeout = timeout
	}

	return coord.execute(ctx)
}

// GetState returns the current run state
func (o *Orchestrator) GetState() *state.RunState {
	return o.runState
}

// Resume continues execution from a checkpoint
func (o *Orchestrator) Resume(ctx context.Context) (map[string]interface{}, error) {
	if o.runState == nil {
		return nil, fmt.Errorf("no run state to resume")
	}

	o.logger.Info("Resuming execution", "run_id", o.runState.ID)

	// Find where we left off
	for _, taskState := range o.runState.Tasks {
		if taskState.Status == "completed" {
			// Restore task state
			if t, ok := o.dag.GetTask(taskState.Name); ok {
				t.Complete(taskState.Outputs)
			}
		}
	}

	// Continue with remaining tasks
	return o.executeSequential(ctx)
}

// NewOrchestratorFromCheckpoint creates an orchestrator from a checkpoint
func NewOrchestratorFromCheckpoint(cfg *config.Config, checkpoint *state.Checkpoint, stateMgr *state.Manager, logger *telemetry.Logger) (*Orchestrator, error) {
	// Load crew config
	crewCfg, err := config.LoadCrew(checkpoint.State.CrewName)
	if err != nil {
		return nil, fmt.Errorf("failed to load crew config: %w", err)
	}

	// Create orchestrator
	o, err := NewOrchestrator(cfg, crewCfg, stateMgr, logger, nil)
	if err != nil {
		return nil, err
	}

	// Restore state
	o.runState = &checkpoint.State

	return o, nil
}
