package provider

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	JitterFraction float64
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     60 * time.Second,
		JitterFraction: 0.2,
	}
}

// RetryProvider wraps a Provider with automatic retry for transient errors.
type RetryProvider struct {
	inner  Provider
	config RetryConfig
}

// NewRetryProvider creates a RetryProvider wrapping inner.
func NewRetryProvider(inner Provider, cfg RetryConfig) *RetryProvider {
	return &RetryProvider{inner: inner, config: cfg}
}

func (r *RetryProvider) Name() string {
	return r.inner.Name()
}

func (r *RetryProvider) Complete(ctx context.Context, req *CompletionRequest) (*Response, error) {
	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		resp, err := r.inner.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		if !r.isRetryable(err) {
			return nil, err
		}

		if attempt == r.config.MaxRetries {
			break
		}

		delay := r.backoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
}

func (r *RetryProvider) Stream(ctx context.Context, req *CompletionRequest, handler StreamHandler) error {
	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt == 0 {
			// First attempt: stream directly for real-time output.
			err := r.inner.Stream(ctx, req, handler)
			if err == nil {
				return nil
			}
			lastErr = err
			if !r.isRetryable(err) {
				return err
			}
		} else {
			// Retry attempts: buffer to prevent duplicate partial output.
			var buffered []StreamEvent
			bufHandler := func(ev StreamEvent) {
				buffered = append(buffered, ev)
			}
			err := r.inner.Stream(ctx, req, bufHandler)
			if err == nil {
				for _, ev := range buffered {
					handler(ev)
				}
				return nil
			}
			lastErr = err
			if !r.isRetryable(err) {
				for _, ev := range buffered {
					handler(ev)
				}
				return err
			}
		}

		if attempt == r.config.MaxRetries {
			break
		}

		delay := r.backoff(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
}

// isRetryable determines whether an error should be retried.
func (r *RetryProvider) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are never retryable.
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	msg := err.Error()

	// Network errors (from claude.go: "request failed: ...")
	if strings.HasPrefix(msg, "request failed:") {
		return true
	}

	// API errors (from claude.go: "API error (status %d): ...")
	if strings.Contains(msg, "API error (status ") {
		for _, code := range []string{"429", "500", "502", "503", "529"} {
			if strings.Contains(msg, fmt.Sprintf("status %s)", code)) {
				return true
			}
		}
		// Non-retryable status codes (400, 401, 403, 404, etc.)
		return false
	}

	// Unknown errors — don't retry by default.
	return false
}

// backoff calculates the delay for a given attempt using exponential backoff with jitter.
func (r *RetryProvider) backoff(attempt int) time.Duration {
	base := float64(r.config.InitialBackoff) * math.Pow(2, float64(attempt))
	if base > float64(r.config.MaxBackoff) {
		base = float64(r.config.MaxBackoff)
	}

	jitter := base * r.config.JitterFraction * (rand.Float64()*2 - 1) // ±jitter
	delay := time.Duration(base + jitter)
	if delay < 0 {
		delay = 0
	}
	return delay
}
