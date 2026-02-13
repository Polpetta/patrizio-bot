package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"testing"

	"github.com/polpetta/patrizio/internal/database"
	"github.com/polpetta/patrizio/internal/domain"
)

//go:embed testdata/migrations
var testMigrationsFS embed.FS

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db, testMigrationsFS, "testdata/migrations"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestRepository_CreateTextFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	err := repo.CreateTextFilter(ctx, 123, []string{"hello", "hi"}, "Hello to you!")
	if err != nil {
		t.Fatalf("CreateTextFilter failed: %v", err)
	}

	// Verify filter was created
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(filters))
	}

	filter := filters[0]
	if filter.ResponseType != domain.ResponseTypeText {
		t.Errorf("ResponseType = %q, want %q", filter.ResponseType, domain.ResponseTypeText)
	}
	if filter.Response.ResponseText != "Hello to you!" {
		t.Errorf("ResponseText = %q, want %q", filter.Response.ResponseText, "Hello to you!")
	}
	if len(filter.Triggers) != 2 {
		t.Errorf("Expected 2 triggers, got %d", len(filter.Triggers))
	}
}

func TestRepository_CreateTextFilter_DuplicateTrigger(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create first filter
	err := repo.CreateTextFilter(ctx, 123, []string{"hello"}, "First response")
	if err != nil {
		t.Fatalf("First CreateTextFilter failed: %v", err)
	}

	// Try to create second filter with same trigger in same chat
	err = repo.CreateTextFilter(ctx, 123, []string{"hello"}, "Second response")
	if err == nil {
		t.Fatal("Expected error for duplicate trigger, got nil")
	}
}

func TestRepository_CreateMediaFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	err := repo.CreateMediaFilter(ctx, 123, []string{"dog"}, "abc123hash", domain.MediaTypeImage)
	if err != nil {
		t.Fatalf("CreateMediaFilter failed: %v", err)
	}

	// Verify filter was created
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(filters))
	}

	filter := filters[0]
	if filter.ResponseType != domain.ResponseTypeMedia {
		t.Errorf("ResponseType = %q, want %q", filter.ResponseType, domain.ResponseTypeMedia)
	}
	if filter.Response.MediaHash != "abc123hash" {
		t.Errorf("MediaHash = %q, want %q", filter.Response.MediaHash, "abc123hash")
	}
	if filter.Response.MediaType != domain.MediaTypeImage {
		t.Errorf("MediaType = %q, want %q", filter.Response.MediaType, domain.MediaTypeImage)
	}
}

func TestRepository_CreateReactionFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	err := repo.CreateReactionFilter(ctx, 123, []string{"lol"}, "😂")
	if err != nil {
		t.Fatalf("CreateReactionFilter failed: %v", err)
	}

	// Verify filter was created
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(filters))
	}

	filter := filters[0]
	if filter.ResponseType != domain.ResponseTypeReaction {
		t.Errorf("ResponseType = %q, want %q", filter.ResponseType, domain.ResponseTypeReaction)
	}
	if filter.Response.Reaction != "😂" {
		t.Errorf("Reaction = %q, want %q", filter.Response.Reaction, "😂")
	}
}

func TestRepository_RemoveTrigger_RemovesFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create filter with single trigger
	err := repo.CreateTextFilter(ctx, 123, []string{"hello"}, "Hi there!")
	if err != nil {
		t.Fatalf("CreateTextFilter failed: %v", err)
	}

	// Remove the only trigger
	mediaHash, err := repo.RemoveTrigger(ctx, 123, "hello")
	if err != nil {
		t.Fatalf("RemoveTrigger failed: %v", err)
	}
	if mediaHash != nil {
		t.Errorf("Expected nil mediaHash for text filter, got %q", *mediaHash)
	}

	// Verify filter was deleted
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}
	if len(filters) != 0 {
		t.Errorf("Expected 0 filters after removal, got %d", len(filters))
	}
}

func TestRepository_RemoveTrigger_KeepsFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create filter with two triggers
	err := repo.CreateTextFilter(ctx, 123, []string{"hello", "hi"}, "Greetings!")
	if err != nil {
		t.Fatalf("CreateTextFilter failed: %v", err)
	}

	// Remove one trigger
	_, err = repo.RemoveTrigger(ctx, 123, "hello")
	if err != nil {
		t.Fatalf("RemoveTrigger failed: %v", err)
	}

	// Verify filter still exists with one trigger
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(filters))
	}
	if len(filters[0].Triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(filters[0].Triggers))
	}
}

