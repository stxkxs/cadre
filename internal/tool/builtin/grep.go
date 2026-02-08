package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool searches for patterns in files
type GrepTool struct{}

// NewGrepTool creates a new grep tool
func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

// Name returns the tool name
func (t *GrepTool) Name() string {
	return "grep"
}

// Description returns the tool description
func (t *GrepTool) Description() string {
	return "Search for a pattern in files. Returns matching lines with file names and line numbers."
}

// Parameters returns the JSON schema for the tool parameters
func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"pattern": map[string]interface{}{
			"type":        "string",
			"description": "The regular expression pattern to search for",
		},
		"path": map[string]interface{}{
			"type":        "string",
			"description": "File or directory path to search in",
		},
		"recursive": map[string]interface{}{
			"type":        "boolean",
			"description": "If true and path is a directory, search recursively",
		},
		"ignore_case": map[string]interface{}{
			"type":        "boolean",
			"description": "If true, perform case-insensitive matching",
		},
		"max_results": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum number of results to return (default: 100)",
		},
		"include": map[string]interface{}{
			"type":        "string",
			"description": "Only search files matching this glob pattern (e.g., '*.go')",
		},
		"exclude": map[string]interface{}{
			"type":        "string",
			"description": "Skip files matching this glob pattern (e.g., '*_test.go')",
		},
	}
}

// GrepArgs represents the arguments for grep
type GrepArgs struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	Recursive  bool   `json:"recursive,omitempty"`
	IgnoreCase bool   `json:"ignore_case,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
	Include    string `json:"include,omitempty"`
	Exclude    string `json:"exclude,omitempty"`
}

// GrepResult represents a search result
type GrepResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// Execute searches for a pattern
func (t *GrepTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args GrepArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	if args.Path == "" {
		args.Path = "."
	}

	if args.MaxResults == 0 {
		args.MaxResults = 100
	}

	// Compile regex
	flags := ""
	if args.IgnoreCase {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + args.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Resolve path
	path := args.Path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path not found: %s", args.Path)
	}

	var results []GrepResult

	// Search function
	searchFile := func(filePath string) error {
		// Check include/exclude patterns
		if args.Include != "" {
			matched, _ := filepath.Match(args.Include, filepath.Base(filePath))
			if !matched {
				return nil
			}
		}
		if args.Exclude != "" {
			matched, _ := filepath.Match(args.Exclude, filepath.Base(filePath))
			if matched {
				return nil
			}
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil // Skip unreadable files
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if len(results) >= args.MaxResults {
				break
			}

			if re.MatchString(line) {
				// Get relative path
				relPath, err := filepath.Rel(args.Path, filePath)
				if err != nil {
					relPath = filePath
				}

				results = append(results, GrepResult{
					File:    relPath,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
			}
		}

		return nil
	}

	if info.IsDir() {
		if args.Recursive {
			filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				// Skip hidden files and common non-text files
				base := filepath.Base(filePath)
				if strings.HasPrefix(base, ".") {
					return nil
				}
				return searchFile(filePath)
			})
		} else {
			entries, err := os.ReadDir(path)
			if err != nil {
				return "", fmt.Errorf("failed to read directory: %w", err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				searchFile(filepath.Join(path, entry.Name()))
			}
		}
	} else {
		searchFile(path)
	}

	// Format results
	if len(results) == 0 {
		return "No matches found.", nil
	}

	var output strings.Builder
	for _, r := range results {
		output.WriteString(fmt.Sprintf("%s:%d: %s\n", r.File, r.Line, r.Content))
	}

	if len(results) >= args.MaxResults {
		output.WriteString(fmt.Sprintf("\n... (limited to %d results)", args.MaxResults))
	}

	return output.String(), nil
}

// Test verifies the tool works
func (t *GrepTool) Test(ctx context.Context) (string, error) {
	// Create a temp file and search it
	tmpFile := filepath.Join(os.TempDir(), "crew_test_grep.txt")
	os.WriteFile(tmpFile, []byte("test line one\ntest line two\nno match here"), 0644)
	defer os.Remove(tmpFile)

	result, err := t.Execute(ctx, []byte(fmt.Sprintf(`{"pattern": "test", "path": "%s"}`, tmpFile)))
	if err != nil {
		return "", err
	}

	if !strings.Contains(result, "test line") {
		return "", fmt.Errorf("search failed: %s", result)
	}

	return "grep tool operational", nil
}
