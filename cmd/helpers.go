package cmd

import (
	"fmt"
	"slices"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/config"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"

	// Import parser subpackages to trigger their init() registration.
	_ "github.com/leonardomso/gone/internal/parser/json"
	_ "github.com/leonardomso/gone/internal/parser/markdown"
	_ "github.com/leonardomso/gone/internal/parser/toml"
	_ "github.com/leonardomso/gone/internal/parser/xml"
	_ "github.com/leonardomso/gone/internal/parser/yaml"
)

// LoadedConfig wraps a loaded configuration and provides helper methods
// for getting effective values that respect CLI overrides.
type LoadedConfig struct {
	cfg      *config.Config
	noConfig bool
}

// LoadConfig loads the configuration file unless noConfig is true.
// Returns an error if the config file exists but is invalid.
func LoadConfig(noConfig bool) (*LoadedConfig, error) {
	if noConfig {
		return &LoadedConfig{cfg: &config.Config{}, noConfig: true}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &LoadedConfig{cfg: cfg, noConfig: false}, nil
}

// Config returns the underlying config for direct access.
func (lc *LoadedConfig) Config() *config.Config {
	return lc.cfg
}

// GetTypes returns the effective file types.
// CLI types override config if they differ from the CLI default.
func (lc *LoadedConfig) GetTypes(cliTypes, cliDefault []string) []string {
	// If CLI types differ from default, CLI was explicitly set
	if !slices.Equal(cliTypes, cliDefault) {
		return cliTypes
	}
	// Use config types if set
	if lc.cfg.HasTypes() {
		return lc.cfg.Types
	}
	// Fall back to CLI default
	return cliDefault
}

// GetScanOptions returns scanner include/exclude patterns from config.
func (lc *LoadedConfig) GetScanOptions() (include, exclude []string) {
	return lc.cfg.Scan.Include, lc.cfg.Scan.Exclude
}

// GetConcurrency returns the effective concurrency.
// CLI overrides config if it differs from the default.
func (lc *LoadedConfig) GetConcurrency(cliValue, defaultValue int) int {
	if cliValue != defaultValue {
		return cliValue // CLI explicitly set
	}
	if lc.cfg.Check.Concurrency > 0 {
		return lc.cfg.Check.Concurrency
	}
	return defaultValue
}

// GetTimeout returns the effective timeout in seconds.
// CLI overrides config if it differs from the default.
func (lc *LoadedConfig) GetTimeout(cliValue, defaultValue int) int {
	if cliValue != defaultValue {
		return cliValue // CLI explicitly set
	}
	if lc.cfg.Check.Timeout > 0 {
		return lc.cfg.Check.Timeout
	}
	return defaultValue
}

// GetRetries returns the effective retry count.
// CLI overrides config if it differs from the default.
func (lc *LoadedConfig) GetRetries(cliValue, defaultValue int) int {
	if cliValue != defaultValue {
		return cliValue // CLI explicitly set
	}
	if lc.cfg.Check.Retries > 0 {
		return lc.cfg.Check.Retries
	}
	return defaultValue
}

// GetStrict returns the effective strict mode setting.
// CLI true overrides config.
func (lc *LoadedConfig) GetStrict(cliValue bool) bool {
	if cliValue {
		return true // CLI explicitly set
	}
	return lc.cfg.Check.Strict
}

// GetOutputFormat returns the effective output format.
// CLI overrides config if set.
func (lc *LoadedConfig) GetOutputFormat(cliValue string) string {
	if cliValue != "" {
		return cliValue // CLI explicitly set
	}
	return lc.cfg.Output.Format
}

// GetShowAlive returns the effective showAlive setting.
// CLI true overrides config.
func (lc *LoadedConfig) GetShowAlive(cliValue bool) bool {
	if cliValue {
		return true
	}
	return lc.cfg.Output.ShowAlive
}

// GetShowWarnings returns the effective showWarnings setting.
// CLI true overrides config. Defaults to true if not set.
func (lc *LoadedConfig) GetShowWarnings(cliValue bool) bool {
	if cliValue {
		return true
	}
	return lc.cfg.GetShowWarnings()
}

// GetShowDead returns the effective showDead setting.
// CLI true overrides config. Defaults to true if not set.
func (lc *LoadedConfig) GetShowDead(cliValue bool) bool {
	if cliValue {
		return true
	}
	return lc.cfg.GetShowDead()
}

// GetShowStats returns the effective showStats setting.
// CLI true overrides config.
func (lc *LoadedConfig) GetShowStats(cliValue bool) bool {
	if cliValue {
		return true
	}
	return lc.cfg.Output.ShowStats
}

// BuildCheckerOptions creates checker.Options from config and CLI values.
func (lc *LoadedConfig) BuildCheckerOptions(cliConcurrency, cliTimeout, cliRetries int) checker.Options {
	defaultOpts := checker.DefaultOptions()

	return defaultOpts.
		WithConcurrency(lc.GetConcurrency(cliConcurrency, checker.DefaultConcurrency)).
		WithTimeout(time.Duration(lc.GetTimeout(cliTimeout, int(checker.DefaultTimeout.Seconds()))) * time.Second).
		WithMaxRetries(lc.GetRetries(cliRetries, checker.DefaultMaxRetries))
}

// BuildScanOptions creates scanner.ScanOptions from config and path.
func (lc *LoadedConfig) BuildScanOptions(path string, cliTypes, cliDefaultTypes []string) scanner.ScanOptions {
	include, exclude := lc.GetScanOptions()
	return scanner.ScanOptions{
		Root:    path,
		Types:   lc.GetTypes(cliTypes, cliDefaultTypes),
		Include: include,
		Exclude: exclude,
	}
}

// FilterOptions holds the configuration for creating a URL filter.
// This struct consolidates the filter-related flags used across commands.
type FilterOptions struct {
	Domains  []string // Domains to ignore (includes subdomains)
	Patterns []string // Glob patterns to ignore
	Regex    []string // Regex patterns to ignore
	NoConfig bool     // Skip loading .gonerc.yaml config file
}

// CreateFilter builds a URL filter from config file and CLI flags.
// If noConfig is true, the .gonerc.yaml file will not be loaded.
// CLI flags are merged additively with config file settings.
// Returns nil if no filter rules are defined.
func CreateFilter(opts FilterOptions) (*filter.Filter, error) {
	var cfg *config.Config

	// Load config file unless --no-config is set
	if !opts.NoConfig {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	} else {
		cfg = &config.Config{}
	}

	// Merge CLI flags (additive)
	cfg.Ignore.Domains = append(cfg.Ignore.Domains, opts.Domains...)
	cfg.Ignore.Patterns = append(cfg.Ignore.Patterns, opts.Patterns...)
	cfg.Ignore.Regex = append(cfg.Ignore.Regex, opts.Regex...)

	// If no ignore rules, return nil (no filtering)
	if !cfg.HasIgnoreRules() {
		return nil, nil
	}

	// Create filter
	return filter.New(filter.Config{
		Domains:       cfg.Ignore.Domains,
		GlobPatterns:  cfg.Ignore.Patterns,
		RegexPatterns: cfg.Ignore.Regex,
	})
}

// CreateFilterWithConfig builds a URL filter using a pre-loaded config.
// CLI flags are merged additively with the config settings.
// Returns nil if no filter rules are defined.
func CreateFilterWithConfig(cfg *config.Config, cliDomains, cliPatterns, cliRegex []string) (*filter.Filter, error) {
	// Merge CLI flags (additive)
	domains := append([]string{}, cfg.Ignore.Domains...)
	domains = append(domains, cliDomains...)

	patterns := append([]string{}, cfg.Ignore.Patterns...)
	patterns = append(patterns, cliPatterns...)

	regex := append([]string{}, cfg.Ignore.Regex...)
	regex = append(regex, cliRegex...)

	// If no ignore rules, return nil (no filtering)
	if len(domains) == 0 && len(patterns) == 0 && len(regex) == 0 {
		return nil, nil
	}

	// Create filter
	return filter.New(filter.Config{
		Domains:       domains,
		GlobPatterns:  patterns,
		RegexPatterns: regex,
	})
}

// CountUniqueURLs returns the number of unique URLs in a slice of checker.Link.
// This is useful for displaying progress information and deduplication stats.
func CountUniqueURLs(links []checker.Link) int {
	seen := make(map[string]bool, len(links))
	for _, l := range links {
		seen[l.URL] = true
	}
	return len(seen)
}

// ConvertParserLinks converts a slice of parser.Link to checker.Link.
// This bridges the gap between the parser and checker packages.
func ConvertParserLinks(parserLinks []parser.Link) []checker.Link {
	links := make([]checker.Link, len(parserLinks))
	for i, pl := range parserLinks {
		links[i] = checker.Link{
			URL:      pl.URL,
			FilePath: pl.FilePath,
			Line:     pl.Line,
			Text:     pl.Text,
		}
	}
	return links
}

// FilterParserLinks applies a URL filter to parser links and returns checker links.
// Links that match the filter are excluded from the result.
// Returns all links converted to checker.Link if urlFilter is nil.
func FilterParserLinks(parserLinks []parser.Link, urlFilter *filter.Filter) []checker.Link {
	links := make([]checker.Link, 0, len(parserLinks))
	for _, pl := range parserLinks {
		// Check if URL should be ignored
		if urlFilter != nil && urlFilter.ShouldIgnore(pl.URL, pl.FilePath, pl.Line) {
			continue
		}
		links = append(links, checker.Link{
			URL:      pl.URL,
			FilePath: pl.FilePath,
			Line:     pl.Line,
			Text:     pl.Text,
		})
	}
	return links
}

// FilterResultsByStatus filters checker results based on status predicates.
// This provides a generic way to filter results for different display modes.

// FilterResultsWarnings returns results with warning status (redirect or blocked).
// Pre-allocates slice capacity based on expected ratio (~10-30% warnings).
func FilterResultsWarnings(results []checker.Result) []checker.Result {
	filtered := make([]checker.Result, 0, len(results)/4)
	for _, r := range results {
		if r.IsWarning() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterResultsDead returns results that are dead or errored.
// Pre-allocates slice capacity based on expected ratio (~5-15% dead).
func FilterResultsDead(results []checker.Result) []checker.Result {
	filtered := make([]checker.Result, 0, len(results)/8)
	for _, r := range results {
		if r.IsDead() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterResultsDuplicates returns only duplicate results.
// Pre-allocates slice capacity based on expected ratio (~10% duplicates).
func FilterResultsDuplicates(results []checker.Result) []checker.Result {
	filtered := make([]checker.Result, 0, len(results)/10)
	for _, r := range results {
		if r.IsDuplicate() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterResultsAlive returns only alive results.
// Pre-allocates slice capacity - alive is typically the majority (~70-80%).
func FilterResultsAlive(results []checker.Result) []checker.Result {
	filtered := make([]checker.Result, 0, len(results)*3/4)
	for _, r := range results {
		if r.IsAlive() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
