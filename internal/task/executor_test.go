package task

import (
	"strings"
	"testing"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/testutil"
)

func TestParseOutputs_JSONBlock(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := `Here is my analysis.

` + "```json" + `
{
  "summary": "Added login feature",
  "files_changed": ["auth.go", "main.go"]
}
` + "```"

	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
		{Name: "files_changed", Type: "string[]"},
	}

	outputs := e.parseOutputs(response, defs)

	if outputs["summary"] != "Added login feature" {
		t.Errorf("expected 'Added login feature', got %v", outputs["summary"])
	}

	files, ok := outputs["files_changed"].([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", outputs["files_changed"])
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	if outputs["_response"] != response {
		t.Error("expected raw response in _response")
	}
}

func TestParseOutputs_NoDefinitions(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	outputs := e.parseOutputs("some response", nil)

	if outputs["_response"] != "some response" {
		t.Error("expected raw response in _response")
	}
	if len(outputs) != 1 {
		t.Errorf("expected 1 output (_response only), got %d", len(outputs))
	}
}

func TestParseOutputs_NoJSONFallback(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := "I completed the task. The summary is: everything worked."
	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
	}

	outputs := e.parseOutputs(response, defs)

	// Fallback: raw response stored as each output
	if outputs["summary"] != response {
		t.Errorf("expected fallback to raw response, got %v", outputs["summary"])
	}
	if outputs["_response"] != response {
		t.Error("expected raw response in _response")
	}
}

func TestParseOutputs_MalformedJSON(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := "Here is the output:\n```json\n{invalid json}\n```"
	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
	}

	outputs := e.parseOutputs(response, defs)

	// Fallback to raw response
	if outputs["summary"] != response {
		t.Errorf("expected fallback to raw response for malformed JSON")
	}
}

func TestParseOutputs_MissingFields(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := "```json\n{\"summary\": \"done\"}\n```"
	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
		{Name: "files_changed", Type: "string[]"},
	}

	outputs := e.parseOutputs(response, defs)

	if outputs["summary"] != "done" {
		t.Errorf("expected 'done', got %v", outputs["summary"])
	}
	// Missing field should not be in outputs
	if _, ok := outputs["files_changed"]; ok {
		t.Error("expected files_changed to be absent for missing field")
	}
}

func TestParseOutputs_MultipleJSONBlocks(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := "First attempt:\n```json\n{\"summary\": \"wrong\"}\n```\n\nCorrected:\n```json\n{\"summary\": \"correct\"}\n```"
	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
	}

	outputs := e.parseOutputs(response, defs)

	// Should use the LAST JSON block
	if outputs["summary"] != "correct" {
		t.Errorf("expected 'correct' (last block), got %v", outputs["summary"])
	}
}

func TestParseOutputs_RawJSONNoFence(t *testing.T) {
	e := NewExecutor(testutil.TestConfig(), testutil.TestLogger())

	response := "Here is the result:\n{\"summary\": \"from raw json\"}"
	defs := []config.OutputConfig{
		{Name: "summary", Type: "string"},
	}

	outputs := e.parseOutputs(response, defs)

	if outputs["summary"] != "from raw json" {
		t.Errorf("expected 'from raw json', got %v", outputs["summary"])
	}
}

func TestExtractLastJSONBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "```json\n{\"a\": 1}\n```",
			want:  `{"a": 1}`,
		},
		{
			name:  "with surrounding text",
			input: "text before\n```json\n{\"a\": 1}\n```\ntext after",
			want:  `{"a": 1}`,
		},
		{
			name:  "multiple blocks",
			input: "```json\n{\"a\": 1}\n```\n```json\n{\"b\": 2}\n```",
			want:  `{"b": 2}`,
		},
		{
			name:  "no block",
			input: "no json here",
			want:  "",
		},
		{
			name:  "unclosed block",
			input: "```json\n{\"a\": 1}",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLastJSONBlock(tt.input)
			if got != tt.want {
				t.Errorf("extractLastJSONBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRawJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing json",
			input: "text {\"a\": 1}",
			want:  `{"a": 1}`,
		},
		{
			name:  "no json",
			input: "no json here",
			want:  "",
		},
		{
			name:  "nested braces",
			input: `text {"a": {"b": 1}}`,
			want:  `{"a": {"b": 1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRawJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractRawJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPrompt_WithOutputs(t *testing.T) {
	cfg := &config.TaskConfig{
		Name:        "test-task",
		Description: "Test description",
		Agent:       "developer",
		Outputs: []config.OutputConfig{
			{Name: "summary", Type: "string"},
			{Name: "files_changed", Type: "string[]"},
		},
	}
	task := NewTask(cfg)
	prompt := task.BuildPrompt()

	if !strings.Contains(prompt, "```json") {
		t.Error("expected JSON code block in prompt with outputs defined")
	}
	if !strings.Contains(prompt, `"summary"`) {
		t.Error("expected summary field in JSON block")
	}
	if !strings.Contains(prompt, `"files_changed"`) {
		t.Error("expected files_changed field in JSON block")
	}
	if !strings.Contains(prompt, "You MUST include a JSON code block") {
		t.Error("expected JSON instruction in prompt")
	}
}

func TestBuildPrompt_WithoutOutputs(t *testing.T) {
	cfg := &config.TaskConfig{
		Name:        "test-task",
		Description: "Test description",
		Agent:       "developer",
	}
	task := NewTask(cfg)
	prompt := task.BuildPrompt()

	if strings.Contains(prompt, "```json") {
		t.Error("expected no JSON code block when no outputs defined")
	}
	if !strings.Contains(prompt, "Please complete this task") {
		t.Error("expected default instruction")
	}
}

func TestBuildPrompt_WithManagerInstructions(t *testing.T) {
	cfg := &config.TaskConfig{
		Name:        "test-task",
		Description: "Test description",
		Agent:       "developer",
	}
	task := NewTask(cfg)
	task.SetInput("_manager_instructions", "Focus on error handling")
	prompt := task.BuildPrompt()

	if !strings.Contains(prompt, "### Manager Instructions:") {
		t.Error("expected manager instructions section")
	}
	if !strings.Contains(prompt, "Focus on error handling") {
		t.Error("expected manager instructions content")
	}
}
