package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool calls an HTTP endpoint as a tool.
type HTTPTool struct {
	name        string
	description string
	url         string
	method      string
	headers     map[string]string
	timeout     time.Duration
}

// NewHTTPTool creates a new HTTP tool.
func NewHTTPTool(name, description, url, method string, headers map[string]string) *HTTPTool {
	if method == "" {
		method = "POST"
	}
	return &HTTPTool{
		name:        name,
		description: description,
		url:         url,
		method:      strings.ToUpper(method),
		headers:     headers,
		timeout:     30 * time.Second,
	}
}

func (t *HTTPTool) Name() string        { return t.name }
func (t *HTTPTool) Description() string  { return t.description }

func (t *HTTPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"input": map[string]interface{}{
			"type":        "string",
			"description": "Input data sent as the request body (JSON string)",
		},
	}
}

type httpArgs struct {
	Input string `json:"input"`
}

func (t *HTTPTool) Execute(ctx context.Context, argsJSON json.RawMessage) (string, error) {
	var args httpArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Substitute {{input}} in URL
	url := strings.ReplaceAll(t.url, "{{input}}", args.Input)

	var body io.Reader
	if t.method == "POST" || t.method == "PUT" || t.method == "PATCH" {
		body = bytes.NewBufferString(args.Input)
	}

	req, err := http.NewRequestWithContext(ctx, t.method, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: t.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return string(respBody), fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

func (t *HTTPTool) Test(ctx context.Context) (string, error) {
	return fmt.Sprintf("http tool %q configured for %s %s", t.name, t.method, t.url), nil
}

// SetTimeout sets the HTTP request timeout.
func (t *HTTPTool) SetTimeout(d time.Duration) {
	t.timeout = d
}
