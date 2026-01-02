// Package toml implements a URL extractor for TOML files.
package toml

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/leonardomso/gone/internal/parser"
)

// Parser implements parser.FileParser for TOML files.
type Parser struct{}

// New creates a new TOML parser.
func New() *Parser {
	return &Parser{}
}

// Extensions returns the file extensions this parser handles.
func (*Parser) Extensions() []string {
	return []string{".toml"}
}

// Validate checks if the content is valid TOML.
func (*Parser) Validate(content []byte) error {
	if len(content) == 0 {
		return nil // Empty file is valid (no links to extract)
	}

	var v any
	if _, err := toml.Decode(string(content), &v); err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}
	return nil
}

// Parse extracts links from TOML content.
// It extracts URLs from both string values and table/key names.
// Deprecated: Use ValidateAndParse for better performance.
func (p *Parser) Parse(filename string, content []byte) ([]parser.Link, error) {
	return p.ValidateAndParse(filename, content)
}

// ValidateAndParse validates the content and extracts links in a single pass.
// This is more efficient than calling Validate and Parse separately.
func (*Parser) ValidateAndParse(filename string, content []byte) ([]parser.Link, error) {
	if len(content) == 0 {
		return nil, nil
	}

	// Parse TOML (single pass - validates and parses)
	var v map[string]any
	if _, err := toml.Decode(string(content), &v); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}

	// Build line index for position tracking
	lines := parser.BuildLineIndex(content)

	// Extract links from the parsed TOML
	extractor := &linkExtractor{
		filePath: filename,
		content:  content,
		lines:    lines,
		links:    make([]parser.Link, 0, 32),
	}

	extractor.extractFromValue(v, "")

	return extractor.links, nil
}

// linkExtractor extracts URLs from TOML values.
type linkExtractor struct {
	filePath string
	content  []byte
	lines    []int
	links    []parser.Link
}

// extractFromValue recursively extracts URLs from a TOML value.
func (e *linkExtractor) extractFromValue(v any, path string) {
	switch val := v.(type) {
	case string:
		e.extractFromString(val, path)
	case map[string]any:
		e.extractFromTable(val, path)
	case []any:
		e.extractFromArray(val, path)
	case []map[string]any:
		// Array of tables
		for i, item := range val {
			// Use string concatenation with strconv.Itoa instead of fmt.Sprintf for performance
			childPath := path + "[" + strconv.Itoa(i) + "]"
			e.extractFromTable(item, childPath)
		}
	}
}

// extractFromString extracts URLs from a string value.
func (e *linkExtractor) extractFromString(s, path string) {
	// Quick check: skip if no "http" substring (covers both http:// and https://)
	if !strings.Contains(s, "http") {
		return
	}

	// Find all URLs in the string
	matches := parser.URLRegex.FindAllString(s, -1)
	for _, url := range matches {
		// Clean up trailing punctuation
		url = parser.CleanURLTrailing(url)
		if !parser.IsHTTPURL(url) {
			continue
		}

		// Find the position of this URL in the original content
		line, col := e.findURLPosition(url)

		e.links = append(e.links, parser.Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     line,
			Column:   col,
			Text:     path,
			Type:     parser.LinkTypeAutolink,
		})
	}
}

// extractFromTable extracts URLs from a TOML table (map).
func (e *linkExtractor) extractFromTable(table map[string]any, path string) {
	for key, value := range table {
		// Check if the key itself is a URL
		if parser.IsHTTPURL(key) {
			line, col := e.findURLPosition(key)
			e.links = append(e.links, parser.Link{
				URL:      key,
				FilePath: e.filePath,
				Line:     line,
				Column:   col,
				Text:     path + ".<key>",
				Type:     parser.LinkTypeAutolink,
			})
		}

		// Build path for this value
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}

		// Recurse into value
		e.extractFromValue(value, childPath)
	}
}

// extractFromArray extracts URLs from an array.
func (e *linkExtractor) extractFromArray(arr []any, path string) {
	for i, value := range arr {
		// Use string concatenation with strconv.Itoa instead of fmt.Sprintf for performance
		childPath := path + "[" + strconv.Itoa(i) + "]"
		e.extractFromValue(value, childPath)
	}
}

// findURLPosition finds the line and column of a URL in the content.
// This is a best-effort approach since TOML doesn't preserve positions after parsing.
func (e *linkExtractor) findURLPosition(url string) (line, col int) {
	idx := bytes.Index(e.content, []byte(url))
	if idx == -1 {
		return 1, 1
	}

	return parser.OffsetToLineCol(e.lines, idx)
}

// init registers the TOML parser with the default registry.
func init() {
	parser.RegisterParser(New())
}
