package telemetry

import (
	"context"
	"testing"
)

func TestTraceContext_NewAndChild(t *testing.T) {
	root := NewTraceContext("run-123")

	if root.RunID != "run-123" {
		t.Errorf("expected RunID 'run-123', got %q", root.RunID)
	}
	if root.TraceID == "" {
		t.Error("expected non-empty TraceID")
	}
	if root.SpanID == "" {
		t.Error("expected non-empty SpanID")
	}
	if root.ParentID != "" {
		t.Error("expected empty ParentID for root")
	}

	child := root.ChildSpan()
	if child.TraceID != root.TraceID {
		t.Error("child should inherit TraceID")
	}
	if child.ParentID != root.SpanID {
		t.Error("child ParentID should be parent's SpanID")
	}
	if child.SpanID == root.SpanID {
		t.Error("child should have a different SpanID")
	}
}

func TestTraceContext_WithAgentTask(t *testing.T) {
	tc := NewTraceContext("run-1")
	withAgent := tc.WithAgent("developer")
	withTask := withAgent.WithTask("implement")

	if withAgent.AgentName != "developer" {
		t.Errorf("expected agent 'developer', got %q", withAgent.AgentName)
	}
	if withTask.TaskName != "implement" {
		t.Errorf("expected task 'implement', got %q", withTask.TaskName)
	}
	// Original unchanged
	if tc.AgentName != "" {
		t.Error("original should not be modified")
	}
}

func TestTraceContext_ContextPropagation(t *testing.T) {
	tc := NewTraceContext("run-2")
	ctx := ContextWithTrace(context.Background(), tc)

	extracted := TraceFromContext(ctx)
	if extracted == nil {
		t.Fatal("expected trace in context")
	}
	if extracted.RunID != "run-2" {
		t.Errorf("expected RunID 'run-2', got %q", extracted.RunID)
	}

	// nil context returns nil
	if TraceFromContext(context.Background()) != nil {
		t.Error("expected nil trace from empty context")
	}
}

func TestTraceContext_Fields(t *testing.T) {
	tc := NewTraceContext("run-3")
	tc = tc.WithAgent("dev").WithTask("build")

	fields := tc.Fields()
	if fields["run_id"] != "run-3" {
		t.Error("expected run_id in fields")
	}
	if fields["agent"] != "dev" {
		t.Error("expected agent in fields")
	}
	if fields["task"] != "build" {
		t.Error("expected task in fields")
	}
}

func TestLogger_WithTrace(t *testing.T) {
	logger := NewLogger(true)
	tc := NewTraceContext("run-4")
	ctx := ContextWithTrace(context.Background(), tc)

	traced := logger.WithTrace(ctx)
	if traced == nil {
		t.Fatal("expected non-nil logger")
	}

	// Should not panic with nil trace
	noTrace := logger.WithTrace(context.Background())
	if noTrace == nil {
		t.Fatal("expected non-nil logger even without trace")
	}
}
