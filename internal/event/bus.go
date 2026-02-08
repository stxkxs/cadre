package event

import (
	"fmt"
	"sync"
)

// Bus dispatches events to registered hooks.
//
// Dispatch rules:
//  1. Blocking hooks execute sequentially in registration order before returning.
//  2. Non-blocking hooks execute concurrently in goroutines.
//  3. A blocking hook failure returns an error to the caller (stops execution).
//  4. A non-blocking hook failure is logged as a warning.
//  5. A nil Bus is safe to use — all methods are no-ops.
type Bus struct {
	mu      sync.RWMutex
	hooks   []Hook
	enabled bool
	logger  Logger
}

// Logger is a minimal logging interface so the bus doesn't depend on telemetry.
type Logger interface {
	Warn(msg string, keyvals ...interface{})
}

// NewBus creates an enabled event bus. Pass nil logger for silent operation.
func NewBus(logger Logger) *Bus {
	return &Bus{
		hooks:   make([]Hook, 0),
		enabled: true,
		logger:  logger,
	}
}

// Register adds a hook to the bus.
func (b *Bus) Register(h Hook) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hooks = append(b.hooks, h)
}

// SetEnabled controls whether the bus dispatches events.
func (b *Bus) SetEnabled(enabled bool) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = enabled
}

// Emit dispatches an event to all matching hooks.
// Blocking hooks run synchronously; non-blocking hooks run in goroutines.
// Returns the first error from a blocking hook, if any.
func (b *Bus) Emit(ev Event) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	if !b.enabled {
		b.mu.RUnlock()
		return nil
	}
	// Copy hooks slice to avoid holding lock during execution.
	hooks := make([]Hook, len(b.hooks))
	copy(hooks, b.hooks)
	b.mu.RUnlock()

	for _, h := range hooks {
		if !h.Matches(ev.Type) {
			continue
		}

		if h.IsBlocking() {
			if err := h.Handle(ev); err != nil {
				return fmt.Errorf("blocking hook %s failed: %w", h.Name(), err)
			}
		} else {
			go func(hook Hook) {
				defer func() {
					if r := recover(); r != nil && b.logger != nil {
						b.logger.Warn("Non-blocking hook panicked",
							"hook", hook.Name(),
							"event", string(ev.Type),
							"panic", r,
						)
					}
				}()
				if err := hook.Handle(ev); err != nil && b.logger != nil {
					b.logger.Warn("Non-blocking hook failed",
						"hook", hook.Name(),
						"event", string(ev.Type),
						"error", err,
					)
				}
			}(h)
		}
	}

	return nil
}
