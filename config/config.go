package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config represents the application configuration.
type Config struct {
	GitHubToken    string `json:"github_token"`
	GitHubUsername string `json:"github_username"`
	ClientID       string `json:"client_id"`
}

// GetConfigPath returns the OS-specific path where the configuration is stored.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "gravitycli")
	return filepath.Join(appDir, "config.json"), nil
}

// Load loads the configuration from disk.
// If the file does not exist, it returns a blank config instead of an error.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to disk, creating any parent folders.
func Save(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// IsAuthenticated checks if a token exists in the config.
func (c *Config) IsAuthenticated() bool {
	return c.GitHubToken != ""
}
