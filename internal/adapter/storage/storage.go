// Package storage provides media file storage implementation using Afero VFS.
package storage

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

// Storage implements the domain MediaStorage port using Afero VFS.
type Storage struct {
	fs   afero.Fs
	path string
}

// New creates a new Storage instance with the given filesystem and base path.
func New(fs afero.Fs, basePath string) *Storage {
	return &Storage{
		fs:   fs,
		path: basePath,
	}
}

// Save writes media data to a file named by its hash.
func (s *Storage) Save(hash string, data []byte) error {
	filePath := filepath.Join(s.path, hash)
	if err := afero.WriteFile(s.fs, filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save media file %s: %w", hash, err)
	}
	return nil
}

// Delete removes a media file by its hash.
func (s *Storage) Delete(hash string) error {
	filePath := filepath.Join(s.path, hash)
	if err := s.fs.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete media file %s: %w", hash, err)
	}
	return nil
}

// Read retrieves media data by its hash.
func (s *Storage) Read(hash string) ([]byte, error) {
	filePath := filepath.Join(s.path, hash)
	data, err := afero.ReadFile(s.fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read media file %s: %w", hash, err)
	}
	return data, nil
}

// Exists checks if a media file exists by its hash.
func (s *Storage) Exists(hash string) (bool, error) {
	filePath := filepath.Join(s.path, hash)
	exists, err := afero.Exists(s.fs, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to check media file %s: %w", hash, err)
	}
	return exists, nil
}
