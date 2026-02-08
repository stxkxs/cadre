package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricsExporter defines the interface for exporting metrics.
type MetricsExporter interface {
	// Export writes a metrics snapshot.
	Export(snapshot MetricsSnapshot) error
	// Close releases resources.
	Close() error
}

// MetricsSnapshot is a point-in-time metrics record.
type MetricsSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Event     string                 `json:"event"` // task.completed, crew.completed, etc.
	Metrics   map[string]interface{} `json:"metrics"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

// JSONFileExporter writes metrics as JSONL to a file.
type JSONFileExporter struct {
	mu   sync.Mutex
	file *os.File
}

// NewJSONFileExporter creates or appends to the given path.
func NewJSONFileExporter(path string) (*JSONFileExporter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metrics directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}

	return &JSONFileExporter{file: f}, nil
}

// Export writes a single snapshot as a JSON line.
func (e *JSONFileExporter) Export(snapshot MetricsSnapshot) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = e.file.Write(append(data, '\n'))
	return err
}

// Close closes the underlying file.
func (e *JSONFileExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.file.Close()
}
