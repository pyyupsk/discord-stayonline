package config

import (
	"database/sql"
	"encoding/json"
	"sync"

	_ "github.com/lib/pq"
)

// DBStore handles configuration persistence using PostgreSQL with normalized tables.
// Schema:
//   - settings: global settings (status, tos_acknowledged)
//   - servers: individual server entries for horizontal scaling
type DBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDBStore creates a new database-backed configuration store.
// It automatically creates the required tables if they don't exist.
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

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate creates the tables if they don't exist and migrates from old schema.
func (s *DBStore) migrate() error {
	// Create settings table (global config)
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY DEFAULT 1,
			status VARCHAR(10) NOT NULL DEFAULT 'online',
			tos_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			CONSTRAINT single_settings_row CHECK (id = 1)
		)
	`)
	if err != nil {
		return err
	}

	// Create servers table (individual entries for horizontal scaling)
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id VARCHAR(32) PRIMARY KEY,
			guild_id VARCHAR(20) NOT NULL,
			guild_name VARCHAR(100),
			channel_id VARCHAR(20) NOT NULL,
			channel_name VARCHAR(100),
			connect_on_start BOOLEAN NOT NULL DEFAULT FALSE,
			priority INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes for faster lookups and potential sharding
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_servers_guild_id ON servers(guild_id);
		CREATE INDEX IF NOT EXISTS idx_servers_priority ON servers(priority);
	`)
	if err != nil {
		return err
	}

	// Migrate from old schema if exists
	if err := s.migrateFromOldSchema(); err != nil {
		return err
	}

	// Ensure settings row exists
	_, err = s.db.Exec(`
		INSERT INTO settings (id, status, tos_acknowledged)
		VALUES (1, 'online', FALSE)
		ON CONFLICT (id) DO NOTHING
	`)

	return err
}

// migrateFromOldSchema migrates data from the old JSONB configuration table.
func (s *DBStore) migrateFromOldSchema() error {
	// Check if old table exists
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'configuration'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		return nil // No old table, nothing to migrate
	}

	// Check if already migrated (settings has data)
	var settingsCount int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settingsCount)
	if err != nil {
		return nil
	}

	// Check if servers table has data
	var serversCount int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM servers`).Scan(&serversCount)
	if err != nil {
		return nil
	}

	// If new tables already have data, skip migration
	if settingsCount > 0 || serversCount > 0 {
		return nil
	}

	// Read old configuration
	var data []byte
	err = s.db.QueryRow("SELECT data FROM configuration WHERE id = 1").Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No data to migrate
		}
		return nil // Ignore errors, old table might have different structure
	}

	// Parse old config
	var oldConfig struct {
		Servers []struct {
			ID             string `json:"id"`
			GuildID        string `json:"guild_id"`
			GuildName      string `json:"guild_name"`
			ChannelID      string `json:"channel_id"`
			ChannelName    string `json:"channel_name"`
			ConnectOnStart bool   `json:"connect_on_start"`
			Priority       int    `json:"priority"`
		} `json:"servers"`
		Status          string `json:"status"`
		TOSAcknowledged bool   `json:"tos_acknowledged"`
	}

	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return nil // Ignore parse errors
	}

	// Migrate settings
	status := oldConfig.Status
	if status == "" {
		status = "online"
	}
	_, err = s.db.Exec(`
		INSERT INTO settings (id, status, tos_acknowledged)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			tos_acknowledged = EXCLUDED.tos_acknowledged
	`, status, oldConfig.TOSAcknowledged)
	if err != nil {
		return err
	}

	// Migrate servers
	for _, srv := range oldConfig.Servers {
		priority := srv.Priority
		if priority < 1 {
			priority = 1
		}
		_, err = s.db.Exec(`
			INSERT INTO servers (id, guild_id, guild_name, channel_id, channel_name, connect_on_start, priority)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`, srv.ID, srv.GuildID, srv.GuildName, srv.ChannelID, srv.ChannelName, srv.ConnectOnStart, priority)
		if err != nil {
			return err
		}
	}

	return nil
}

// Load reads the configuration from the database.
// Returns a default configuration if no record exists.
func (s *DBStore) Load() (*Configuration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := &Configuration{
		Servers: []ServerEntry{},
		Status:  StatusOnline,
	}

	// Load settings
	var status string
	err := s.db.QueryRow(`
		SELECT status, tos_acknowledged FROM settings WHERE id = 1
	`).Scan(&status, &cfg.TOSAcknowledged)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if status != "" {
		cfg.Status = Status(status)
	}

	// Load servers ordered by priority
	rows, err := s.db.Query(`
		SELECT id, guild_id, COALESCE(guild_name, ''), channel_id, COALESCE(channel_name, ''), connect_on_start, priority
		FROM servers
		ORDER BY priority ASC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var srv ServerEntry
		err := rows.Scan(&srv.ID, &srv.GuildID, &srv.GuildName, &srv.ChannelID, &srv.ChannelName, &srv.ConnectOnStart, &srv.Priority)
		if err != nil {
			return nil, err
		}
		cfg.Servers = append(cfg.Servers, srv)
	}

	return cfg, rows.Err()
}

// Save writes the configuration to the database.
// Uses transactions for consistency across tables.
func (s *DBStore) Save(cfg *Configuration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update settings
	status := string(cfg.Status)
	if status == "" {
		status = "online"
	}
	_, err = tx.Exec(`
		INSERT INTO settings (id, status, tos_acknowledged, updated_at)
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			tos_acknowledged = EXCLUDED.tos_acknowledged,
			updated_at = NOW()
	`, status, cfg.TOSAcknowledged)
	if err != nil {
		return err
	}

	// Get existing server IDs
	existingIDs := make(map[string]bool)
	rows, err := tx.Query(`SELECT id FROM servers`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existingIDs[id] = true
	}
	rows.Close()

	// Track which IDs are in the new config
	newIDs := make(map[string]bool)
	for _, srv := range cfg.Servers {
		newIDs[srv.ID] = true
	}

	// Delete servers not in new config
	for id := range existingIDs {
		if !newIDs[id] {
			_, err = tx.Exec(`DELETE FROM servers WHERE id = $1`, id)
			if err != nil {
				return err
			}
		}
	}

	// Upsert servers
	for _, srv := range cfg.Servers {
		_, err = tx.Exec(`
			INSERT INTO servers (id, guild_id, guild_name, channel_id, channel_name, connect_on_start, priority, updated_at)
			VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7, NOW())
			ON CONFLICT (id) DO UPDATE SET
				guild_id = EXCLUDED.guild_id,
				guild_name = EXCLUDED.guild_name,
				channel_id = EXCLUDED.channel_id,
				channel_name = EXCLUDED.channel_name,
				connect_on_start = EXCLUDED.connect_on_start,
				priority = EXCLUDED.priority,
				updated_at = NOW()
		`, srv.ID, srv.GuildID, srv.GuildName, srv.ChannelID, srv.ChannelName, srv.ConnectOnStart, srv.Priority)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Close closes the database connection.
func (s *DBStore) Close() error {
	return s.db.Close()
}
