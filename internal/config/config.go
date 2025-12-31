// Package config handles loading configuration from .gonerc.yaml files.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfigFileName is the default configuration file name.
const DefaultConfigFileName = ".gonerc.yaml"

// Config represents the complete configuration structure.
type Config struct {
	Ignore IgnoreConfig `yaml:"ignore"`
}

// IgnoreConfig holds all ignore rules.
type IgnoreConfig struct {
	// Domains to ignore (automatically includes subdomains).
	// Example: "example.com" will also match "www.example.com", "api.example.com".
	Domains []string `yaml:"domains"`

	// Patterns are glob patterns for URL matching.
	// Example: "*.local/*", "*/internal/*"
	Patterns []string `yaml:"patterns"`

	// Regex are regular expression patterns for URL matching.
	// Example: ".*\\.test$", ".*/v[0-9]+/draft/.*"
	Regex []string `yaml:"regex"`
}

// Load reads configuration from .gonerc.yaml in the current directory.
// Returns an empty config if the file doesn't exist (not an error).
// Returns an error only if the file exists but cannot be parsed.
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigFileName)
}

// LoadFrom reads configuration from a specific path.
// Returns an empty config if the file doesn't exist (not an error).
// Returns an error only if the file exists but cannot be parsed.
func LoadFrom(path string) (*Config, error) {
	// Start with empty config
	cfg := &Config{}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		// File not found is not an error - just return empty config
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// FindAndLoad searches for a config file starting from the given directory
// and walking up to parent directories until it finds one or reaches root.
// This allows project-specific configs to be found from subdirectories.
func FindAndLoad(startDir string) (*Config, error) {
	dir := startDir

	for {
		configPath := filepath.Join(dir, DefaultConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			// Found a config file
			return LoadFrom(configPath)
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no config found
			return &Config{}, nil
		}
		dir = parent
	}
}

// IsEmpty returns true if the config has no ignore rules defined.
func (c *Config) IsEmpty() bool {
	return len(c.Ignore.Domains) == 0 &&
		len(c.Ignore.Patterns) == 0 &&
		len(c.Ignore.Regex) == 0
}

// Merge combines another config into this one (additive).
// This is useful for merging CLI flags with file config.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}
	c.Ignore.Domains = append(c.Ignore.Domains, other.Ignore.Domains...)
	c.Ignore.Patterns = append(c.Ignore.Patterns, other.Ignore.Patterns...)
	c.Ignore.Regex = append(c.Ignore.Regex, other.Ignore.Regex...)
}
