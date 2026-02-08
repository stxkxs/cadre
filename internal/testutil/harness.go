package testutil

import (
	"testing"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/event"
	"github.com/cadre-oss/cadre/internal/provider"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// TestHarness provides everything needed for integration tests:
// config, state, events, mock provider, and assertion helpers.
type TestHarness struct {
	T        *testing.T
	Config   *config.Config
	StateMgr *state.Manager
	EventBus *event.Bus
	Logger   *telemetry.Logger
	Provider *MockProvider
	Events   []event.Event // captured events
}

// NewTestHarness creates a test harness with default configuration.
func NewTestHarness(t *testing.T) *TestHarness {
	t.Helper()

	stateMgr, err := state.NewManager("memory", "")
	if err != nil {
		t.Fatal(err)
	}

	logger := TestLogger()
	bus := event.NewBus(logger)

	h := &TestHarness{
		T:        t,
		Config:   TestConfig(),
		StateMgr: stateMgr,
		EventBus: bus,
		Logger:   logger,
		Provider: &MockProvider{},
		Events:   make([]event.Event, 0),
	}

	// Capture events via a hook
	bus.Register(&eventCapture{harness: h})

	return h
}

// SetResponses queues mock provider responses.
func (h *TestHarness) SetResponses(responses ...*provider.Response) {
	h.Provider.Responses = responses
}

// AssertEventEmitted checks that an event with the given type was emitted.
func (h *TestHarness) AssertEventEmitted(eventType event.EventType) {
	h.T.Helper()
	for _, e := range h.Events {
		if e.Type == eventType {
			return
		}
	}
	h.T.Errorf("expected event %q to be emitted", eventType)
}

// AssertNoEvent checks that an event type was NOT emitted.
func (h *TestHarness) AssertNoEvent(eventType event.EventType) {
	h.T.Helper()
	for _, e := range h.Events {
		if e.Type == eventType {
			h.T.Errorf("expected event %q NOT to be emitted, but it was", eventType)
			return
		}
	}
}

// EventCount returns the number of events with the given type.
func (h *TestHarness) EventCount(eventType event.EventType) int {
	count := 0
	for _, e := range h.Events {
		if e.Type == eventType {
			count++
		}
	}
	return count
}

// eventCapture is a non-blocking hook that records events.
type eventCapture struct {
	harness *TestHarness
}

func (c *eventCapture) Name() string             { return "test-capture" }
func (c *eventCapture) Matches(event.EventType) bool { return true } // match all
func (c *eventCapture) IsBlocking() bool          { return true }     // sync for tests

func (c *eventCapture) Handle(ev event.Event) error {
	c.harness.Events = append(c.harness.Events, ev)
	return nil
}
