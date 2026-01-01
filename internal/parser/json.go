package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONParser implements FileParser for JSON files.
type JSONParser struct{}

// NewJSONParser creates a new JSON parser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Extensions returns the file extensions this parser handles.
func (*JSONParser) Extensions() []string {
	return []string{".json"}
}

// Validate checks if the content is valid JSON.
func (*JSONParser) Validate(content []byte) error {
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
func (*JSONParser) Parse(filename string, content []byte) ([]Link, error) {
	if len(content) == 0 {
		return nil, nil
	}

	// Parse JSON
	var v any
	if err := json.Unmarshal(content, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Build line index for position tracking
	lines := buildLineIndex(content)

	// Extract links from the parsed JSON
	extractor := &jsonLinkExtractor{
		filePath: filename,
		content:  content,
		lines:    lines,
		links:    make([]Link, 0, 32),
	}

	extractor.extractFromValue(v, "")

	return extractor.links, nil
}

// jsonLinkExtractor extracts URLs from JSON values.
type jsonLinkExtractor struct {
	filePath string
	content  []byte
	lines    []int
	links    []Link
}

// urlRegex matches HTTP/HTTPS URLs.
var urlRegex = regexp.MustCompile(`https?://[^\s"'\]\}>,]+`)

// extractFromValue recursively extracts URLs from a JSON value.
func (e *jsonLinkExtractor) extractFromValue(v any, path string) {
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
func (e *jsonLinkExtractor) extractFromString(s, path string) {
	if !isHTTPURL(s) && !strings.Contains(s, "http://") && !strings.Contains(s, "https://") {
		return
	}

	// Find all URLs in the string
	matches := urlRegex.FindAllString(s, -1)
	for _, url := range matches {
		// Clean up trailing punctuation
		url = cleanURLTrailing(url)
		if !isHTTPURL(url) {
			continue
		}

		// Find the position of this URL in the original content
		line, col := e.findURLPosition(url)

		e.links = append(e.links, Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     line,
			Column:   col,
			Text:     path,
			Type:     LinkTypeAutolink,
		})
	}
}

// extractFromObject extracts URLs from an object (map).
func (e *jsonLinkExtractor) extractFromObject(obj map[string]any, path string) {
	for key, value := range obj {
		// Check if the key itself is a URL
		if isHTTPURL(key) {
			line, col := e.findURLPosition(key)
			e.links = append(e.links, Link{
				URL:      key,
				FilePath: e.filePath,
				Line:     line,
				Column:   col,
				Text:     path + ".<key>",
				Type:     LinkTypeAutolink,
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
func (e *jsonLinkExtractor) extractFromArray(arr []any, path string) {
	for i, value := range arr {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		e.extractFromValue(value, childPath)
	}
}

// findURLPosition finds the line and column of a URL in the content.
// This is a best-effort approach since JSON doesn't preserve positions after parsing.
func (e *jsonLinkExtractor) findURLPosition(url string) (line, col int) {
	idx := bytes.Index(e.content, []byte(url))
	if idx == -1 {
		return 1, 1
	}

	return e.offsetToLineCol(idx)
}

// offsetToLineCol converts a byte offset to line and column numbers.
func (e *jsonLinkExtractor) offsetToLineCol(offset int) (lineNum, colNum int) {
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

// cleanURLTrailing removes trailing punctuation from URLs.
func cleanURLTrailing(url string) string {
	// Remove trailing punctuation that's likely not part of the URL
	for url != "" {
		last := url[len(url)-1]
		if last == '.' || last == ',' || last == ';' || last == ':' ||
			last == ')' || last == ']' || last == '}' || last == '"' || last == '\'' {
			url = url[:len(url)-1]
		} else {
			break
		}
	}
	return url
}

// init registers the JSON parser with the default registry.
func init() {
	RegisterParser(NewJSONParser())
}
