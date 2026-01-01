package cmd

import (
	"fmt"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/config"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/parser"
)

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
	if cfg.IsEmpty() {
		return nil, nil
	}

	// Create filter
	return filter.New(filter.Config{
		Domains:       cfg.Ignore.Domains,
		GlobPatterns:  cfg.Ignore.Patterns,
		RegexPatterns: cfg.Ignore.Regex,
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
