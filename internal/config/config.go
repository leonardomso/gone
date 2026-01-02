// Package config handles loading configuration from .gonerc.yaml files.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"
)

// DefaultConfigFileName is the default configuration file name.
const DefaultConfigFileName = ".gonerc.yaml"

// Config represents the complete configuration structure.
type Config struct {
	// Types specifies which file types to scan.
	// Supported: md, json, yaml, toml, xml
	// If empty, defaults to ["md"] at runtime.
	Types []string `yaml:"types"`

	// Scan holds scanner configuration.
	Scan ScanConfig `yaml:"scan"`

	// Check holds checker configuration.
	Check CheckConfig `yaml:"check"`

	// Output holds output preferences.
	Output OutputConfig `yaml:"output"`

	// Ignore holds URL ignore rules (backwards compatible).
	Ignore IgnoreConfig `yaml:"ignore"`
}

// ScanConfig holds scanner settings for file discovery.
type ScanConfig struct {
	// Include specifies glob patterns for paths to include.
	// If empty, all files matching the types are included.
	// Example: ["docs/**", "README.md"]
	Include []string `yaml:"include"`

	// Exclude specifies glob patterns for paths to exclude.
	// Example: ["node_modules/**", "vendor/**", "**/testdata/**"]
	Exclude []string `yaml:"exclude"`
}

// CheckConfig holds checker settings for URL validation.
type CheckConfig struct {
	// Concurrency is the number of concurrent workers.
	// Default: 50 (set at runtime if 0)
	Concurrency int `yaml:"concurrency"`

	// Timeout is the request timeout in seconds.
	// Default: 5 (set at runtime if 0)
	Timeout int `yaml:"timeout"`

	// Retries is the number of retry attempts for failed requests.
	// Default: 1 (set at runtime if 0)
	Retries int `yaml:"retries"`

	// Strict fails on malformed files instead of skipping them.
	// Default: false
	Strict bool `yaml:"strict"`
}

// OutputConfig holds output preferences for the check command.
type OutputConfig struct {
	// Format specifies the default output format.
	// Valid: json, yaml, xml, junit, markdown
	// Empty means text output to stdout.
	Format string `yaml:"format"`

	// ShowAlive shows alive links in output.
	// Default: false
	ShowAlive bool `yaml:"showAlive"`

	// ShowWarnings shows warning links (redirects, blocked) in output.
	// Default: true (set at runtime)
	ShowWarnings *bool `yaml:"showWarnings"`

	// ShowDead shows dead links and errors in output.
	// Default: true (set at runtime)
	ShowDead *bool `yaml:"showDead"`

	// ShowStats shows performance statistics.
	// Default: false
	ShowStats bool `yaml:"showStats"`
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

// validOutputFormats lists all valid output format values.
var validOutputFormats = []string{"json", "yaml", "xml", "junit", "markdown"}

// validFileTypes lists all valid file type values.
// This is duplicated here to avoid circular dependency with parser package.
var validFileTypes = []string{"md", "json", "yaml", "toml", "xml"}

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

// Validate checks the configuration for errors.
// Returns an error if any configuration value is invalid.
func (c *Config) Validate() error {
	// Validate file types
	for _, t := range c.Types {
		if !slices.Contains(validFileTypes, t) {
			return fmt.Errorf("invalid type %q: valid types are %v", t, validFileTypes)
		}
	}

	// Validate check config
	if c.Check.Concurrency < 0 {
		return fmt.Errorf("check.concurrency must be >= 0, got %d", c.Check.Concurrency)
	}
	if c.Check.Timeout < 0 {
		return fmt.Errorf("check.timeout must be >= 0, got %d", c.Check.Timeout)
	}
	if c.Check.Retries < 0 {
		return fmt.Errorf("check.retries must be >= 0, got %d", c.Check.Retries)
	}

	// Validate output format
	if c.Output.Format != "" && !slices.Contains(validOutputFormats, c.Output.Format) {
		return fmt.Errorf("invalid output.format %q: valid formats are %v", c.Output.Format, validOutputFormats)
	}

	// Validate scan include patterns
	for _, p := range c.Scan.Include {
		if _, err := glob.Compile(p); err != nil {
			return fmt.Errorf("invalid scan.include pattern %q: %w", p, err)
		}
	}

	// Validate scan exclude patterns
	for _, p := range c.Scan.Exclude {
		if _, err := glob.Compile(p); err != nil {
			return fmt.Errorf("invalid scan.exclude pattern %q: %w", p, err)
		}
	}

	// Validate ignore glob patterns
	for _, p := range c.Ignore.Patterns {
		if _, err := glob.Compile(p); err != nil {
			return fmt.Errorf("invalid ignore.patterns pattern %q: %w", p, err)
		}
	}

	// Validate ignore regex patterns
	for _, p := range c.Ignore.Regex {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid ignore.regex pattern %q: %w", p, err)
		}
	}

	return nil
}

