package storage

import (
	"testing"

	"github.com/spf13/afero"
)

func TestStorage_SaveAndRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	storage := New(fs, "/media")

	hash := "abc123"
	data := []byte("test data")

	// Save
	if err := storage.Save(hash, data); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Read back
	result, err := storage.Read(hash)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(result) != string(data) {
		t.Errorf("Read() = %q, want %q", result, data)
	}
}

func TestStorage_IdempotentSave(t *testing.T) {
	fs := afero.NewMemMapFs()
	storage := New(fs, "/media")

	hash := "abc123"
	data1 := []byte("test data")
	data2 := []byte("test data")

	// Save once
	if err := storage.Save(hash, data1); err != nil {
		t.Fatalf("First Save() failed: %v", err)
	}

	// Save again with identical data
	if err := storage.Save(hash, data2); err != nil {
		t.Fatalf("Second Save() failed: %v", err)
	}

	// Verify content is still correct
	result, err := storage.Read(hash)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(result) != string(data1) {
		t.Errorf("Read() = %q, want %q", result, data1)
	}
}

func TestStorage_Delete(t *testing.T) {
	fs := afero.NewMemMapFs()
	storage := New(fs, "/media")

	hash := "abc123"
	data := []byte("test data")

	// Save
	if err := storage.Save(hash, data); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify exists
	exists, err := storage.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true before delete")
	}

	// Delete
	if err := storage.Delete(hash); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify doesn't exist
	exists, err = storage.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() failed after delete: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false after delete")
	}
}

func TestStorage_Exists(t *testing.T) {
	fs := afero.NewMemMapFs()
	storage := New(fs, "/media")

	hash := "abc123"

	// Should not exist initially
	exists, err := storage.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false for non-existent file")
	}

	// Save
	if err := storage.Save(hash, []byte("test")); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Should exist now
	exists, err = storage.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() failed after save: %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true after save")
	}
}

func TestStorage_ReadNonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	storage := New(fs, "/media")

	_, err := storage.Read("nonexistent")
	if err == nil {
		t.Error("Read() of non-existent file should return error")
	}
}
