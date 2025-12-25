package config

import (
	"database/sql"
	"encoding/json"
	"sync"

	_ "github.com/lib/pq"
)

// DBStore handles configuration persistence using PostgreSQL.
type DBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDBStore creates a new database-backed configuration store.
// It automatically creates the required table if it doesn't exist.
func NewDBStore(databaseURL string) (*DBStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	store := &DBStore{db: db}

	// Create table if not exists
	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate creates the configuration table if it doesn't exist.
func (s *DBStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS configuration (
			id INTEGER PRIMARY KEY DEFAULT 1,
			data JSONB NOT NULL DEFAULT '{}',
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			CONSTRAINT single_row CHECK (id = 1)
		)
	`)
	return err
}

// Load reads the configuration from the database.
// Returns a default configuration if no record exists.
func (s *DBStore) Load() (*Configuration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []byte
	err := s.db.QueryRow("SELECT data FROM configuration WHERE id = 1").Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return Default(), nil
		}
		return nil, err
	}

	var cfg Configuration
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to the database using upsert.
func (s *DBStore) Save(cfg *Configuration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO configuration (id, data, updated_at)
		VALUES (1, $1, NOW())
		ON CONFLICT (id) DO UPDATE SET
			data = EXCLUDED.data,
			updated_at = NOW()
	`, data)

	return err
}

// Close closes the database connection.
func (s *DBStore) Close() error {
	return s.db.Close()
}