// IsEmpty returns true if the config has no settings defined.
// This checks all configuration sections, not just ignore rules.
func (c *Config) IsEmpty() bool {
	return len(c.Types) == 0 &&
		len(c.Scan.Include) == 0 &&
		len(c.Scan.Exclude) == 0 &&
		c.Check.Concurrency == 0 &&
		c.Check.Timeout == 0 &&
		c.Check.Retries == 0 &&
		!c.Check.Strict &&
		c.Output.Format == "" &&
		!c.Output.ShowAlive &&
		c.Output.ShowWarnings == nil &&
		c.Output.ShowDead == nil &&
		!c.Output.ShowStats &&
		len(c.Ignore.Domains) == 0 &&
		len(c.Ignore.Patterns) == 0 &&
		len(c.Ignore.Regex) == 0
}

// HasIgnoreRules returns true if any ignore rules are defined.
func (c *Config) HasIgnoreRules() bool {
	return len(c.Ignore.Domains) > 0 ||
		len(c.Ignore.Patterns) > 0 ||
		len(c.Ignore.Regex) > 0
}

// HasTypes returns true if file types are configured.
func (c *Config) HasTypes() bool {
	return len(c.Types) > 0
}

// HasScanConfig returns true if any scan configuration is set.
func (c *Config) HasScanConfig() bool {
	return len(c.Scan.Include) > 0 || len(c.Scan.Exclude) > 0
}

// HasCheckConfig returns true if any check configuration is set.
func (c *Config) HasCheckConfig() bool {
	return c.Check.Concurrency > 0 ||
		c.Check.Timeout > 0 ||
		c.Check.Retries > 0 ||
		c.Check.Strict
}

// HasOutputConfig returns true if any output configuration is set.
func (c *Config) HasOutputConfig() bool {
	return c.Output.Format != "" ||
		c.Output.ShowAlive ||
		c.Output.ShowWarnings != nil ||
		c.Output.ShowDead != nil ||
		c.Output.ShowStats
}

// GetShowWarnings returns the ShowWarnings value, defaulting to true if not set.
func (c *Config) GetShowWarnings() bool {
	if c.Output.ShowWarnings == nil {
		return true // Default
	}
	return *c.Output.ShowWarnings
}

// GetShowDead returns the ShowDead value, defaulting to true if not set.
func (c *Config) GetShowDead() bool {
	if c.Output.ShowDead == nil {
		return true // Default
	}
	return *c.Output.ShowDead
}

// Merge combines another config into this one (additive for slices).
// This is useful for merging CLI flags with file config.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Merge types (additive)
	c.Types = append(c.Types, other.Types...)

	// Merge scan config (additive)
	c.Scan.Include = append(c.Scan.Include, other.Scan.Include...)
	c.Scan.Exclude = append(c.Scan.Exclude, other.Scan.Exclude...)

	// Merge check config (other overrides if set)
	if other.Check.Concurrency > 0 {
		c.Check.Concurrency = other.Check.Concurrency
	}
	if other.Check.Timeout > 0 {
		c.Check.Timeout = other.Check.Timeout
	}
	if other.Check.Retries > 0 {
		c.Check.Retries = other.Check.Retries
	}
	if other.Check.Strict {
		c.Check.Strict = true
	}

	// Merge output config (other overrides if set)
	if other.Output.Format != "" {
		c.Output.Format = other.Output.Format
	}
	if other.Output.ShowAlive {
		c.Output.ShowAlive = true
	}
	if other.Output.ShowWarnings != nil {
		c.Output.ShowWarnings = other.Output.ShowWarnings
	}
	if other.Output.ShowDead != nil {
		c.Output.ShowDead = other.Output.ShowDead
	}
	if other.Output.ShowStats {
		c.Output.ShowStats = true
	}

	// Merge ignore config (additive)
	c.Ignore.Domains = append(c.Ignore.Domains, other.Ignore.Domains...)
	c.Ignore.Patterns = append(c.Ignore.Patterns, other.Ignore.Patterns...)
	c.Ignore.Regex = append(c.Ignore.Regex, other.Ignore.Regex...)
}