func TestRepository_RemoveTrigger_ReturnsMediaHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create media filter
	err := repo.CreateMediaFilter(ctx, 123, []string{"dog"}, "abc123", domain.MediaTypeImage)
	if err != nil {
		t.Fatalf("CreateMediaFilter failed: %v", err)
	}

	// Remove trigger (should delete filter and return hash)
	mediaHash, err := repo.RemoveTrigger(ctx, 123, "dog")
	if err != nil {
		t.Fatalf("RemoveTrigger failed: %v", err)
	}
	if mediaHash == nil {
		t.Fatal("Expected media hash, got nil")
	}
	if *mediaHash != "abc123" {
		t.Errorf("MediaHash = %q, want %q", *mediaHash, "abc123")
	}
}

func TestRepository_RemoveAllFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create multiple filters
	repo.CreateTextFilter(ctx, 123, []string{"hello"}, "Hi!")
	repo.CreateMediaFilter(ctx, 123, []string{"dog"}, "hash1", domain.MediaTypeImage)
	repo.CreateMediaFilter(ctx, 123, []string{"cat"}, "hash2", domain.MediaTypeImage)

	// Remove all
	hashes, err := repo.RemoveAllFilters(ctx, 123)
	if err != nil {
		t.Fatalf("RemoveAllFilters failed: %v", err)
	}

	// Should return 2 media hashes
	if len(hashes) != 2 {
		t.Errorf("Expected 2 media hashes, got %d", len(hashes))
	}

	// Verify all filters deleted
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}
	if len(filters) != 0 {
		t.Errorf("Expected 0 filters, got %d", len(filters))
	}
}

func TestRepository_FindMatchingFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create filters
	repo.CreateTextFilter(ctx, 123, []string{"dog"}, "Woof!")
	repo.CreateTextFilter(ctx, 123, []string{"i love dogs"}, "Dogs are great!")
	repo.CreateReactionFilter(ctx, 123, []string{"lol"}, "😂")

	tests := []struct {
		name     string
		message  string
		expected int
	}{
		{
			name:     "Match single word",
			message:  "i love my dog",
			expected: 1, // "dog" only (not "i love dogs" because of "my")
		},
		{
			name:     "Match at start",
			message:  "dog is cute",
			expected: 1, // "dog"
		},
		{
			name:     "Match at end",
			message:  "look at that dog",
			expected: 1, // "dog" only
		},
		{
			name:     "No match - partial word",
			message:  "hotdog",
			expected: 0,
		},
		{
			name:     "Match reaction",
			message:  "that was funny lol",
			expected: 1, // "lol"
		},
		{
			name:     "No matches",
			message:  "cat bird fish",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := domain.NormalizeMessage(tt.message)
			matches, err := repo.FindMatchingFilters(ctx, 123, normalized)
			if err != nil {
				t.Fatalf("FindMatchingFilters failed: %v", err)
			}
			if len(matches) != tt.expected {
				t.Errorf("Expected %d matches, got %d", tt.expected, len(matches))
			}
		})
	}
}

func TestRepository_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Create filter
	err := repo.CreateTextFilter(ctx, 123, []string{"hello"}, "Hi!")
	if err != nil {
		t.Fatalf("CreateTextFilter failed: %v", err)
	}

	// Remove trigger (will delete filter via cascade)
	_, err = repo.RemoveTrigger(ctx, 123, "hello")
	if err != nil {
		t.Fatalf("RemoveTrigger failed: %v", err)
	}

	// Verify filter and all related data deleted
	filters, err := repo.ListFilters(ctx, 123)
	if err != nil {
		t.Fatalf("ListFilters failed: %v", err)
	}
	if len(filters) != 0 {
		t.Errorf("Expected cascade delete to remove filter, but got %d filters", len(filters))
	}
}

func TestRepository_InvalidTrigger(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	// Try to create filter with invalid trigger
	err := repo.CreateTextFilter(ctx, 123, []string{"c++"}, "Programming language")
	if err == nil {
		t.Fatal("Expected error for invalid trigger, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidTrigger) {
		t.Errorf("Expected ErrInvalidTrigger, got %v", err)
	}
}
