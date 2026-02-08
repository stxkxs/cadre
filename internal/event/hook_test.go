package event

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestShellHook_Matches(t *testing.T) {
	hook := NewShellHook("test", "echo hi", []EventType{TaskStarted, TaskCompleted}, false)

	if !hook.Matches(TaskStarted) {
		t.Error("should match TaskStarted")
	}
	if !hook.Matches(TaskCompleted) {
		t.Error("should match TaskCompleted")
	}
	if hook.Matches(CrewStarted) {
		t.Error("should not match CrewStarted")
	}
}

func TestShellHook_Execute(t *testing.T) {
	hook := NewShellHook("test", "true", []EventType{TaskStarted}, false)

	ev := NewEvent(TaskStarted, map[string]interface{}{"task": "a"})
	err := hook.Handle(ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShellHook_Failure(t *testing.T) {
	hook := NewShellHook("test", "false", []EventType{TaskStarted}, true)

	ev := NewEvent(TaskStarted, nil)
	err := hook.Handle(ev)
	if err == nil {
		t.Fatal("expected error from failed shell command")
	}
}

func TestWebhookHook_Execute(t *testing.T) {
	var received struct {
		mu   sync.Mutex
		body []byte
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received.mu.Lock()
		received.body = body
		received.mu.Unlock()
		w.WriteHeader(200)
	}))
	defer server.Close()

	hook := NewWebhookHook("test", server.URL, []EventType{CrewCompleted}, true)
	ev := NewEvent(CrewCompleted, map[string]interface{}{"crew": "test-crew"})
	err := hook.Handle(ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	received.mu.Lock()
	defer received.mu.Unlock()

	var payload Event
	if err := json.Unmarshal(received.body, &payload); err != nil {
		t.Fatalf("failed to parse webhook payload: %v", err)
	}
	if payload.Type != CrewCompleted {
		t.Errorf("expected CrewCompleted, got %s", payload.Type)
	}
}

func TestWebhookHook_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	hook := NewWebhookHook("test", server.URL, []EventType{CrewFailed}, true)
	err := hook.Handle(NewEvent(CrewFailed, nil))
	if err == nil {
		t.Fatal("expected error from 500 status")
	}
}

func TestLogHook_Execute(t *testing.T) {
	logger := &testLogger{}
	hook := NewLogHook("test", []EventType{TaskStarted}, logger, "info")

	ev := NewEvent(TaskStarted, map[string]interface{}{"task": "a"})
	err := hook.Handle(ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LogHook with a FullLogger calls Info; testLogger implements FullLogger
	// so the warn path won't be used here.
}

func TestLogHook_AlwaysNonBlocking(t *testing.T) {
	hook := NewLogHook("test", nil, &testLogger{}, "debug")
	if hook.IsBlocking() {
		t.Error("log hook should always be non-blocking")
	}
}

func TestPauseHook_Execute(t *testing.T) {
	// Simulate user pressing Enter via a bytes.Buffer.
	reader := bytes.NewReader([]byte("\n"))
	hook := NewPauseHook("approve", []EventType{TaskStarted}, "Continue?")
	hook.Reader = reader

	ev := NewEvent(TaskStarted, map[string]interface{}{"task": "deploy"})
	err := hook.Handle(ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPauseHook_AlwaysBlocking(t *testing.T) {
	hook := NewPauseHook("test", nil, "")
	if !hook.IsBlocking() {
		t.Error("pause hook should always be blocking")
	}
}

func TestBaseHook_MatchesAll(t *testing.T) {
	h := &baseHook{name: "all", events: nil}
	if !h.Matches(TaskStarted) {
		t.Error("nil events should match everything")
	}
	if !h.Matches(CrewFailed) {
		t.Error("nil events should match everything")
	}
}

func TestBaseHook_MatchesNone(t *testing.T) {
	h := &baseHook{name: "specific", events: []EventType{CrewStarted}}
	if h.Matches(TaskStarted) {
		t.Error("should not match TaskStarted")
	}
}
