package task

import (
	"fmt"

	cadreErrors "github.com/stxkxs/cadre/internal/errors"
)

// DAG represents a directed acyclic graph of tasks
type DAG struct {
	tasks    map[string]*Task
	deps     map[string][]string // task -> dependencies
	children map[string][]string // task -> tasks that depend on it
}

// NewDAG creates a new task DAG
func NewDAG() *DAG {
	return &DAG{
		tasks:    make(map[string]*Task),
		deps:     make(map[string][]string),
		children: make(map[string][]string),
	}
}

// AddTask adds a task to the DAG
func (d *DAG) AddTask(task *Task) error {
	if _, exists := d.tasks[task.Name()]; exists {
		return fmt.Errorf("task already exists: %s", task.Name())
	}

	d.tasks[task.Name()] = task
	d.deps[task.Name()] = task.Dependencies()

	// Update children map
	for _, dep := range task.Dependencies() {
		d.children[dep] = append(d.children[dep], task.Name())
	}

	return nil
}

// GetTask returns a task by name
func (d *DAG) GetTask(name string) (*Task, bool) {
	task, ok := d.tasks[name]
	return task, ok
}

// GetTasks returns all tasks
func (d *DAG) GetTasks() []*Task {
	tasks := make([]*Task, 0, len(d.tasks))
	for _, t := range d.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// GetDependencies returns dependencies for a task
func (d *DAG) GetDependencies(name string) []string {
	return d.deps[name]
}

// GetChildren returns tasks that depend on a given task
func (d *DAG) GetChildren(name string) []string {
	return d.children[name]
}

// Validate checks if the DAG is valid (no cycles, all deps exist)
func (d *DAG) Validate() error {
	// Check all dependencies exist
	for name, deps := range d.deps {
		for _, dep := range deps {
			if _, exists := d.tasks[dep]; !exists {
				return fmt.Errorf("task %s depends on unknown task %s", name, dep)
			}
		}
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(name string) (bool, string)
	hasCycle = func(name string) (bool, string) {
		visited[name] = true
		recStack[name] = true

		for _, dep := range d.deps[name] {
			if !visited[dep] {
				if found, cycle := hasCycle(dep); found {
					return true, cycle
				}
			} else if recStack[dep] {
				return true, fmt.Sprintf("%s -> %s", name, dep)
			}
		}

		recStack[name] = false
		return false, ""
	}

	for name := range d.tasks {
		if !visited[name] {
			if found, cycle := hasCycle(name); found {
				return cadreErrors.New(cadreErrors.CodeCyclicDependency,
					fmt.Sprintf("cycle detected involving task %s (%s)", name, cycle)).
					WithSuggestion("Remove or restructure the circular dependency in your task graph")
			}
		}
	}

	return nil
}

// TopologicalSort returns tasks in execution order
func (d *DAG) TopologicalSort() ([]*Task, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for name := range d.tasks {
		inDegree[name] = len(d.deps[name])
	}

	// Find tasks with no dependencies
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var result []*Task
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		task := d.tasks[name]
		result = append(result, task)

		// Reduce in-degree for children
		for _, child := range d.children[name] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if len(result) != len(d.tasks) {
		return nil, fmt.Errorf("could not sort all tasks - check for cycles")
	}

	return result, nil
}

// GetReady returns tasks that are ready to execute (all deps completed)
func (d *DAG) GetReady() []*Task {
	var ready []*Task

	for _, task := range d.tasks {
		if task.GetStatus() != "pending" {
			continue
		}

		allDepsComplete := true
		for _, dep := range d.deps[task.Name()] {
			depTask := d.tasks[dep]
			if depTask.GetStatus() != "completed" {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			ready = append(ready, task)
		}
	}

	return ready
}

// IsComplete returns true if all tasks are completed or failed
func (d *DAG) IsComplete() bool {
	for _, task := range d.tasks {
		s := task.GetStatus()
		if s == "pending" || s == "running" || s == "queued" {
			return false
		}
	}
	return true
}

// HasFailures returns true if any task failed
func (d *DAG) HasFailures() bool {
	for _, task := range d.tasks {
		if task.GetStatus() == "failed" {
			return true
		}
	}
	return false
}

// HasCycles returns true if the graph contains any cycles.
func (d *DAG) HasCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(name string) bool
	dfs = func(name string) bool {
		visited[name] = true
		recStack[name] = true
		for _, dep := range d.deps[name] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}
		recStack[name] = false
		return false
	}

	for name := range d.tasks {
		if !visited[name] {
			if dfs(name) {
				return true
			}
		}
	}
	return false
}

// ValidateDeps checks that all dependency references point to existing tasks.
// Unlike Validate(), it does not reject cycles.
func (d *DAG) ValidateDeps() error {
	for name, deps := range d.deps {
		for _, dep := range deps {
			if _, exists := d.tasks[dep]; !exists {
				return fmt.Errorf("task %s depends on unknown task %s", name, dep)
			}
		}
	}
	return nil
}

// Linearize returns tasks in an execution order that handles cycles.
// Back-edges (edges that form cycles) are skipped during DFS so that
// the result respects forward edges while breaking loops.
// If the graph has no cycles, it falls back to TopologicalSort.
func (d *DAG) Linearize() ([]*Task, error) {
	if !d.HasCycles() {
		return d.TopologicalSort()
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var order []*Task

	var dfs func(name string)
	dfs = func(name string) {
		color[name] = gray
		for _, dep := range d.deps[name] {
			if color[dep] == white {
				dfs(dep)
			}
			// gray = back-edge (cycle), skip
		}
		color[name] = black
		order = append(order, d.tasks[name])
	}

	// Visit all nodes
	for name := range d.tasks {
		if color[name] == white {
			dfs(name)
		}
	}

	// Reverse for correct execution order (dependencies before dependents)
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order, nil
}

// Reset resets all tasks to pending status
func (d *DAG) Reset() {
	for _, task := range d.tasks {
		task.mu.Lock()
		task.Status = "pending"
		task.Attempts = 0
		task.Error = nil
		task.Outputs = make(map[string]interface{})
		task.mu.Unlock()
	}
}
