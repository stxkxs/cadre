//go:build integration

package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cadre-oss/cadre/internal/agent"
)

func TestMemoryPersistenceAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memory.db")

	// --- Run 1: Create an agent, add messages, close ---
	store1, err := agent.NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	mem1 := agent.NewMemory("long_term", 100000)
	if err := mem1.SetStore(store1, "agent-persistent"); err != nil {
		t.Fatal(err)
	}

	mem1.Add(agent.Message{Role: "user", Content: "What is the project architecture?"})
	mem1.Add(agent.Message{Role: "assistant", Content: "The project uses a modular architecture with agents, tasks, and crews."})
	mem1.Add(agent.Message{Role: "user", Content: "Tell me about the memory system."})

	store1.Close()
	time.Sleep(10 * time.Millisecond) // ensure DB is flushed

	// --- Run 2: New instance, should see all 3 messages ---
	store2, err := agent.NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	mem2 := agent.NewMemory("long_term", 100000)
	if err := mem2.SetStore(store2, "agent-persistent"); err != nil {
		t.Fatal(err)
	}

	msgs := mem2.Messages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 persisted messages, got %d", len(msgs))
	}

	// Search should work across persisted data
	results := mem2.Search("architecture")
	if len(results) < 1 {
		t.Error("expected search to find 'architecture' in persisted messages")
	}

	// Adding new messages should also persist
	mem2.Add(agent.Message{Role: "assistant", Content: "Memory uses SQLite for persistence."})

	// Verify via direct store query
	all, err := store2.Load("agent-persistent", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 total messages in store, got %d", len(all))
	}
}

func TestSharedMemoryAcrossAgents(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared.db")

	store, err := agent.NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	shared := agent.NewSharedMemory(store, "dev-crew")

	// Agent 1 writes
	shared.Write(agent.Message{Role: "user", Content: "API spec: POST /users creates a user"})

	// Agent 2 writes
	shared.Write(agent.Message{Role: "user", Content: "Schema: users table with id, name, email"})

	// Both should be visible
	msgs, err := shared.Read(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 shared messages, got %d", len(msgs))
	}

	// Search
	results, err := shared.Search("API")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 search result, got %d", len(results))
	}

	// Agent 3 can read what agents 1 and 2 wrote
	mem3 := agent.NewMemory("shared", 100000)
	if err := mem3.SetStore(store, shared.Namespace()); err != nil {
		t.Fatal(err)
	}
	if mem3.Len() != 2 {
		t.Errorf("agent 3 expected 2 messages from shared memory, got %d", mem3.Len())
	}
}
