package sqlite

import (
	"context"
	"testing"
)

func TestConversationRepository_SaveMessage(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Save a root message (no parent)
	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "Hello, AI!", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Save a reply (with parent)
	parentID := int64(100)
	err = repo.SaveMessage(ctx, 100, 101, &parentID, "assistant", "Hello! How can I help?", "")
	if err != nil {
		t.Fatalf("SaveMessage with parent failed: %v", err)
	}
}

func TestConversationRepository_SaveMessage_DuplicateMsgID(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "First message", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Saving with the same msg_id should fail (UNIQUE constraint)
	err = repo.SaveMessage(ctx, 100, 100, nil, "user", "Duplicate message", "")
	if err == nil {
		t.Fatal("Expected error for duplicate msg_id, got nil")
	}
}

func TestConversationRepository_IsConversationMessage(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Check non-existent message
	exists, threadRootID, err := repo.IsConversationMessage(ctx, 999)
	if err != nil {
		t.Fatalf("IsConversationMessage failed: %v", err)
	}
	if exists {
		t.Error("Expected exists=false for non-existent message")
	}
	if threadRootID != nil {
		t.Errorf("Expected nil threadRootID, got %d", *threadRootID)
	}

	// Save a message and check it
	err = repo.SaveMessage(ctx, 100, 200, nil, "user", "Test message", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	exists, threadRootID, err = repo.IsConversationMessage(ctx, 200)
	if err != nil {
		t.Fatalf("IsConversationMessage failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists=true for saved message")
	}
	if threadRootID == nil {
		t.Fatal("Expected non-nil threadRootID")
	}
	if *threadRootID != 100 {
		t.Errorf("threadRootID = %d, want 100", *threadRootID)
	}
}

func TestConversationRepository_GetThreadChain(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Build a 3-message chain:
	// msg 100 (user) -> msg 101 (assistant) -> msg 102 (user)
	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "What is Go?", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent1 := int64(100)
	err = repo.SaveMessage(ctx, 100, 101, &parent1, "assistant", "Go is a programming language.", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent2 := int64(101)
	err = repo.SaveMessage(ctx, 100, 102, &parent2, "user", "Tell me more.", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Get chain from leaf (msg 102)
	messages, err := repo.GetThreadChain(ctx, 102, 50)
	if err != nil {
		t.Fatalf("GetThreadChain failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// Verify chronological order
	if messages[0].Role != "user" || messages[0].Content != "What is Go?" {
		t.Errorf("messages[0] = {%q, %q}, want {\"user\", \"What is Go?\"}", messages[0].Role, messages[0].Content)
	}
	if messages[1].Role != "assistant" || messages[1].Content != "Go is a programming language." {
		t.Errorf("messages[1] = {%q, %q}, want {\"assistant\", \"Go is a programming language.\"}", messages[1].Role, messages[1].Content)
	}
	if messages[2].Role != "user" || messages[2].Content != "Tell me more." {
		t.Errorf("messages[2] = {%q, %q}, want {\"user\", \"Tell me more.\"}", messages[2].Role, messages[2].Content)
	}
}

func TestConversationRepository_GetThreadChain_Limit(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Build a 4-message chain
	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "Message 1", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent := int64(100)
	err = repo.SaveMessage(ctx, 100, 101, &parent, "assistant", "Message 2", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent = int64(101)
	err = repo.SaveMessage(ctx, 100, 102, &parent, "user", "Message 3", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent = int64(102)
	err = repo.SaveMessage(ctx, 100, 103, &parent, "assistant", "Message 4", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Get chain with limit=2 from leaf
	messages, err := repo.GetThreadChain(ctx, 103, 2)
	if err != nil {
		t.Fatalf("GetThreadChain failed: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages with limit=2, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Message 3" {
		t.Errorf("messages[0] = {%q, %q}, want {\"user\", \"Message 3\"}", messages[0].Role, messages[0].Content)
	}

	if messages[1].Role != "assistant" || messages[1].Content != "Message 4" {
		t.Errorf("messages[1] = {%q, %q}, want {\"assistant\", \"Message 4\"}", messages[1].Role, messages[1].Content)
	}
}

func TestConversationRepository_GetThreadChain_SingleMessage(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Save a single root message
	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "Just one message", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	messages, err := repo.GetThreadChain(ctx, 100, 50)
	if err != nil {
		t.Fatalf("GetThreadChain failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Just one message" {
		t.Errorf("messages[0] = {%q, %q}, want {\"user\", \"Just one message\"}", messages[0].Role, messages[0].Content)
	}
}

func TestConversationRepository_GetThreadChain_NonExistentMessage(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Get chain for non-existent message
	messages, err := repo.GetThreadChain(ctx, 999, 50)
	if err != nil {
		t.Fatalf("GetThreadChain failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for non-existent leaf, got %d", len(messages))
	}
}

func TestConversationRepository_GetThreadChain_WithSenderName(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Build a 3-message chain with sender names
	err := repo.SaveMessage(ctx, 100, 100, nil, "user", "[Mario Rossi]: Hello!", "Mario Rossi")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent1 := int64(100)
	err = repo.SaveMessage(ctx, 100, 101, &parent1, "assistant", "Hi there!", "")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	parent2 := int64(101)
	err = repo.SaveMessage(ctx, 100, 102, &parent2, "user", "[Luigi Verdi]: How are you?", "Luigi Verdi")
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	messages, err := repo.GetThreadChain(ctx, 102, 50)
	if err != nil {
		t.Fatalf("GetThreadChain failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0].Name != "Mario Rossi" {
		t.Errorf("messages[0].Name = %q, want %q", messages[0].Name, "Mario Rossi")
	}
	if messages[1].Name != "" {
		t.Errorf("messages[1].Name = %q, want empty (assistant)", messages[1].Name)
	}
	if messages[2].Name != "Luigi Verdi" {
		t.Errorf("messages[2].Name = %q, want %q", messages[2].Name, "Luigi Verdi")
	}
}
