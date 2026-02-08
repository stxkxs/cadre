package event

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testLogger records warn messages.
type testLogger struct {
	mu       sync.Mutex
	warnings []string
}

func (l *testLogger) Warn(msg string, keyvals ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnings = append(l.warnings, msg)
}

func (l *testLogger) Info(msg string, keyvals ...interface{}) {}
func (l *testLogger) Debug(msg string, keyvals ...interface{}) {}

// collectHook records handled events.
type collectHook struct {
	baseHook
	mu       sync.Mutex
	handled  []Event
	handleFn func(Event) error
}

func newCollectHook(name string, events []EventType, blocking bool) *collectHook {
	return &collectHook{
		baseHook: baseHook{name: name, events: events, blocking: blocking},
	}
}

func (h *collectHook) Handle(ev Event) error {
	if h.handleFn != nil {
		return h.handleFn(ev)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handled = append(h.handled, ev)
	return nil
}

func (h *collectHook) events() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]Event, len(h.handled))
	copy(cp, h.handled)
	return cp
}

func TestBus_Emit_BlockingHook(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("test", []EventType{TaskStarted}, true)
	bus.Register(hook)

	ev := NewEvent(TaskStarted, map[string]interface{}{"task": "a"})
	err := bus.Emit(ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handled := hook.events()
	if len(handled) != 1 {
		t.Fatalf("expected 1 handled event, got %d", len(handled))
	}
	if handled[0].Type != TaskStarted {
		t.Errorf("expected TaskStarted, got %s", handled[0].Type)
	}
}

func TestBus_Emit_NonBlockingHook(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("async", []EventType{TaskCompleted}, false)
	bus.Register(hook)

	ev := NewEvent(TaskCompleted, nil)
	bus.Emit(ev)

	// Give the goroutine time to execute.
	time.Sleep(50 * time.Millisecond)

	handled := hook.events()
	if len(handled) != 1 {
		t.Fatalf("expected 1 handled event, got %d", len(handled))
	}
}

func TestBus_Emit_RoutingByEventType(t *testing.T) {
	bus := NewBus(nil)
	taskHook := newCollectHook("task-hook", []EventType{TaskStarted, TaskCompleted}, true)
	crewHook := newCollectHook("crew-hook", []EventType{CrewStarted}, true)
	bus.Register(taskHook)
	bus.Register(crewHook)

	bus.Emit(NewEvent(TaskStarted, nil))
	bus.Emit(NewEvent(CrewStarted, nil))
	bus.Emit(NewEvent(TaskCompleted, nil))

	taskEvents := taskHook.events()
	crewEvents := crewHook.events()

	if len(taskEvents) != 2 {
		t.Errorf("expected task hook to handle 2 events, got %d", len(taskEvents))
	}
	if len(crewEvents) != 1 {
		t.Errorf("expected crew hook to handle 1 event, got %d", len(crewEvents))
	}
}

func TestBus_Emit_NoMatchingEvents(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("test", []EventType{CrewFailed}, true)
	bus.Register(hook)

	bus.Emit(NewEvent(TaskStarted, nil))

	if len(hook.events()) != 0 {
		t.Error("hook should not have been called for non-matching event")
	}
}

func TestBus_Emit_MatchAllEvents(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("catch-all", nil, true) // nil events = match all
	bus.Register(hook)

	bus.Emit(NewEvent(TaskStarted, nil))
	bus.Emit(NewEvent(CrewCompleted, nil))

	if len(hook.events()) != 2 {
		t.Errorf("expected 2 events, got %d", len(hook.events()))
	}
}

func TestBus_BlockingHookError(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("failing", []EventType{TaskStarted}, true)
	hook.handleFn = func(ev Event) error {
		return fmt.Errorf("hook error")
	}
	bus.Register(hook)

	err := bus.Emit(NewEvent(TaskStarted, nil))
	if err == nil {
		t.Fatal("expected error from blocking hook")
	}
}

func TestBus_NonBlockingHookErrorLogged(t *testing.T) {
	logger := &testLogger{}
	bus := NewBus(logger)
	hook := newCollectHook("failing-async", []EventType{TaskStarted}, false)
	hook.handleFn = func(ev Event) error {
		return fmt.Errorf("async hook error")
	}
	bus.Register(hook)

	bus.Emit(NewEvent(TaskStarted, nil))
	time.Sleep(50 * time.Millisecond)

	logger.mu.Lock()
	defer logger.mu.Unlock()
	if len(logger.warnings) == 0 {
		t.Error("expected warning to be logged for failed async hook")
	}
}

func TestBus_BlockingHooksSequential(t *testing.T) {
	bus := NewBus(nil)
	var order []string
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		idx := i
		hook := newCollectHook(fmt.Sprintf("hook-%d", idx), []EventType{TaskStarted}, true)
		hook.handleFn = func(ev Event) error {
			mu.Lock()
			order = append(order, fmt.Sprintf("hook-%d", idx))
			mu.Unlock()
			return nil
		}
		bus.Register(hook)
	}

	bus.Emit(NewEvent(TaskStarted, nil))

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("expected 3 hook executions, got %d", len(order))
	}
	// Blocking hooks execute in registration order.
	for i, name := range order {
		expected := fmt.Sprintf("hook-%d", i)
		if name != expected {
			t.Errorf("expected %s at position %d, got %s", expected, i, name)
		}
	}
}

func TestBus_Disabled(t *testing.T) {
	bus := NewBus(nil)
	hook := newCollectHook("test", nil, true)
	bus.Register(hook)

	bus.SetEnabled(false)
	bus.Emit(NewEvent(TaskStarted, nil))

	if len(hook.events()) != 0 {
		t.Error("disabled bus should not dispatch events")
	}
}

func TestBus_NilBusSafe(t *testing.T) {
	var bus *Bus

	// All operations should be no-ops, not panic.
	bus.Register(nil)
	bus.SetEnabled(false)
	err := bus.Emit(NewEvent(TaskStarted, nil))
	if err != nil {
		t.Errorf("nil bus Emit should return nil error, got %v", err)
	}
}

func TestBus_ConcurrentEmit(t *testing.T) {
	bus := NewBus(nil)
	var count int64
	hook := newCollectHook("concurrent", nil, true)
	hook.handleFn = func(ev Event) error {
		atomic.AddInt64(&count, 1)
		return nil
	}
	bus.Register(hook)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Emit(NewEvent(TaskStarted, nil))
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&count) != 100 {
		t.Errorf("expected 100 hook invocations, got %d", count)
	}
}
