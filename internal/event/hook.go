package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Hook processes lifecycle events.
type Hook interface {
	// Name returns the hook's identifier.
	Name() string
	// Matches returns true if the hook should handle this event type.
	Matches(t EventType) bool
	// IsBlocking returns true if execution should wait for this hook.
	IsBlocking() bool
	// Handle processes an event. For blocking hooks, an error stops execution.
	Handle(ev Event) error
}

// baseHook provides shared fields for all hook implementations.
type baseHook struct {
	name     string
	events   []EventType
	blocking bool
}

func (h *baseHook) Name() string      { return h.name }
func (h *baseHook) IsBlocking() bool   { return h.blocking }
func (h *baseHook) Matches(t EventType) bool {
	if len(h.events) == 0 {
		return true // match all events if no filter specified
	}
	for _, ev := range h.events {
		if ev == t {
			return true
		}
	}
	return false
}

// ShellHook executes a shell command with event data in environment variables.
//
// Environment variables set:
//   - CADRE_EVENT_TYPE: the event type string
//   - CADRE_EVENT_JSON: JSON-encoded event data
type ShellHook struct {
	baseHook
	Command string
}

func NewShellHook(name, command string, events []EventType, blocking bool) *ShellHook {
	return &ShellHook{
		baseHook: baseHook{name: name, events: events, blocking: blocking},
		Command:  command,
	}
}

func (h *ShellHook) Handle(ev Event) error {
	eventJSON, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	cmd := exec.Command("sh", "-c", h.Command)
	cmd.Env = append(os.Environ(),
		"CADRE_EVENT_TYPE="+string(ev.Type),
		"CADRE_EVENT_JSON="+string(eventJSON),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell hook %s failed: %w", h.name, err)
	}
	return nil
}

// WebhookHook sends an HTTP POST with event JSON to a URL.
type WebhookHook struct {
	baseHook
	URL     string
	Timeout time.Duration
}

func NewWebhookHook(name, url string, events []EventType, blocking bool) *WebhookHook {
	return &WebhookHook{
		baseHook: baseHook{name: name, events: events, blocking: blocking},
		URL:      url,
		Timeout:  10 * time.Second,
	}
}

func (h *WebhookHook) Handle(ev Event) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	client := &http.Client{Timeout: h.Timeout}
	resp, err := client.Post(h.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook %s failed: %w", h.name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook %s returned status %d", h.name, resp.StatusCode)
	}
	return nil
}

// LogHook logs events at the configured level. Always non-blocking.
type LogHook struct {
	baseHook
	logger Logger
	level  string // "debug", "info", "warn"
}

// FullLogger extends Logger with additional log levels for the LogHook.
type FullLogger interface {
	Logger
	Info(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
}

func NewLogHook(name string, events []EventType, logger Logger, level string) *LogHook {
	if level == "" {
		level = "info"
	}
	return &LogHook{
		baseHook: baseHook{name: name, events: events, blocking: false},
		logger:   logger,
		level:    level,
	}
}

func (h *LogHook) Handle(ev Event) error {
	msg := fmt.Sprintf("[event] %s", ev.Type)
	keyvals := make([]interface{}, 0, len(ev.Data)*2+2)
	keyvals = append(keyvals, "event_type", string(ev.Type))
	for k, v := range ev.Data {
		keyvals = append(keyvals, k, v)
	}

	if fl, ok := h.logger.(FullLogger); ok {
		switch h.level {
		case "debug":
			fl.Debug(msg, keyvals...)
		case "warn":
			fl.Warn(msg, keyvals...)
		default:
			fl.Info(msg, keyvals...)
		}
	} else {
		// Fallback: use Warn since Logger only guarantees Warn.
		h.logger.Warn(msg, keyvals...)
	}
	return nil
}

// PauseHook prints a message and waits for user to press Enter.
// Used for human-in-the-loop approval gates. Always blocking.
type PauseHook struct {
	baseHook
	Message string
	Reader  io.Reader // defaults to os.Stdin
}

func NewPauseHook(name string, events []EventType, message string) *PauseHook {
	return &PauseHook{
		baseHook: baseHook{name: name, events: events, blocking: true},
		Message:  message,
		Reader:   os.Stdin,
	}
}

func (h *PauseHook) Handle(ev Event) error {
	msg := h.Message
	if msg == "" {
		msg = fmt.Sprintf("Event %s occurred. Press Enter to continue...", ev.Type)
	}
	// Replace template variables in message.
	msg = strings.ReplaceAll(msg, "{{.EventType}}", string(ev.Type))
	if taskName, ok := ev.Data["task"].(string); ok {
		msg = strings.ReplaceAll(msg, "{{.Task}}", taskName)
	}

	fmt.Fprintln(os.Stderr, msg)

	reader := h.Reader
	if reader == nil {
		reader = os.Stdin
	}
	buf := make([]byte, 1)
	reader.Read(buf)
	return nil
}
