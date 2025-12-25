package config

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

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

	// Create logs table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id SERIAL PRIMARY KEY,
			level VARCHAR(10) NOT NULL,
			message TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Create index for log filtering and cleanup
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
	`)
	if err != nil {
		return err
	}

	// Create sessions table for Discord Gateway session resumption
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			server_id VARCHAR(32) PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
			session_id VARCHAR(64) NOT NULL,
			sequence INTEGER NOT NULL DEFAULT 0,
			resume_url VARCHAR(255) NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
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

// oldConfigData represents the old JSONB configuration structure.
type oldConfigData struct {
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

// migrateFromOldSchema migrates data from the old JSONB configuration table.
func (s *DBStore) migrateFromOldSchema() error {
	if s.shouldSkipMigration() {
		return nil
	}

	oldConfig, ok := s.loadOldConfig()
	if !ok {
		return nil
	}

	if err := s.migrateSettings(oldConfig); err != nil {
		return err
	}

	return s.migrateServers(oldConfig)
}

// shouldSkipMigration checks if migration should be skipped.
func (s *DBStore) shouldSkipMigration() bool {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'configuration'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		return true
	}

	var settingsCount, serversCount int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settingsCount)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM servers`).Scan(&serversCount)

	return settingsCount > 0 || serversCount > 0
}

// loadOldConfig loads and parses the old configuration.
func (s *DBStore) loadOldConfig() (*oldConfigData, bool) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM configuration WHERE id = 1").Scan(&data)
	if err != nil {
		return nil, false
	}

	var oldConfig oldConfigData
	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return nil, false
	}

	return &oldConfig, true
}

// migrateSettings migrates settings from old config.
func (s *DBStore) migrateSettings(oldConfig *oldConfigData) error {
	status := oldConfig.Status
	if status == "" {
		status = "online"
	}
	_, err := s.db.Exec(`
		INSERT INTO settings (id, status, tos_acknowledged)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			tos_acknowledged = EXCLUDED.tos_acknowledged
	`, status, oldConfig.TOSAcknowledged)
	return err
}

// migrateServers migrates servers from old config.
func (s *DBStore) migrateServers(oldConfig *oldConfigData) error {
	for _, srv := range oldConfig.Servers {
		priority := max(srv.Priority, 1)
		_, err := s.db.Exec(`
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

	if err := cfg.Validate(); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.saveSettings(tx, cfg); err != nil {
		return err
	}

	if err = s.syncServers(tx, cfg.Servers); err != nil {
		return err
	}

	return tx.Commit()
}

// saveSettings saves global settings to the database.
func (s *DBStore) saveSettings(tx *sql.Tx, cfg *Configuration) error {
	status := string(cfg.Status)
	if status == "" {
		status = "online"
	}
	_, err := tx.Exec(`
		INSERT INTO settings (id, status, tos_acknowledged, updated_at)
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			tos_acknowledged = EXCLUDED.tos_acknowledged,
			updated_at = NOW()
	`, status, cfg.TOSAcknowledged)
	return err
}

// syncServers synchronizes servers in the database with the provided list.
func (s *DBStore) syncServers(tx *sql.Tx, servers []ServerEntry) error {
	existingIDs, err := s.getExistingServerIDs(tx)
	if err != nil {
		return err
	}

	newIDs := make(map[string]bool)
	for _, srv := range servers {
		newIDs[srv.ID] = true
	}

	if err := s.deleteRemovedServers(tx, existingIDs, newIDs); err != nil {
		return err
	}

	return s.upsertServers(tx, servers)
}

// getExistingServerIDs returns a set of existing server IDs.
func (s *DBStore) getExistingServerIDs(tx *sql.Tx) (map[string]bool, error) {
	existingIDs := make(map[string]bool)
	rows, err := tx.Query(`SELECT id FROM servers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		existingIDs[id] = true
	}
	return existingIDs, rows.Err()
}

// deleteRemovedServers deletes servers that are no longer in the config.
func (s *DBStore) deleteRemovedServers(tx *sql.Tx, existingIDs, newIDs map[string]bool) error {
	for id := range existingIDs {
		if !newIDs[id] {
			if _, err := tx.Exec(`DELETE FROM servers WHERE id = $1`, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// upsertServers inserts or updates servers in the database.
func (s *DBStore) upsertServers(tx *sql.Tx, servers []ServerEntry) error {
	for _, srv := range servers {
		_, err := tx.Exec(`
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
	return nil
}

// Close closes the database connection.
func (s *DBStore) Close() error {
	return s.db.Close()
}

// LogEntry represents a stored log entry.
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// MaxLogEntries is the maximum number of log entries to keep in the database.
const MaxLogEntries = 1000

// AddLog inserts a new log entry and trims old entries if needed.
func (s *DBStore) AddLog(level, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO logs (level, message) VALUES ($1, $2)
	`, level, message)
	if err != nil {
		return err
	}

	// Trim old logs to keep only the last MaxLogEntries
	_, err = s.db.Exec(`
		DELETE FROM logs WHERE id NOT IN (
			SELECT id FROM logs ORDER BY created_at DESC LIMIT $1
		)
	`, MaxLogEntries)

	return err
}

// GetLogs retrieves log entries, optionally filtered by level.
// Returns logs ordered from oldest to newest.
func (s *DBStore) GetLogs(level string) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error

	if level == "" {
		rows, err = s.db.Query(`
			SELECT level, message, created_at FROM logs
			ORDER BY created_at ASC
			LIMIT $1
		`, MaxLogEntries)
	} else {
		rows, err = s.db.Query(`
			SELECT level, message, created_at FROM logs
			WHERE level = $1
			ORDER BY created_at ASC
			LIMIT $2
		`, level, MaxLogEntries)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var log LogEntry
		if err := rows.Scan(&log.Level, &log.Message, &log.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ClearLogs removes all log entries from the database.
func (s *DBStore) ClearLogs() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM logs`)
	return err
}

// SessionState holds Discord Gateway session data for resumption.
type SessionState struct {
	ServerID  string `json:"server_id"`
	SessionID string `json:"session_id"`
	Sequence  int    `json:"sequence"`
	ResumeURL string `json:"resume_url"`
}

// SaveSession persists session state for later resumption.
func (s *DBStore) SaveSession(state SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO sessions (server_id, session_id, sequence, resume_url, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (server_id) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			sequence = EXCLUDED.sequence,
			resume_url = EXCLUDED.resume_url,
			updated_at = NOW()
	`, state.ServerID, state.SessionID, state.Sequence, state.ResumeURL)
	return err
}

// LoadSession retrieves saved session state for resumption.
func (s *DBStore) LoadSession(serverID string) (*SessionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var state SessionState
	err := s.db.QueryRow(`
		SELECT server_id, session_id, sequence, resume_url FROM sessions
		WHERE server_id = $1
	`, serverID).Scan(&state.ServerID, &state.SessionID, &state.Sequence, &state.ResumeURL)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteSession removes session state.
func (s *DBStore) DeleteSession(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM sessions WHERE server_id = $1`, serverID)
	return err
}

// UpdateSessionSequence updates just the sequence number for a session.
func (s *DBStore) UpdateSessionSequence(serverID string, sequence int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE sessions SET sequence = $1, updated_at = NOW()
		WHERE server_id = $2
	`, sequence, serverID)
	return err
}
