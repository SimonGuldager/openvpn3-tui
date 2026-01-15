package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Profile represents a saved VPN configuration
type Profile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config holds the application configuration
type Config struct {
	Profiles []Profile `json:"profiles"`
}

// configDir returns the config directory path
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "openvpn3-tui"), nil
}

// configPath returns the full path to the config file
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk, returning empty config if not found
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Profiles: []Profile{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddProfile adds a new profile to the config
func (c *Config) AddProfile(name, path string) {
	c.Profiles = append(c.Profiles, Profile{Name: name, Path: path})
}

// RemoveProfile removes a profile by index
func (c *Config) RemoveProfile(index int) {
	if index >= 0 && index < len(c.Profiles) {
		c.Profiles = append(c.Profiles[:index], c.Profiles[index+1:]...)
	}
}

// ValidateProfiles checks if profile files exist and returns validity status
func (c *Config) ValidateProfiles() map[int]bool {
	valid := make(map[int]bool)
	for i, p := range c.Profiles {
		_, err := os.Stat(p.Path)
		valid[i] = err == nil
	}
	return valid
}
