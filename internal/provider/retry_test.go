package provider

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// testProvider is a minimal mock for retry tests.
type testProvider struct {
	responses []*Response
	errors    []error
	calls     int
}

func (p *testProvider) Name() string { return "test" }

func (p *testProvider) Complete(ctx context.Context, req *CompletionRequest) (*Response, error) {
	idx := p.calls
	p.calls++
	if idx < len(p.errors) && p.errors[idx] != nil {
		return nil, p.errors[idx]
	}
	if idx < len(p.responses) {
		return p.responses[idx], nil
	}
	return &Response{Content: "default", StopReason: "end_turn"}, nil
}

func (p *testProvider) Stream(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return err
	}
	handler(StreamEvent{Type: "content", Content: resp.Content, Done: true})
	return nil
}

// streamTestProvider allows custom Stream implementations for retry tests.
type streamTestProvider struct {
	testProvider
	streamFn func(ctx context.Context, req *CompletionRequest, handler StreamHandler) error
}

func (p *streamTestProvider) Stream(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
	return p.streamFn(ctx, req, handler)
}

func fastRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		JitterFraction: 0,
	}
}

func TestRetryProvider_SuccessFirstTry(t *testing.T) {
	inner := &testProvider{
		responses: []*Response{{Content: "ok", StopReason: "end_turn"}},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	resp, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Content)
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 call, got %d", inner.calls)
	}
}

func TestRetryProvider_RetryOn500(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("API error (status 500): internal server error"),
			fmt.Errorf("API error (status 500): internal server error"),
			nil,
		},
		responses: []*Response{nil, nil, {Content: "recovered", StopReason: "end_turn"}},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	resp, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "recovered" {
		t.Errorf("expected 'recovered', got %q", resp.Content)
	}
	if inner.calls != 3 {
		t.Errorf("expected 3 calls, got %d", inner.calls)
	}
}

func TestRetryProvider_RetryOn429(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("API error (status 429): rate limited"),
			nil,
		},
		responses: []*Response{nil, {Content: "ok", StopReason: "end_turn"}},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	resp, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Content)
	}
	if inner.calls != 2 {
		t.Errorf("expected 2 calls, got %d", inner.calls)
	}
}

func TestRetryProvider_NoRetryOn400(t *testing.T) {
	inner := &testProvider{
		errors: []error{fmt.Errorf("API error (status 400): bad request")},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	_, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", inner.calls)
	}
}

func TestRetryProvider_NoRetryOn401(t *testing.T) {
	inner := &testProvider{
		errors: []error{fmt.Errorf("API error (status 401): unauthorized")},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	_, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", inner.calls)
	}
}

func TestRetryProvider_MaxRetriesExhausted(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("API error (status 503): unavailable"),
			fmt.Errorf("API error (status 503): unavailable"),
			fmt.Errorf("API error (status 503): unavailable"),
			fmt.Errorf("API error (status 503): unavailable"),
		},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	_, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	// 1 initial + 3 retries = 4
	if inner.calls != 4 {
		t.Errorf("expected 4 calls, got %d", inner.calls)
	}
}

func TestRetryProvider_ContextCancelledDuringBackoff(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("API error (status 500): error"),
			fmt.Errorf("API error (status 500): error"),
		},
	}
	cfg := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 5 * time.Second, // long backoff
		MaxBackoff:     10 * time.Second,
		JitterFraction: 0,
	}
	rp := NewRetryProvider(inner, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := rp.Complete(ctx, &CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != context.DeadlineExceeded {
		t.Logf("got error: %v (this is acceptable)", err)
	}
}

func TestRetryProvider_NetworkError(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("request failed: connection refused"),
			nil,
		},
		responses: []*Response{nil, {Content: "ok", StopReason: "end_turn"}},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	resp, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Content)
	}
	if inner.calls != 2 {
		t.Errorf("expected 2 calls, got %d", inner.calls)
	}
}

