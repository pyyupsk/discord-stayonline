// Package store provides configuration storage implementations.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/pyyupsk/discord-stayonline/internal/config"
)

// File handles configuration persistence with atomic writes.
type File struct {
	path string
	mu   sync.RWMutex
}

// NewFile creates a new file-based configuration store.
// The path should be the full path to the config.json file.
func NewFile(path string) *File {
	return &File{
		path: path,
	}
}

// Load reads the configuration from disk.
// Returns a default configuration if the file doesn't exist.
func (s *File) Load() (*config.Configuration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Return default config if file doesn't exist
			return config.Default(), nil
		}
		return nil, err
	}

	// Handle empty file
	if len(data) == 0 {
		return config.Default(), nil
	}

	var cfg config.Configuration
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to disk using atomic write.
// It writes to a temporary file first, then renames to prevent corruption.
func (s *File) Save(cfg *config.Configuration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write to temporary file first
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, s.path)
}

// Path returns the configuration file path.
func (s *File) Path() string {
	return s.path
}
