// Package json implements a URL extractor for JSON files.
package jsonparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/parser"
)

// Parser implements parser.FileParser for JSON files.
type Parser struct{}

// New creates a new JSON parser.
func New() *Parser {
	return &Parser{}
}

// Extensions returns the file extensions this parser handles.
func (*Parser) Extensions() []string {
	return []string{".json"}
}

// Validate checks if the content is valid JSON.
func (*Parser) Validate(content []byte) error {
	if len(content) == 0 {
		return nil // Empty file is valid (no links to extract)
	}

	var v any
	if err := json.Unmarshal(content, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// Parse extracts links from JSON content.
// It extracts URLs from both string values and object keys.
func (*Parser) Parse(filename string, content []byte) ([]parser.Link, error) {
	if len(content) == 0 {
		return nil, nil
	}

	// Parse JSON
	var v any
	if err := json.Unmarshal(content, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Build line index for position tracking
	lines := parser.BuildLineIndex(content)

	// Extract links from the parsed JSON
	extractor := &linkExtractor{
		filePath: filename,
		content:  content,
		lines:    lines,
		links:    make([]parser.Link, 0, 32),
	}

	extractor.extractFromValue(v, "")

	return extractor.links, nil
}

// linkExtractor extracts URLs from JSON values.
type linkExtractor struct {
	filePath string
	content  []byte
	lines    []int
	links    []parser.Link
}

// extractFromValue recursively extracts URLs from a JSON value.
func (e *linkExtractor) extractFromValue(v any, path string) {
	switch val := v.(type) {
	case string:
		e.extractFromString(val, path)
	case map[string]any:
		e.extractFromObject(val, path)
	case []any:
		e.extractFromArray(val, path)
	}
}

// extractFromString extracts URLs from a string value.
func (e *linkExtractor) extractFromString(s, path string) {
	if !parser.IsHTTPURL(s) && !strings.Contains(s, "http://") && !strings.Contains(s, "https://") {
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

// extractFromObject extracts URLs from an object (map).
func (e *linkExtractor) extractFromObject(obj map[string]any, path string) {
	for key, value := range obj {
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
		childPath := fmt.Sprintf("%s[%d]", path, i)
		e.extractFromValue(value, childPath)
	}
}

// findURLPosition finds the line and column of a URL in the content.
// This is a best-effort approach since JSON doesn't preserve positions after parsing.
func (e *linkExtractor) findURLPosition(url string) (line, col int) {
	idx := bytes.Index(e.content, []byte(url))
	if idx == -1 {
		return 1, 1
	}

	return e.offsetToLineCol(idx)
}

// offsetToLineCol converts a byte offset to line and column numbers.
func (e *linkExtractor) offsetToLineCol(offset int) (lineNum, colNum int) {
	lineNum = 1
	colNum = 1

	for i, lineStart := range e.lines {
		if offset < lineStart {
			if i > 0 {
				lineNum = i
				colNum = offset - e.lines[i-1] + 1
			}
			return lineNum, colNum
		}
		lineNum = i + 1
		colNum = offset - lineStart + 1
	}

	return lineNum, colNum
}

// init registers the JSON parser with the default registry.
func init() {
	parser.RegisterParser(New())
}
