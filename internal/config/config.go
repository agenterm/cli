package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultRelayURL = "https://push.agenterm.app"

type Config struct {
	RelayURL string `json:"relay_url"`
	PushKey  string `json:"push_key"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agenterm"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ConfigPath returns the path to ~/.agenterm/config.json.
func ConfigPath() (string, error) {
	return configPath()
}

// Load reads config from ~/.agenterm/config.json.
// Returns default config if the file does not exist.
func Load() (*Config, error) {
	cfg := &Config{RelayURL: DefaultRelayURL}

	p, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.RelayURL == "" {
		cfg.RelayURL = DefaultRelayURL
	}
	return cfg, nil
}

// Save writes config to ~/.agenterm/config.json, creating the directory if needed.
func (c *Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
