package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cadre-oss/cadre/internal/provider"
)

func TestSQLiteMemoryStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_memory.db")

	store, err := NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ns := "test-agent"
	now := time.Now().Truncate(time.Second)

	// Save messages
	msgs := []Message{
		{Role: "user", Content: "hello", Timestamp: now},
		{Role: "assistant", Content: "hi there", Timestamp: now.Add(time.Second)},
		{Role: "user", Content: "how are you?", Timestamp: now.Add(2 * time.Second)},
	}
	for _, m := range msgs {
		if err := store.Save(ns, m); err != nil {
			t.Fatal(err)
		}
	}

	// Load all
	loaded, err := store.Load(ns, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(loaded))
	}
	if loaded[0].Content != "hello" {
		t.Errorf("expected first message 'hello', got %q", loaded[0].Content)
	}
	if loaded[2].Content != "how are you?" {
		t.Errorf("expected last message 'how are you?', got %q", loaded[2].Content)
	}

	// Load with limit
	limited, err := store.Load(ns, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(limited))
	}
	if limited[0].Content != "hi there" {
		t.Errorf("expected 'hi there', got %q", limited[0].Content)
	}
}

func TestSQLiteMemoryStore_Search(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "search.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ns := "agent-search"
	store.Save(ns, Message{Role: "user", Content: "I like apples", Timestamp: time.Now()})
	store.Save(ns, Message{Role: "assistant", Content: "Oranges are good too", Timestamp: time.Now()})
	store.Save(ns, Message{Role: "user", Content: "Apple pie is the best", Timestamp: time.Now()})

	results, err := store.Search(ns, "apple")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSQLiteMemoryStore_Clear(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "clear.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ns := "agent-clear"
	store.Save(ns, Message{Role: "user", Content: "test", Timestamp: time.Now()})

	if err := store.Clear(ns); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(ns, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", len(loaded))
	}
}

func TestSQLiteMemoryStore_Namespaces(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "ns.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Save("agent-a", Message{Role: "user", Content: "msg a", Timestamp: time.Now()})
	store.Save("agent-b", Message{Role: "user", Content: "msg b", Timestamp: time.Now()})

	aMessages, _ := store.Load("agent-a", 10)
	bMessages, _ := store.Load("agent-b", 10)

	if len(aMessages) != 1 || aMessages[0].Content != "msg a" {
		t.Errorf("agent-a: expected 1 message 'msg a', got %v", aMessages)
	}
	if len(bMessages) != 1 || bMessages[0].Content != "msg b" {
		t.Errorf("agent-b: expected 1 message 'msg b', got %v", bMessages)
	}
}

func TestSQLiteMemoryStore_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var wg sync.WaitGroup
	ns := "concurrent-agent"

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.Save(ns, Message{
				Role:      "user",
				Content:   "concurrent message",
				Timestamp: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	loaded, err := store.Load(ns, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(loaded))
	}
}

func TestMemory_PersistenceAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")

	// First instance: write messages
	store1, err := NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	mem1 := NewMemory("long_term", 100000)
	mem1.SetStore(store1, "agent-persist")
	mem1.Add(Message{Role: "user", Content: "remember this"})
	mem1.Add(Message{Role: "assistant", Content: "I will remember"})
	store1.Close()

	// Second instance: should see persisted messages
	store2, err := NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	mem2 := NewMemory("long_term", 100000)
	mem2.SetStore(store2, "agent-persist")

	msgs := mem2.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 persisted messages, got %d", len(msgs))
	}
	if msgs[0].Content != "remember this" {
		t.Errorf("expected 'remember this', got %q", msgs[0].Content)
	}
}

func TestMemory_LongTermSearch(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "search.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mem := NewMemory("long_term", 100000)
	mem.SetStore(store, "search-agent")
	mem.Add(Message{Role: "user", Content: "Go is great for concurrency"})
	mem.Add(Message{Role: "assistant", Content: "Yes, goroutines are lightweight"})

	results := mem.Search("concurrency")
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
}

func TestMemory_Truncation(t *testing.T) {
	mem := NewMemory("conversation", 10) // very small token limit
	mem.Add(Message{Role: "user", Content: "a very long message that definitely exceeds the tiny token limit we have set"})
	mem.Add(Message{Role: "assistant", Content: "another message"})
	mem.Add(Message{Role: "user", Content: "third"})

	// Should not panic, and should have at least 2 messages (minimum)
	msgs := mem.Messages()
	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages after truncation, got %d", len(msgs))
	}
}

func TestMemory_Type(t *testing.T) {
	mem := NewMemory("long_term", 100000)
	if mem.Type() != "long_term" {
		t.Errorf("expected type 'long_term', got %q", mem.Type())
	}

	memDefault := NewMemory("", 0)
	if memDefault.Type() != "conversation" {
		t.Errorf("expected type 'conversation', got %q", memDefault.Type())
	}
}

