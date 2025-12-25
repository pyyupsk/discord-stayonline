package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pyyupsk/discord-stayonline/internal/config"
)

func TestConfigStoreLoadSave(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	store := config.NewStore(configPath)

	// Test loading non-existent file returns default config
	t.Run("load non-existent file returns default", func(t *testing.T) {
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		if len(cfg.Servers) != 0 {
			t.Errorf("expected empty servers, got %d", len(cfg.Servers))
		}
		if cfg.TOSAcknowledged {
			t.Error("expected TOSAcknowledged to be false")
		}
	})

	// Test save and load roundtrip
	t.Run("save and load roundtrip", func(t *testing.T) {
		cfg := &config.Configuration{
			Servers: []config.ServerEntry{
				{
					ID:             "test-1",
					GuildID:        "123456789012345678",
					ChannelID:      "234567890123456789",
					Status:         config.StatusOnline,
					ConnectOnStart: true,
					Priority:       1,
				},
				{
					ID:             "test-2",
					GuildID:        "987654321098765432",
					ChannelID:      "876543210987654321",
					Status:         config.StatusIdle,
					ConnectOnStart: false,
					Priority:       2,
				},
			},
			TOSAcknowledged: true,
		}

		if err := store.Save(cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := store.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if len(loaded.Servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(loaded.Servers))
		}
		if !loaded.TOSAcknowledged {
			t.Error("expected TOSAcknowledged to be true")
		}

		// Verify first server
		if loaded.Servers[0].ID != "test-1" {
			t.Errorf("expected server ID 'test-1', got '%s'", loaded.Servers[0].ID)
		}
		if loaded.Servers[0].GuildID != "123456789012345678" {
			t.Errorf("expected guild ID '123456789012345678', got '%s'", loaded.Servers[0].GuildID)
		}
		if loaded.Servers[0].Status != config.StatusOnline {
			t.Errorf("expected status 'online', got '%s'", loaded.Servers[0].Status)
		}
	})

	// Test atomic write (temp file is cleaned up)
	t.Run("atomic write cleans up temp file", func(t *testing.T) {
		tmpPath := configPath + ".tmp"
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Error("temp file should not exist after save")
		}
	})

	// Test file permissions
	t.Run("saved file has correct permissions", func(t *testing.T) {
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		// File should be readable/writable by owner only (0600)
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("expected permissions 0600, got %o", perm)
		}
	})
}

func TestConfigStoreEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config for empty file")
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("expected empty servers, got %d", len(cfg.Servers))
	}
}

func TestConfigStoreInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create file with invalid JSON
	if err := os.WriteFile(configPath, []byte("{invalid json}"), 0600); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	store := config.NewStore(configPath)
	_, err := store.Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestConfigStoreSaveValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	store := config.NewStore(configPath)

	// Test saving config with too many servers
	t.Run("save fails with too many servers", func(t *testing.T) {
		cfg := &config.Configuration{
			Servers: make([]config.ServerEntry, 16), // One more than max
		}
		// Fill with valid entries
		for i := range cfg.Servers {
			cfg.Servers[i] = config.ServerEntry{
				ID:             "test-" + string(rune('A'+i)),
				GuildID:        "123456789012345678",
				ChannelID:      "234567890123456789",
				Status:         config.StatusOnline,
				ConnectOnStart: true,
				Priority:       1,
			}
		}

		err := store.Save(cfg)
		if err != config.ErrTooManyServers {
			t.Errorf("expected ErrTooManyServers, got %v", err)
		}
	})

	// Test saving config with invalid server entry
	t.Run("save fails with invalid server entry", func(t *testing.T) {
		cfg := &config.Configuration{
			Servers: []config.ServerEntry{
				{
					ID:        "",
					GuildID:   "123456789012345678",
					ChannelID: "234567890123456789",
					Status:    config.StatusOnline,
					Priority:  1,
				},
			},
		}

		err := store.Save(cfg)
		if err != config.ErrEmptyID {
			t.Errorf("expected ErrEmptyID, got %v", err)
		}
	})
}

func TestServerEntryValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   config.ServerEntry
		wantErr error
	}{
		{
			name: "valid entry",
			entry: config.ServerEntry{
				ID:             "test-1",
				GuildID:        "123456789012345678",
				ChannelID:      "234567890123456789",
				Status:         config.StatusOnline,
				ConnectOnStart: true,
				Priority:       1,
			},
			wantErr: nil,
		},
		{
			name: "empty ID",
			entry: config.ServerEntry{
				ID:        "",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    config.StatusOnline,
				Priority:  1,
			},
			wantErr: config.ErrEmptyID,
		},
		{
			name: "empty guild ID",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "",
				ChannelID: "234567890123456789",
				Status:    config.StatusOnline,
				Priority:  1,
			},
			wantErr: config.ErrEmptyGuildID,
		},
		{
			name: "empty channel ID",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "",
				Status:    config.StatusOnline,
				Priority:  1,
			},
			wantErr: config.ErrEmptyChannelID,
		},
		{
			name: "invalid status",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    "invalid",
				Priority:  1,
			},
			wantErr: config.ErrInvalidStatus,
		},
		{
			name: "zero priority",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    config.StatusOnline,
				Priority:  0,
			},
			wantErr: config.ErrInvalidPriority,
		},
		{
			name: "negative priority",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    config.StatusOnline,
				Priority:  -1,
			},
			wantErr: config.ErrInvalidPriority,
		},
		{
			name: "idle status is valid",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    config.StatusIdle,
				Priority:  1,
			},
			wantErr: nil,
		},
		{
			name: "dnd status is valid",
			entry: config.ServerEntry{
				ID:        "test-1",
				GuildID:   "123456789012345678",
				ChannelID: "234567890123456789",
				Status:    config.StatusDND,
				Priority:  1,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigurationJSONFormat(t *testing.T) {
	cfg := &config.Configuration{
		Servers: []config.ServerEntry{
			{
				ID:             "test-1",
				GuildID:        "123456789012345678",
				ChannelID:      "234567890123456789",
				Status:         config.StatusOnline,
				ConnectOnStart: true,
				Priority:       1,
			},
		},
		TOSAcknowledged: true,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}

	// Verify JSON field names (snake_case)
	jsonStr := string(data)
	expectedFields := []string{
		`"servers"`,
		`"tos_acknowledged"`,
		`"id"`,
		`"guild_id"`,
		`"channel_id"`,
		`"status"`,
		`"connect_on_start"`,
		`"priority"`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain field %s", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
