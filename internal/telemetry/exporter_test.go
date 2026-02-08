package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONFileExporter_Export(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cadre", "metrics.jsonl")

	exporter, err := NewJSONFileExporter(path)
	if err != nil {
		t.Fatal(err)
	}
	defer exporter.Close()

	snapshot := MetricsSnapshot{
		Timestamp: time.Now(),
		Event:     "task.completed",
		Metrics: map[string]interface{}{
			"tasks_completed": int64(5),
			"tool_calls":      int64(12),
		},
		Labels: map[string]string{
			"crew": "development",
			"task": "implement",
		},
	}

	if err := exporter.Export(snapshot); err != nil {
		t.Fatal(err)
	}

	// Write another snapshot
	snapshot.Event = "crew.completed"
	if err := exporter.Export(snapshot); err != nil {
		t.Fatal(err)
	}

	exporter.Close()

	// Read and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}

	var parsed MetricsSnapshot
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Event != "task.completed" {
		t.Errorf("expected event 'task.completed', got %q", parsed.Event)
	}
}

func TestMetrics_FlushWithExporter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.jsonl")

	exporter, err := NewJSONFileExporter(path)
	if err != nil {
		t.Fatal(err)
	}

	m := NewMetrics()
	m.SetExporter(exporter)
	m.IncTasksCompleted()

	m.Flush("task.completed", map[string]string{"task": "test"})
	exporter.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty metrics file")
	}

	var snapshot MetricsSnapshot
	if err := json.Unmarshal(data[:len(data)-1], &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Event != "task.completed" {
		t.Errorf("expected event 'task.completed', got %q", snapshot.Event)
	}
}

func TestMetrics_FlushWithoutExporter(t *testing.T) {
	m := NewMetrics()
	// Should not panic
	m.Flush("test", nil)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
