package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileReadTool reads file contents
type FileReadTool struct{}

// NewFileReadTool creates a new file read tool
func NewFileReadTool() *FileReadTool {
	return &FileReadTool{}
}

// Name returns the tool name
func (t *FileReadTool) Name() string {
	return "file_read"
}

// Description returns the tool description
func (t *FileReadTool) Description() string {
	return "Read the contents of a file. Returns the file content as a string."
}

// Parameters returns the JSON schema for the tool parameters
func (t *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "The path to the file to read",
		},
		"start_line": map[string]interface{}{
			"type":        "integer",
			"description": "Optional: Start reading from this line number (1-based)",
		},
		"end_line": map[string]interface{}{
			"type":        "integer",
			"description": "Optional: Stop reading at this line number (inclusive)",
		},
	}
}

// FileReadArgs represents the arguments for file_read
type FileReadArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

// Execute reads a file
func (t *FileReadTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args FileReadArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path
	path := args.Path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}

	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", args.Path)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", args.Path)
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Handle line range if specified
	if args.StartLine > 0 || args.EndLine > 0 {
		lines := strings.Split(string(content), "\n")
		start := 0
		end := len(lines)

		if args.StartLine > 0 {
			start = args.StartLine - 1
			if start >= len(lines) {
				start = len(lines)
			}
		}

		if args.EndLine > 0 && args.EndLine <= len(lines) {
			end = args.EndLine
		}

		if start < end {
			content = []byte(strings.Join(lines[start:end], "\n"))
		} else {
			content = []byte{}
		}
	}

	return string(content), nil
}

// Test verifies the tool works
func (t *FileReadTool) Test(ctx context.Context) (string, error) {
	// Test by reading current directory
	entries, err := os.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("failed to read current directory: %w", err)
	}

	return fmt.Sprintf("file_read tool operational, current directory has %d entries", len(entries)), nil
}

// FileWriteTool writes file contents
type FileWriteTool struct{}

// NewFileWriteTool creates a new file write tool
func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{}
}

// Name returns the tool name
func (t *FileWriteTool) Name() string {
	return "file_write"
}

// Description returns the tool description
func (t *FileWriteTool) Description() string {
	return "Write content to a file. Creates the file if it doesn't exist, or overwrites if it does."
}

// Parameters returns the JSON schema for the tool parameters
func (t *FileWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "The path to the file to write",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "The content to write to the file",
		},
		"append": map[string]interface{}{
			"type":        "boolean",
			"description": "If true, append to the file instead of overwriting",
		},
	}
}

// FileWriteArgs represents the arguments for file_write
type FileWriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append,omitempty"`
}

// Execute writes to a file
func (t *FileWriteTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args FileWriteArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path
	path := args.Path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	var err error
	if args.Append {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()
		_, err = f.WriteString(args.Content)
	} else {
		err = os.WriteFile(path, []byte(args.Content), 0644)
	}

	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(args.Content), args.Path), nil
}

// Test verifies the tool works
func (t *FileWriteTool) Test(ctx context.Context) (string, error) {
	// Test write permissions in temp directory
	tmpFile := filepath.Join(os.TempDir(), "crew_test_write")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		return "", fmt.Errorf("write test failed: %w", err)
	}
	os.Remove(tmpFile)
	return "file_write tool operational", nil
}
