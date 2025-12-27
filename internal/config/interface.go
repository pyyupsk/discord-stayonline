package config

type ConfigStore interface {
	Load() (*Configuration, error)
	Save(cfg *Configuration) error
}
