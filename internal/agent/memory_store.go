package agent

// MemoryStore persists agent memory across sessions.
type MemoryStore interface {
	// Save persists a message under the given agent namespace.
	Save(namespace string, msg Message) error

	// Load returns the most recent n messages for the namespace.
	Load(namespace string, n int) ([]Message, error)

	// Search returns messages whose content matches the pattern (case-insensitive LIKE).
	Search(namespace string, pattern string) ([]Message, error)

	// Clear deletes all messages for the namespace.
	Clear(namespace string) error

	// Close releases any resources held by the store.
	Close() error
}
