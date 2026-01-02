// Package parser provides file parsers for extracting URLs from various file formats.
// This file implements the parser registry which manages different file type parsers.
package parser

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FileParser defines the interface for file type parsers.
// Each parser implementation handles a specific file format (markdown, JSON, YAML, etc.).
type FileParser interface {
	// Extensions returns the file extensions this parser handles (e.g., [".json"]).
	// Extensions should include the leading dot.
	Extensions() []string

	// ValidateAndParse validates the content and extracts links in a single pass.
	// Returns an error if the content is malformed.
	ValidateAndParse(filename string, content []byte) ([]Link, error)
}

// Registry manages file parsers by extension.
// It provides thread-safe registration and lookup of parsers.
type Registry struct {
	mu      sync.RWMutex
	parsers map[string]FileParser // extension -> parser
}

// NewRegistry creates a new empty parser registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: map[string]FileParser{},
	}
}

// Register adds a parser to the registry for all its supported extensions.
// If an extension is already registered, it will be overwritten.
func (r *Registry) Register(p FileParser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, ext := range p.Extensions() {
		// Normalize extension to lowercase with leading dot
		ext = normalizeExtension(ext)
		r.parsers[ext] = p
	}
}

// Get returns the parser for the given file extension.
// Returns nil, false if no parser is registered for the extension.
func (r *Registry) Get(ext string) (FileParser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext = normalizeExtension(ext)
	p, ok := r.parsers[ext]
	return p, ok
}

// GetForFile returns the parser for the given filename based on its extension.
// Returns nil, false if no parser is registered for the file's extension.
func (r *Registry) GetForFile(filename string) (FileParser, bool) {
	ext := filepath.Ext(filename)
	return r.Get(ext)
}

// SupportedTypes returns a sorted list of registered file type names.
// For display purposes in CLI help text.
func (r *Registry) SupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use a map to deduplicate (multiple extensions can map to same parser)
	typeNames := map[string]struct{}{}
	for ext := range r.parsers {
		// Convert extension to type name (remove dot)
		typeName := strings.TrimPrefix(ext, ".")
		typeNames[typeName] = struct{}{}
	}

	// Convert to slice
	result := make([]string, 0, len(typeNames))
	for name := range typeNames {
		result = append(result, name)
	}

	sort.Strings(result)
	return result
}

// SupportedExtensions returns all registered file extensions.
func (r *Registry) SupportedExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exts := make([]string, 0, len(r.parsers))
	for ext := range r.parsers {
		exts = append(exts, ext)
	}
	return exts
}

// HasParser returns true if a parser is registered for the given extension.
func (r *Registry) HasParser(ext string) bool {
	_, ok := r.Get(ext)
	return ok
}

// ExtensionsForTypes returns the file extensions for the given type names.
// Type names are without the leading dot (e.g., "md", "json").
// Returns an error if any type name is not supported.
func (r *Registry) ExtensionsForTypes(types []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extensions := make([]string, 0, len(types))
	for _, typeName := range types {
		ext := normalizeExtension(typeName)
		if _, ok := r.parsers[ext]; !ok {
			return nil, fmt.Errorf("unsupported file type: %s", typeName)
		}
		extensions = append(extensions, ext)
	}
	return extensions, nil
}

// normalizeExtension ensures the extension is lowercase and has a leading dot.
func normalizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// defaultRegistry is the global parser registry.
var defaultRegistry = NewRegistry()

// DefaultRegistry returns the global parser registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// RegisterParser registers a parser with the default registry.
func RegisterParser(p FileParser) {
	defaultRegistry.Register(p)
}

// GetParser returns a parser from the default registry for the given extension.
func GetParser(ext string) (FileParser, bool) {
	return defaultRegistry.Get(ext)
}

// GetParserForFile returns a parser from the default registry for the given filename.
func GetParserForFile(filename string) (FileParser, bool) {
	return defaultRegistry.GetForFile(filename)
}

// SupportedFileTypes returns all supported file types from the default registry.
func SupportedFileTypes() []string {
	return defaultRegistry.SupportedTypes()
}
