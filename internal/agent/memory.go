package agent

import (
	"sync"
	"time"

	"github.com/stxkxs/cadre/internal/provider"
)

// Message represents a conversation message
type Message struct {
	Role      string    `json:"role"` // user, assistant, system, tool
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`

	// For tool messages
	ToolName   string `json:"tool_name,omitempty"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// ContentBlocks holds structured content blocks for tool_use / tool_result flows.
	ContentBlocks []provider.ContentBlock `json:"content_blocks,omitempty"`
}

// Memory manages agent conversation history
type Memory struct {
	mu        sync.RWMutex
	memType   string // conversation, long_term, shared
	maxTokens int
	messages  []Message

	// Persistent storage (non-nil for long_term and shared memory).
	store     MemoryStore
	namespace string // agent name or shared namespace key
}

// NewMemory creates a new memory instance
func NewMemory(memType string, maxTokens int) *Memory {
	if memType == "" {
		memType = "conversation"
	}
	if maxTokens == 0 {
		maxTokens = 100000
	}

	return &Memory{
		memType:   memType,
		maxTokens: maxTokens,
		messages:  make([]Message, 0),
	}
}

// SetStore attaches a persistent store to this memory instance.
// On attach it bootstraps recent messages from the store (capped by maxTokens).
func (m *Memory) SetStore(store MemoryStore, namespace string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = store
	m.namespace = namespace

	// Bootstrap: load recent messages from store.
	msgs, err := store.Load(namespace, 200) // load last 200 messages
	if err != nil {
		return err
	}

	// Prepend persisted messages, then truncate to maxTokens.
	m.messages = append(msgs, m.messages...)
	m.truncateIfNeeded()
	return nil
}

// Add adds a message to memory
func (m *Memory) Add(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	m.messages = append(m.messages, msg)

	// Write-through to persistent store.
	if m.store != nil && m.namespace != "" {
		// Best-effort persist — don't block on errors.
		_ = m.store.Save(m.namespace, msg)
	}

	// Simple truncation if we exceed estimated token limit
	// Rough estimate: 1 token ≈ 4 characters
	m.truncateIfNeeded()
}

// Messages returns all messages
func (m *Memory) Messages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// Last returns the last n messages
func (m *Memory) Last(n int) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n >= len(m.messages) {
		result := make([]Message, len(m.messages))
		copy(result, m.messages)
		return result
	}

	start := len(m.messages) - n
	result := make([]Message, n)
	copy(result, m.messages[start:])
	return result
}

// Clear removes all messages
func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = m.messages[:0]

	if m.store != nil && m.namespace != "" {
		_ = m.store.Clear(m.namespace)
	}
}

// Len returns the number of messages
func (m *Memory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// EstimatedTokens returns estimated token count
func (m *Memory) EstimatedTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.estimatedTokensLocked()
}

// truncateIfNeeded removes old messages if we exceed token limit.
// Must be called with m.mu already held (write lock).
func (m *Memory) truncateIfNeeded() {
	for m.estimatedTokensLocked() > m.maxTokens && len(m.messages) > 2 {
		m.messages = m.messages[1:]
	}
}

// estimatedTokensLocked returns estimated token count without locking.
// Caller must hold m.mu.
func (m *Memory) estimatedTokensLocked() int {
	totalChars := 0
	for _, msg := range m.messages {
		totalChars += len(msg.Content)
		for _, block := range msg.ContentBlocks {
			totalChars += len(block.Text) + len(block.Content) + len(block.Name)
		}
	}
	return totalChars / 4
}

// Search searches memory for messages containing a pattern (for long_term memory)
func (m *Memory) Search(pattern string) []Message {
	// If we have a persistent store, search there too.
	if m.store != nil && m.namespace != "" {
		results, err := m.store.Search(m.namespace, pattern)
		if err == nil && len(results) > 0 {
			return results
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []Message
	for _, msg := range m.messages {
		if containsIgnoreCase(msg.Content, pattern) {
			results = append(results, msg)
		}
	}
	return results
}

// GetByRole returns all messages from a specific role
func (m *Memory) GetByRole(role string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []Message
	for _, msg := range m.messages {
		if msg.Role == role {
			results = append(results, msg)
		}
	}
	return results
}

// Type returns the memory type.
func (m *Memory) Type() string {
	return m.memType
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	sLower := toLower(s)
	substrLower := toLower(substr)

	return contains(sLower, substrLower)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
