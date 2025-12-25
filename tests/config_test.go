package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pyyupsk/discord-stayonline/internal/config"
)

const (
	testConfigFile = "config.json"
	testServerID1  = "test-1"
	testGuildID1   = "123456789012345678"
	testChannelID1 = "234567890123456789"
	errLoadFormat  = "Load() error = %v"
	errSaveFormat  = "Save() error = %v"
)

func TestConfigStoreLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, testConfigFile)
	store := config.NewStore(configPath)

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf(errLoadFormat, err)
	}

	assertDefaultConfig(t, cfg)
}

func TestConfigStoreSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, testConfigFile)
	store := config.NewStore(configPath)

	cfg := createTestConfig()

	if err := store.Save(cfg); err != nil {
		t.Fatalf(errSaveFormat, err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf(errLoadFormat, err)
	}

	assertLoadedConfig(t, loaded)
}

func TestConfigStoreAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, testConfigFile)
	store := config.NewStore(configPath)

	cfg := createTestConfig()
	if err := store.Save(cfg); err != nil {
		t.Fatalf(errSaveFormat, err)
	}

	tmpPath := configPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after save")
	}
}

func TestConfigStoreFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, testConfigFile)
	store := config.NewStore(configPath)

	cfg := createTestConfig()
	if err := store.Save(cfg); err != nil {
		t.Fatalf(errSaveFormat, err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}
}

// assertDefaultConfig verifies that cfg has default values.
func assertDefaultConfig(t *testing.T, cfg *config.Configuration) {
	t.Helper()
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("expected empty servers, got %d", len(cfg.Servers))
	}
	if cfg.TOSAcknowledged {
		t.Error("expected TOSAcknowledged to be false")
	}
	if cfg.Status != config.StatusOnline {
		t.Errorf("expected default status 'online', got '%s'", cfg.Status)
	}
}

// assertLoadedConfig verifies loaded config matches expected values.
func assertLoadedConfig(t *testing.T, loaded *config.Configuration) {
	t.Helper()
	if len(loaded.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(loaded.Servers))
	}
	if !loaded.TOSAcknowledged {
		t.Error("expected TOSAcknowledged to be true")
	}
	if loaded.Status != config.StatusIdle {
		t.Errorf("expected status 'idle', got '%s'", loaded.Status)
	}
	if loaded.Servers[0].ID != testServerID1 {
		t.Errorf("expected server ID '%s', got '%s'", testServerID1, loaded.Servers[0].ID)
	}
	if loaded.Servers[0].GuildID != testGuildID1 {
		t.Errorf("expected guild ID '%s', got '%s'", testGuildID1, loaded.Servers[0].GuildID)
	}
}

// createTestConfig creates a configuration for testing.
func createTestConfig() *config.Configuration {
	return &config.Configuration{
		Servers: []config.ServerEntry{
			{
				ID:             testServerID1,
				GuildID:        testGuildID1,
				ChannelID:      testChannelID1,
				ConnectOnStart: true,
				Priority:       1,
			},
			{
				ID:             "test-2",
				GuildID:        "987654321098765432",
				ChannelID:      "876543210987654321",
				ConnectOnStart: false,
				Priority:       2,
			},
		},
		Status:          config.StatusIdle,
		TOSAcknowledged: true,
	}
}

func TestConfigStoreEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, testConfigFile)

	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf(errLoadFormat, err)
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
	configPath := filepath.Join(tmpDir, testConfigFile)

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
	configPath := filepath.Join(tmpDir, testConfigFile)
	store := config.NewStore(configPath)

	// Test saving config with too many servers
	t.Run("save fails with too many servers", func(t *testing.T) {
		cfg := &config.Configuration{
			Servers: make([]config.ServerEntry, config.MaxServerEntries+1), // One more than max
			Status:  config.StatusOnline,
		}
		// Fill with valid entries
		for i := range cfg.Servers {
			cfg.Servers[i] = config.ServerEntry{
				ID:             fmt.Sprintf("test-%d", i),
				GuildID:        "123456789012345678",
				ChannelID:      "234567890123456789",
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
					GuildID:   testGuildID1,
					ChannelID: testChannelID1,
					Priority:  1,
				},
			},
			Status: config.StatusOnline,
		}

		err := store.Save(cfg)
		if err != config.ErrEmptyID {
			t.Errorf("expected ErrEmptyID, got %v", err)
		}
	})

	// Test saving config with invalid status
	t.Run("save fails with invalid status", func(t *testing.T) {
		cfg := &config.Configuration{
			Servers: []config.ServerEntry{},
			Status:  "invalid",
		}

		err := store.Save(cfg)
		if err != config.ErrInvalidStatus {
			t.Errorf("expected ErrInvalidStatus, got %v", err)
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
				ID:             testServerID1,
				GuildID:        testGuildID1,
				ChannelID:      testChannelID1,
				ConnectOnStart: true,
				Priority:       1,
			},
			wantErr: nil,
		},
		{
			name: "empty ID",
			entry: config.ServerEntry{
				ID:        "",
				GuildID:   testGuildID1,
				ChannelID: testChannelID1,
				Priority:  1,
			},
			wantErr: config.ErrEmptyID,
		},
		{
			name: "empty guild ID",
			entry: config.ServerEntry{
				ID:        testServerID1,
				GuildID:   "",
				ChannelID: testChannelID1,
				Priority:  1,
			},
			wantErr: config.ErrEmptyGuildID,
		},
		{
			name: "empty channel ID",
			entry: config.ServerEntry{
				ID:        testServerID1,
				GuildID:   testGuildID1,
				ChannelID: "",
				Priority:  1,
			},
			wantErr: config.ErrEmptyChannelID,
		},
		{
			name: "zero priority",
			entry: config.ServerEntry{
				ID:        testServerID1,
				GuildID:   testGuildID1,
				ChannelID: testChannelID1,
				Priority:  0,
			},
			wantErr: config.ErrInvalidPriority,
		},
		{
			name: "negative priority",
			entry: config.ServerEntry{
				ID:        testServerID1,
				GuildID:   testGuildID1,
				ChannelID: testChannelID1,
				Priority:  -1,
			},
			wantErr: config.ErrInvalidPriority,
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
				ID:             testServerID1,
				GuildID:        testGuildID1,
				ChannelID:      testChannelID1,
				ConnectOnStart: true,
				Priority:       1,
			},
		},
		Status:          config.StatusOnline,
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
		`"status"`,
		`"id"`,
		`"guild_id"`,
		`"channel_id"`,
		`"connect_on_start"`,
		`"priority"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON should contain field %s", field)
		}
	}
}
