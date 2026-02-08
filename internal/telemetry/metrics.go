package telemetry

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects runtime metrics
type Metrics struct {
	mu sync.RWMutex

	// Counters
	TasksStarted   int64
	TasksCompleted int64
	TasksFailed    int64
	ToolCalls      int64
	APIRequests    int64

	// Gauges
	ActiveTasks  int64
	ActiveAgents int64

	// Histograms (simplified)
	taskDurations []time.Duration
	apiLatencies  []time.Duration

	// Exporter (optional)
	exporter MetricsExporter
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		taskDurations: make([]time.Duration, 0, 1000),
		apiLatencies:  make([]time.Duration, 0, 1000),
	}
}

// IncTasksStarted increments the tasks started counter
func (m *Metrics) IncTasksStarted() {
	atomic.AddInt64(&m.TasksStarted, 1)
	atomic.AddInt64(&m.ActiveTasks, 1)
}

// IncTasksCompleted increments the tasks completed counter
func (m *Metrics) IncTasksCompleted() {
	atomic.AddInt64(&m.TasksCompleted, 1)
	atomic.AddInt64(&m.ActiveTasks, -1)
}

// IncTasksFailed increments the tasks failed counter
func (m *Metrics) IncTasksFailed() {
	atomic.AddInt64(&m.TasksFailed, 1)
	atomic.AddInt64(&m.ActiveTasks, -1)
}

// IncToolCalls increments the tool calls counter
func (m *Metrics) IncToolCalls() {
	atomic.AddInt64(&m.ToolCalls, 1)
}

// IncAPIRequests increments the API requests counter
func (m *Metrics) IncAPIRequests() {
	atomic.AddInt64(&m.APIRequests, 1)
}

// RecordTaskDuration records a task duration
func (m *Metrics) RecordTaskDuration(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskDurations = append(m.taskDurations, d)
}

// RecordAPILatency records an API call latency
func (m *Metrics) RecordAPILatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiLatencies = append(m.apiLatencies, d)
}

// GetSummary returns a summary of collected metrics
func (m *Metrics) GetSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := map[string]interface{}{
		"tasks_started":   atomic.LoadInt64(&m.TasksStarted),
		"tasks_completed": atomic.LoadInt64(&m.TasksCompleted),
		"tasks_failed":    atomic.LoadInt64(&m.TasksFailed),
		"tool_calls":      atomic.LoadInt64(&m.ToolCalls),
		"api_requests":    atomic.LoadInt64(&m.APIRequests),
		"active_tasks":    atomic.LoadInt64(&m.ActiveTasks),
		"active_agents":   atomic.LoadInt64(&m.ActiveAgents),
	}

	// Add duration stats
	if len(m.taskDurations) > 0 {
		var total time.Duration
		for _, d := range m.taskDurations {
			total += d
		}
		summary["avg_task_duration_ms"] = total.Milliseconds() / int64(len(m.taskDurations))
	}

	if len(m.apiLatencies) > 0 {
		var total time.Duration
		for _, d := range m.apiLatencies {
			total += d
		}
		summary["avg_api_latency_ms"] = total.Milliseconds() / int64(len(m.apiLatencies))
	}

	return summary
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.TasksStarted, 0)
	atomic.StoreInt64(&m.TasksCompleted, 0)
	atomic.StoreInt64(&m.TasksFailed, 0)
	atomic.StoreInt64(&m.ToolCalls, 0)
	atomic.StoreInt64(&m.APIRequests, 0)
	atomic.StoreInt64(&m.ActiveTasks, 0)
	atomic.StoreInt64(&m.ActiveAgents, 0)

	m.taskDurations = m.taskDurations[:0]
	m.apiLatencies = m.apiLatencies[:0]
}

// SetExporter attaches a metrics exporter.
func (m *Metrics) SetExporter(e MetricsExporter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exporter = e
}

// Flush exports the current metrics snapshot with the given event label.
func (m *Metrics) Flush(event string, labels map[string]string) {
	m.mu.RLock()
	exporter := m.exporter
	m.mu.RUnlock()

	if exporter == nil {
		return
	}

	snapshot := MetricsSnapshot{
		Timestamp: time.Now(),
		Event:     event,
		Metrics:   m.GetSummary(),
		Labels:    labels,
	}
	// Best-effort export.
	_ = exporter.Export(snapshot)
}
