package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type traceKey struct{}

// TraceContext carries correlation IDs through the execution pipeline.
type TraceContext struct {
	RunID     string `json:"run_id"`
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
	ParentID  string `json:"parent_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	TaskName  string `json:"task_name,omitempty"`
}

// NewTraceContext creates a root trace context with a fresh TraceID and SpanID.
func NewTraceContext(runID string) *TraceContext {
	return &TraceContext{
		RunID:   runID,
		TraceID: randomID(),
		SpanID:  randomID(),
	}
}

// ChildSpan creates a child trace context inheriting the TraceID and RunID.
func (tc *TraceContext) ChildSpan() *TraceContext {
	return &TraceContext{
		RunID:    tc.RunID,
		TraceID:  tc.TraceID,
		SpanID:   randomID(),
		ParentID: tc.SpanID,
	}
}

// WithAgent returns a copy with the AgentName set.
func (tc *TraceContext) WithAgent(name string) *TraceContext {
	child := *tc
	child.AgentName = name
	return &child
}

// WithTask returns a copy with the TaskName set.
func (tc *TraceContext) WithTask(name string) *TraceContext {
	child := *tc
	child.TaskName = name
	return &child
}

// Fields returns key-value pairs suitable for structured logging.
func (tc *TraceContext) Fields() map[string]interface{} {
	fields := map[string]interface{}{
		"run_id":   tc.RunID,
		"trace_id": tc.TraceID,
		"span_id":  tc.SpanID,
	}
	if tc.ParentID != "" {
		fields["parent_id"] = tc.ParentID
	}
	if tc.AgentName != "" {
		fields["agent"] = tc.AgentName
	}
	if tc.TaskName != "" {
		fields["task"] = tc.TaskName
	}
	return fields
}

// ContextWithTrace stores a TraceContext in the context.
func ContextWithTrace(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceKey{}, tc)
}

// TraceFromContext extracts a TraceContext from the context, or nil.
func TraceFromContext(ctx context.Context) *TraceContext {
	tc, _ := ctx.Value(traceKey{}).(*TraceContext)
	return tc
}

// WithTrace returns a logger enriched with trace fields from the context.
func (l *Logger) WithTrace(ctx context.Context) *Logger {
	tc := TraceFromContext(ctx)
	if tc == nil {
		return l
	}
	return l.WithFields(tc.Fields())
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
