// Package filter provides URL filtering based on domains and patterns.
package filter

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// IgnoreReason describes why a URL was ignored.
type IgnoreReason struct {
	Type string // "domain", "pattern", or "regex"
	Rule string // The rule that matched
	URL  string // The URL that was ignored
	File string // Source file
	Line int    // Line number
}

// Filter determines which URLs should be skipped during link checking.
type Filter struct {
	// domains maps domain names for O(1) lookup.
	// Each domain also matches its subdomains.
	domains map[string]bool

	// globPatterns are compiled glob patterns for URL matching.
	globPatterns []compiledGlob

	// regexPatterns are compiled regex patterns for URL matching.
	regexPatterns []compiledRegex

	// Track ignored URLs for reporting
	ignored []IgnoreReason
}

// compiledGlob holds a glob pattern and its original string for error reporting.
type compiledGlob struct {
	pattern  glob.Glob
	original string
}

// compiledRegex holds a regex pattern and its original string for error reporting.
type compiledRegex struct {
	pattern  *regexp.Regexp
	original string
}

// Config holds filter configuration.
type Config struct {
	Domains       []string // Domains to ignore (includes subdomains)
	GlobPatterns  []string // Glob patterns (e.g., "*.local/*")
	RegexPatterns []string // Regex patterns (e.g., ".*\\.internal\\..*")
}

// New creates a new Filter from the given configuration.
// Patterns are compiled once for performance.
// Returns an error if any pattern fails to compile.
func New(cfg Config) (*Filter, error) {
	f := &Filter{
		domains: map[string]bool{},
		ignored: []IgnoreReason{},
	}

	// Add domains to map (normalize to lowercase)
	for _, d := range cfg.Domains {
		// Normalize: lowercase, trim whitespace
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			f.domains[d] = true
		}
	}

	// Compile glob patterns
	for _, p := range cfg.GlobPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", p, err)
		}
		f.globPatterns = append(f.globPatterns, compiledGlob{
			pattern:  g,
			original: p,
		})
	}

	// Compile regex patterns
	for _, p := range cfg.RegexPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", p, err)
		}
		f.regexPatterns = append(f.regexPatterns, compiledRegex{
			pattern:  r,
			original: p,
		})
	}

	return f, nil
}

// ShouldIgnore checks if a URL should be skipped.
// If the URL matches any rule, it records the reason and returns true.
// Check order (fastest first): domain → glob → regex.
func (f *Filter) ShouldIgnore(rawURL, file string, line int) bool {
	if f == nil {
		return false
	}

	// Check domain first (O(1) lookup)
	if reason, ok := f.matchesDomain(rawURL); ok {
		f.ignored = append(f.ignored, IgnoreReason{
			Type: "domain",
			Rule: reason,
			URL:  rawURL,
			File: file,
			Line: line,
		})
		return true
	}

	// Check glob patterns
	if reason, ok := f.matchesGlob(rawURL); ok {
		f.ignored = append(f.ignored, IgnoreReason{
			Type: "pattern",
			Rule: reason,
			URL:  rawURL,
			File: file,
			Line: line,
		})
		return true
	}

	// Check regex patterns
	if reason, ok := f.matchesRegex(rawURL); ok {
		f.ignored = append(f.ignored, IgnoreReason{
			Type: "regex",
			Rule: reason,
			URL:  rawURL,
			File: file,
			Line: line,
		})
		return true
	}

	return false
}

// matchesDomain checks if the URL's domain matches any ignored domain.
// Also checks if the URL's domain is a subdomain of an ignored domain.
func (f *Filter) matchesDomain(rawURL string) (string, bool) {
	if len(f.domains) == 0 {
		return "", false
	}

	// Parse URL to extract host
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", false
	}

	// Direct match
	if f.domains[host] {
		return host, true
	}

	// Check if host is a subdomain of any ignored domain.
	// For example, if "example.com" is ignored, "www.example.com" should also match.
	// We do this by checking if the host ends with ".example.com".
	for domain := range f.domains {
		if strings.HasSuffix(host, "."+domain) {
			return domain, true
		}
	}

	return "", false
}

// matchesGlob checks if the URL matches any glob pattern.
func (f *Filter) matchesGlob(rawURL string) (string, bool) {
	for _, g := range f.globPatterns {
		if g.pattern.Match(rawURL) {
			return g.original, true
		}
	}
	return "", false
}

// matchesRegex checks if the URL matches any regex pattern.
func (f *Filter) matchesRegex(rawURL string) (string, bool) {
	for _, r := range f.regexPatterns {
		if r.pattern.MatchString(rawURL) {
			return r.original, true
		}
	}
	return "", false
}

// IgnoredCount returns the number of URLs that were ignored.
func (f *Filter) IgnoredCount() int {
	if f == nil {
		return 0
	}
	return len(f.ignored)
}

// IgnoredURLs returns all ignored URLs with their reasons.
func (f *Filter) IgnoredURLs() []IgnoreReason {
	if f == nil {
		return nil
	}
	return f.ignored
}

// Reset clears the list of ignored URLs.
// Useful if reusing a filter for multiple checks.
func (f *Filter) Reset() {
	if f != nil {
		f.ignored = f.ignored[:0]
	}
}

// HasRules returns true if the filter has any rules defined.
func (f *Filter) HasRules() bool {
	if f == nil {
		return false
	}
	return len(f.domains) > 0 || len(f.globPatterns) > 0 || len(f.regexPatterns) > 0
}

// Stats returns a summary of the filter's rules.
func (f *Filter) Stats() (domains, globs, regexes int) {
	if f == nil {
		return 0, 0, 0
	}
	return len(f.domains), len(f.globPatterns), len(f.regexPatterns)
}
