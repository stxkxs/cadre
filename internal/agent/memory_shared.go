package agent

// SharedMemory provides a crew-scoped namespace over a MemoryStore.
// Multiple agents reference the same SharedMemory to exchange knowledge.
type SharedMemory struct {
	store     MemoryStore
	namespace string // crew-scoped key, e.g. "crew:development"
}

// NewSharedMemory creates a shared memory instance backed by the given store.
func NewSharedMemory(store MemoryStore, crewName string) *SharedMemory {
	return &SharedMemory{
		store:     store,
		namespace: "shared:" + crewName,
	}
}

// Store returns the underlying MemoryStore.
func (sm *SharedMemory) Store() MemoryStore {
	return sm.store
}

// Namespace returns the crew-scoped namespace key.
func (sm *SharedMemory) Namespace() string {
	return sm.namespace
}

// Write stores a message in the shared namespace.
func (sm *SharedMemory) Write(msg Message) error {
	return sm.store.Save(sm.namespace, msg)
}

// Read returns the most recent n messages from the shared namespace.
func (sm *SharedMemory) Read(n int) ([]Message, error) {
	return sm.store.Load(sm.namespace, n)
}

// Search returns messages matching the pattern from the shared namespace.
func (sm *SharedMemory) Search(pattern string) ([]Message, error) {
	return sm.store.Search(sm.namespace, pattern)
}