func TestSQLiteMemoryStore_DirectoryCreation(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c", "memory.db")

	store, err := NewSQLiteMemoryStore(nested)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(nested)); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestSQLiteMemoryStore_ContentBlocks_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteMemoryStore(filepath.Join(dir, "blocks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ns := "agent-blocks"
	now := time.Now().Truncate(time.Second)
	toolInput := json.RawMessage(`{"query":"test"}`)

	// Save assistant message with tool_use content blocks
	assistantMsg := Message{
		Role:      "assistant",
		Content:   "Let me search.",
		Timestamp: now,
		ContentBlocks: []provider.ContentBlock{
			provider.TextBlock("Let me search."),
			provider.ToolUseBlock("toolu_01", "grep", toolInput),
		},
	}
	if err := store.Save(ns, assistantMsg); err != nil {
		t.Fatal(err)
	}

	// Save user message with tool_result content blocks
	userMsg := Message{
		Role:      "user",
		Content:   "",
		Timestamp: now.Add(time.Second),
		ContentBlocks: []provider.ContentBlock{
			provider.ToolResultBlock(provider.ToolResult{ID: "toolu_01", Result: "found 3 matches"}),
		},
	}
	if err := store.Save(ns, userMsg); err != nil {
		t.Fatal(err)
	}

	// Save a plain message without content blocks
	plainMsg := Message{
		Role:      "assistant",
		Content:   "Here are the results.",
		Timestamp: now.Add(2 * time.Second),
	}
	if err := store.Save(ns, plainMsg); err != nil {
		t.Fatal(err)
	}

	// Load and verify round-trip
	loaded, err := store.Load(ns, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(loaded))
	}

	// First message: assistant with tool_use blocks
	if len(loaded[0].ContentBlocks) != 2 {
		t.Fatalf("expected 2 content blocks on first message, got %d", len(loaded[0].ContentBlocks))
	}
	if loaded[0].ContentBlocks[0].Type != "text" {
		t.Errorf("expected 'text' block, got %q", loaded[0].ContentBlocks[0].Type)
	}
	if loaded[0].ContentBlocks[1].Type != "tool_use" {
		t.Errorf("expected 'tool_use' block, got %q", loaded[0].ContentBlocks[1].Type)
	}
	if loaded[0].ContentBlocks[1].ID != "toolu_01" {
		t.Errorf("expected tool_use ID 'toolu_01', got %q", loaded[0].ContentBlocks[1].ID)
	}
	if loaded[0].ContentBlocks[1].Name != "grep" {
		t.Errorf("expected tool name 'grep', got %q", loaded[0].ContentBlocks[1].Name)
	}

	// Second message: user with tool_result blocks
	if len(loaded[1].ContentBlocks) != 1 {
		t.Fatalf("expected 1 content block on second message, got %d", len(loaded[1].ContentBlocks))
	}
	if loaded[1].ContentBlocks[0].Type != "tool_result" {
		t.Errorf("expected 'tool_result' block, got %q", loaded[1].ContentBlocks[0].Type)
	}
	if loaded[1].ContentBlocks[0].ToolUseID != "toolu_01" {
		t.Errorf("expected tool_use_id 'toolu_01', got %q", loaded[1].ContentBlocks[0].ToolUseID)
	}
	if loaded[1].ContentBlocks[0].Content != "found 3 matches" {
		t.Errorf("expected result content 'found 3 matches', got %q", loaded[1].ContentBlocks[0].Content)
	}

	// Third message: plain text, no content blocks
	if len(loaded[2].ContentBlocks) != 0 {
		t.Errorf("expected 0 content blocks on third message, got %d", len(loaded[2].ContentBlocks))
	}
}

func TestSQLiteMemoryStore_ContentBlocks_PersistenceAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist-blocks.db")

	// First instance: write message with content blocks
	store1, err := NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	toolInput := json.RawMessage(`{"file":"main.go"}`)
	store1.Save("agent", Message{
		Role:      "assistant",
		Content:   "Reading file.",
		Timestamp: time.Now(),
		ContentBlocks: []provider.ContentBlock{
			provider.TextBlock("Reading file."),
			provider.ToolUseBlock("toolu_02", "file_read", toolInput),
		},
	})
	store1.Close()

	// Second instance: should see persisted content blocks
	store2, err := NewSQLiteMemoryStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	loaded, err := store2.Load("agent", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 message, got %d", len(loaded))
	}
	if len(loaded[0].ContentBlocks) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(loaded[0].ContentBlocks))
	}
	if loaded[0].ContentBlocks[1].Name != "file_read" {
		t.Errorf("expected tool name 'file_read', got %q", loaded[0].ContentBlocks[1].Name)
	}
}
