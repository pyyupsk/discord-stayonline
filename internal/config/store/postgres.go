package store

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Postgres handles configuration persistence using PostgreSQL with GORM.
type Postgres struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewPostgres creates a new database-backed configuration store.
// It automatically creates the required tables if they don't exist.
func NewPostgres(databaseURL string) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	store := &Postgres{db: db}

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate runs GORM auto-migration and handles schema evolution.
func (s *Postgres) migrate() error {
	// Auto-migrate all models
	if err := s.db.AutoMigrate(&Setting{}, &Server{}, &Log{}, &Session{}); err != nil {
		return err
	}

	// Add CHECK constraint for single settings row (GORM doesn't support this directly)
	s.db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'single_settings_row'
			) THEN
				ALTER TABLE settings ADD CONSTRAINT single_settings_row CHECK (id = 1);
			END IF;
		END $$;
	`)

	// Add foreign key constraint for sessions (GORM doesn't auto-create this for non-embedded relations)
	s.db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_sessions_server'
			) THEN
				ALTER TABLE sessions ADD CONSTRAINT fk_sessions_server
				FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE;
			END IF;
		END $$;
	`)

	// Migrate from old schema if exists
	if err := s.migrateFromOldSchema(); err != nil {
		return err
	}

	// Ensure settings row exists
	var count int64
	s.db.Model(&Setting{}).Count(&count)
	if count == 0 {
		s.db.Create(&Setting{
			ID:              1,
			Status:          "online",
			TOSAcknowledged: false,
		})
	}

	return nil
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
func (s *Postgres) migrateFromOldSchema() error {
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
func (s *Postgres) shouldSkipMigration() bool {
	var exists bool
	s.db.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'configuration'
		)
	`).Scan(&exists)

	if !exists {
		return true
	}

	var settingsCount, serversCount int64
	s.db.Model(&Setting{}).Count(&settingsCount)
	s.db.Model(&Server{}).Count(&serversCount)

	return settingsCount > 0 || serversCount > 0
}

// loadOldConfig loads and parses the old configuration.
func (s *Postgres) loadOldConfig() (*oldConfigData, bool) {
	var data []byte
	result := s.db.Raw("SELECT data FROM configuration WHERE id = 1").Scan(&data)
	if result.Error != nil || len(data) == 0 {
		return nil, false
	}

	var oldConfig oldConfigData
	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return nil, false
	}

	return &oldConfig, true
}

// migrateSettings migrates settings from old config.
func (s *Postgres) migrateSettings(oldConfig *oldConfigData) error {
	status := oldConfig.Status
	if status == "" {
		status = "online"
	}
	return s.db.Save(&Setting{
		ID:              1,
		Status:          status,
		TOSAcknowledged: oldConfig.TOSAcknowledged,
	}).Error
}

// migrateServers migrates servers from old config.
func (s *Postgres) migrateServers(oldConfig *oldConfigData) error {
	for _, srv := range oldConfig.Servers {
		priority := max(srv.Priority, 1)
		server := Server{
			ID:             srv.ID,
			GuildID:        srv.GuildID,
			GuildName:      stringToPtr(srv.GuildName),
			ChannelID:      srv.ChannelID,
			ChannelName:    stringToPtr(srv.ChannelName),
			ConnectOnStart: srv.ConnectOnStart,
			Priority:       priority,
		}
		// Use FirstOrCreate to avoid overwriting existing data
		if err := s.db.FirstOrCreate(&server, Server{ID: srv.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Load reads the configuration from the database.
// Returns a default configuration if no record exists.
func (s *Postgres) Load() (*config.Configuration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := &config.Configuration{
		Servers: []config.ServerEntry{},
		Status:  config.StatusOnline,
	}

	// Load settings
	var setting Setting
	if err := s.db.First(&setting).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if setting.Status != "" {
		cfg.Status = config.Status(setting.Status)
	}
	cfg.TOSAcknowledged = setting.TOSAcknowledged

	// Load servers ordered by priority
	var servers []Server
	if err := s.db.Order("priority ASC, created_at ASC").Find(&servers).Error; err != nil {
		return nil, err
	}

	for _, srv := range servers {
		cfg.Servers = append(cfg.Servers, config.ServerEntry{
			ID:             srv.ID,
			GuildID:        srv.GuildID,
			GuildName:      ptrToString(srv.GuildName),
			GuildIcon:      ptrToString(srv.GuildIcon),
			ChannelID:      srv.ChannelID,
			ChannelName:    ptrToString(srv.ChannelName),
			ConnectOnStart: srv.ConnectOnStart,
			Priority:       srv.Priority,
		})
	}

	return cfg, nil
}

// ptrToString safely converts *string to string.
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringToPtr converts non-empty string to *string.
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Save writes the configuration to the database.
// Uses transactions for consistency across tables.
func (s *Postgres) Save(cfg *config.Configuration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := cfg.Validate(); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Save settings
		status := string(cfg.Status)
		if status == "" {
			status = "online"
		}
		if err := tx.Save(&Setting{
			ID:              1,
			Status:          status,
			TOSAcknowledged: cfg.TOSAcknowledged,
		}).Error; err != nil {
			return err
		}

		// Sync servers
		return s.syncServers(tx, cfg.Servers)
	})
}

// syncServers synchronizes servers in the database with the provided list.
func (s *Postgres) syncServers(tx *gorm.DB, servers []config.ServerEntry) error {
	// Get existing server IDs
	var existingIDs []string
	if err := tx.Model(&Server{}).Pluck("id", &existingIDs).Error; err != nil {
		return err
	}

	// Build map of new IDs
	newIDs := make(map[string]bool)
	for _, srv := range servers {
		newIDs[srv.ID] = true
	}

	// Delete removed servers
	for _, id := range existingIDs {
		if !newIDs[id] {
			if err := tx.Delete(&Server{}, "id = ?", id).Error; err != nil {
				return err
			}
		}
	}

	// Upsert servers
	for _, srv := range servers {
		server := Server{
			ID:             srv.ID,
			GuildID:        srv.GuildID,
			GuildName:      stringToPtr(srv.GuildName),
			GuildIcon:      stringToPtr(srv.GuildIcon),
			ChannelID:      srv.ChannelID,
			ChannelName:    stringToPtr(srv.ChannelName),
			ConnectOnStart: srv.ConnectOnStart,
			Priority:       srv.Priority,
		}
		if err := tx.Save(&server).Error; err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database connection.
func (s *Postgres) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// LogEntry represents a stored log entry for API responses.
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// MaxLogEntries is the maximum number of log entries to keep in the database.
const MaxLogEntries = 1000

// whereServerID is the query condition for server_id lookups.
const whereServerID = "server_id = ?"

// AddLog inserts a new log entry and trims old entries if needed.
func (s *Postgres) AddLog(level, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Create(&Log{
		Level:   level,
		Message: message,
	}).Error; err != nil {
		return err
	}

	// Trim old logs using subquery
	s.db.Exec(`
		DELETE FROM logs WHERE id NOT IN (
			SELECT id FROM logs ORDER BY created_at DESC LIMIT ?
		)
	`, MaxLogEntries)

	return nil
}

// GetLogs retrieves log entries, optionally filtered by level.
// Returns logs ordered from oldest to newest.
func (s *Postgres) GetLogs(level string) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var logs []Log
	query := s.db.Order("created_at ASC").Limit(MaxLogEntries)

	if level != "" {
		query = query.Where("level = ?", level)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	result := make([]LogEntry, len(logs))
	for i, log := range logs {
		result[i] = LogEntry{
			Level:     log.Level,
			Message:   log.Message,
			Timestamp: log.CreatedAt,
		}
	}

	return result, nil
}

// ClearLogs removes all log entries from the database.
func (s *Postgres) ClearLogs() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Where("1 = 1").Delete(&Log{}).Error
}

// SaveSession persists session state for later resumption.
func (s *Postgres) SaveSession(state config.SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Save(&Session{
		ServerID:  state.ServerID,
		SessionID: state.SessionID,
		Sequence:  state.Sequence,
		ResumeURL: state.ResumeURL,
	}).Error
}

// LoadSession retrieves saved session state for resumption.
func (s *Postgres) LoadSession(serverID string) (*config.SessionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var session Session
	if err := s.db.First(&session, whereServerID, serverID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &config.SessionState{
		ServerID:  session.ServerID,
		SessionID: session.SessionID,
		Sequence:  session.Sequence,
		ResumeURL: session.ResumeURL,
	}, nil
}

// DeleteSession removes session state.
func (s *Postgres) DeleteSession(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Delete(&Session{}, whereServerID, serverID).Error
}

// UpdateSessionSequence updates just the sequence number for a session.
func (s *Postgres) UpdateSessionSequence(serverID string, sequence int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Model(&Session{}).
		Where(whereServerID, serverID).
		Update("sequence", sequence).Error
}