func TestRetryProvider_StreamRetry(t *testing.T) {
	inner := &testProvider{
		errors: []error{
			fmt.Errorf("API error (status 502): bad gateway"),
			nil,
		},
		responses: []*Response{nil, {Content: "streamed", StopReason: "end_turn"}},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	var content string
	err := rp.Stream(context.Background(), &CompletionRequest{}, func(ev StreamEvent) {
		content = ev.Content
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "streamed" {
		t.Errorf("expected 'streamed', got %q", content)
	}
}

func TestRetryProvider_StreamRetry_NoDoubleEmit(t *testing.T) {
	// First call fails (retryable) — events stream directly to handler.
	// Second call succeeds — events are buffered then flushed.
	// Verify the handler sees first attempt's partial events + second attempt's success events.
	callCount := 0
	inner := &streamTestProvider{
		streamFn: func(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
			callCount++
			if callCount == 1 {
				// First attempt: emit some events, then fail.
				handler(StreamEvent{Type: "text", Content: "partial"})
				handler(StreamEvent{Type: "text", Content: " data"})
				return fmt.Errorf("API error (status 502): bad gateway")
			}
			// Second attempt (retry): success.
			handler(StreamEvent{Type: "text", Content: "good"})
			handler(StreamEvent{Type: "done", Done: true, StopReason: "end_turn"})
			return nil
		},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	var events []StreamEvent
	err := rp.Stream(context.Background(), &CompletionRequest{}, func(ev StreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First attempt streams directly (partial, data), retry buffers (good, done).
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d: %+v", len(events), events)
	}
	// First attempt's partial events streamed directly.
	if events[0].Content != "partial" {
		t.Errorf("expected first event 'partial', got %q", events[0].Content)
	}
	if events[1].Content != " data" {
		t.Errorf("expected second event ' data', got %q", events[1].Content)
	}
	// Retry's buffered events flushed on success.
	if events[2].Content != "good" {
		t.Errorf("expected third event 'good', got %q", events[2].Content)
	}
	if !events[3].Done {
		t.Error("expected fourth event to be done")
	}

	if callCount != 2 {
		t.Errorf("expected 2 stream calls, got %d", callCount)
	}
}

func TestRetryProvider_StreamFirstAttemptRealTime(t *testing.T) {
	// Verify first attempt calls handler directly (events arrive immediately, not batched).
	var order []string
	inner := &streamTestProvider{
		streamFn: func(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
			handler(StreamEvent{Type: "text", Content: "a"})
			order = append(order, "after-a")
			handler(StreamEvent{Type: "text", Content: "b"})
			order = append(order, "after-b")
			handler(StreamEvent{Type: "done", Done: true, StopReason: "end_turn"})
			return nil
		},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	var events []StreamEvent
	err := rp.Stream(context.Background(), &CompletionRequest{}, func(ev StreamEvent) {
		events = append(events, ev)
		if ev.Content == "a" {
			order = append(order, "handler-a")
		}
		if ev.Content == "b" {
			order = append(order, "handler-b")
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Handler should be called synchronously during Stream (real-time).
	// order should be: handler-a, after-a, handler-b, after-b
	expected := []string{"handler-a", "after-a", "handler-b", "after-b"}
	if len(order) != len(expected) {
		t.Fatalf("expected order %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestRetryProvider_StreamFirstFailRetrySuccess(t *testing.T) {
	// First attempt streams partial + fails, retry buffers + flushes success only.
	callCount := 0
	inner := &streamTestProvider{
		streamFn: func(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
			callCount++
			if callCount == 1 {
				handler(StreamEvent{Type: "text", Content: "partial-from-first"})
				return fmt.Errorf("API error (status 500): server error")
			}
			handler(StreamEvent{Type: "text", Content: "retry-success"})
			handler(StreamEvent{Type: "done", Done: true, StopReason: "end_turn"})
			return nil
		},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	var events []StreamEvent
	err := rp.Stream(context.Background(), &CompletionRequest{}, func(ev StreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should see: partial-from-first (direct), retry-success (buffered), done (buffered)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %+v", len(events), events)
	}
	if events[0].Content != "partial-from-first" {
		t.Errorf("expected 'partial-from-first', got %q", events[0].Content)
	}
	if events[1].Content != "retry-success" {
		t.Errorf("expected 'retry-success', got %q", events[1].Content)
	}
	if !events[2].Done {
		t.Error("expected done event")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestRetryProvider_NoRetryOnContextCanceled(t *testing.T) {
	inner := &testProvider{
		errors: []error{context.Canceled},
	}
	rp := NewRetryProvider(inner, fastRetryConfig())

	_, err := rp.Complete(context.Background(), &CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 call (no retry on context.Canceled), got %d", inner.calls)
	}
}
