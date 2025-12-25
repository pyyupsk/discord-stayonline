package config

// ConfigStore is the interface for configuration storage backends.
type ConfigStore interface {
	// Load reads the configuration from storage.
	// Returns a default configuration if none exists.
	Load() (*Configuration, error)

	// Save writes the configuration to storage.
	Save(cfg *Configuration) error
}
