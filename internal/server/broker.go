package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cadre-oss/cadre/internal/event"
	"github.com/cadre-oss/cadre/internal/telemetry"
)

// SSEEvent is sent to connected clients.
type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	RunID     string      `json:"run_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Client is a connected SSE client.
type Client struct {
	ID     string
	RunID  string // empty = subscribe to all
	Events chan SSEEvent
}

// Broker manages SSE client connections and broadcasts events.
// It implements event.Hook so it plugs into cadre's event bus.
type Broker struct {
	mu      sync.RWMutex
	clients map[string]*Client
	logger  *telemetry.Logger
}

// NewBroker creates a new SSE broker.
func NewBroker(logger *telemetry.Logger) *Broker {
	return &Broker{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// Subscribe adds a new SSE client. The returned Client's Events channel
// receives events until the context is cancelled.
func (b *Broker) Subscribe(ctx context.Context, clientID, runID string) *Client {
	client := &Client{
		ID:     clientID,
		RunID:  runID,
		Events: make(chan SSEEvent, 64),
	}

	b.mu.Lock()
	b.clients[clientID] = client
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.clients, clientID)
		b.mu.Unlock()
		close(client.Events)
	}()

	return client
}

// Broadcast sends an event to all matching clients.
func (b *Broker) Broadcast(ev SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, client := range b.clients {
		if client.RunID != "" && ev.RunID != "" && client.RunID != ev.RunID {
			continue
		}
		select {
		case client.Events <- ev:
		default:
			// Drop if client buffer is full
			b.logger.Warn("Dropping SSE event for slow client", "client", client.ID)
		}
	}
}

// --- event.Hook interface ---

func (b *Broker) Name() string { return "sse-broker" }

func (b *Broker) Matches(_ event.EventType) bool { return true }

func (b *Broker) IsBlocking() bool { return false }

func (b *Broker) Handle(ev event.Event) error {
	runID, _ := ev.Data["run_id"].(string)

	sseEv := SSEEvent{
		Type:      string(ev.Type),
		Timestamp: ev.Timestamp,
		RunID:     runID,
		Data:      ev.Data,
	}
	b.Broadcast(sseEv)
	return nil
}

// WriteSSE writes an SSE event to the client as JSON.
func WriteSSE(data interface{}) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return b, nil
}
