// Package xml implements a URL extractor for XML files.
package xml //nolint:revive // package name matches file type being parsed

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/leonardomso/gone/internal/parser"
)

// urlAttributes lists common XML/HTML attributes that typically contain URLs.
var urlAttributes = map[string]bool{
	"href":       true,
	"src":        true,
	"url":        true,
	"link":       true,
	"action":     true,
	"data":       true,
	"poster":     true,
	"srcset":     true,
	"formaction": true,
	"cite":       true,
	"background": true,
	"xlink:href": true,
}

// Parser implements parser.FileParser for XML files.
type Parser struct{}

// New creates a new XML parser.
func New() *Parser {
	return &Parser{}
}

// Extensions returns the file extensions this parser handles.
func (*Parser) Extensions() []string {
	return []string{".xml"}
}

// Validate checks if the content is valid XML.
func (*Parser) Validate(content []byte) error {
	if len(content) == 0 {
		return nil // Empty file is valid (no links to extract)
	}

	decoder := xml.NewDecoder(bytes.NewReader(content))
	for {
		_, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid XML: %w", err)
		}
	}
	return nil
}

// Parse extracts links from XML content.
// It extracts URLs from known URL attributes and text content.
//
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

	// Build line index for position tracking
	lines := parser.BuildLineIndex(content)

	// Extract links (single pass - validates and parses)
	extractor := &linkExtractor{
		filePath: filename,
		content:  content,
		lines:    lines,
		links:    make([]parser.Link, 0, 32),
		seen:     map[string]bool{},
	}

	decoder := xml.NewDecoder(bytes.NewReader(content))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid XML: %w", err)
		}

		extractor.processToken(token, decoder.InputOffset())
	}

	return extractor.links, nil
}

// linkExtractor extracts URLs from XML tokens.
type linkExtractor struct {
	filePath string
	content  []byte
	lines    []int
	links    []parser.Link
	seen     map[string]bool // Track seen URLs to avoid duplicates from same position
}

// processToken processes a single XML token.
func (e *linkExtractor) processToken(token xml.Token, offset int64) {
	switch t := token.(type) {
	case xml.StartElement:
		e.extractFromElement(t, offset)
	case xml.CharData:
		e.extractFromText(string(t), offset)
	}
}

// extractFromElement extracts URLs from element attributes.
func (e *linkExtractor) extractFromElement(elem xml.StartElement, offset int64) {
	for _, attr := range elem.Attr {
		// Check if this is a URL attribute
		attrName := strings.ToLower(attr.Name.Local)
		if attr.Name.Space != "" {
			attrName = attr.Name.Space + ":" + attrName
		}

		if urlAttributes[attrName] {
			url := strings.TrimSpace(attr.Value)
			if parser.IsHTTPURL(url) {
				line, col := e.findURLPosition(url)
				e.addLink(url, line, col, elem.Name.Local+"."+attrName)
			}
		}

		// Also check for URLs embedded in attribute values
		e.extractEmbeddedURLs(attr.Value, offset, elem.Name.Local+"."+attrName)
	}
}

// extractFromText extracts URLs from text content.
func (e *linkExtractor) extractFromText(text string, offset int64) {
	e.extractEmbeddedURLs(text, offset, "text")
}

// extractEmbeddedURLs finds URLs embedded in a string value.
func (e *linkExtractor) extractEmbeddedURLs(s string, _ int64, context string) {
	// Quick check: skip if no "http" substring (covers both http:// and https://)
	if !strings.Contains(s, "http") {
		return
	}

	matches := parser.URLRegex.FindAllString(s, -1)
	for _, url := range matches {
		url = parser.CleanURLTrailing(url)
		if !parser.IsHTTPURL(url) {
			continue
		}

		line, col := e.findURLPosition(url)
		e.addLink(url, line, col, context)
	}
}

// addLink adds a link if not already seen at this position.
func (e *linkExtractor) addLink(url string, line, col int, context string) {
	// Create a unique key for this URL at this position
	key := fmt.Sprintf("%s:%d:%d", url, line, col)
	if e.seen[key] {
		return
	}
	e.seen[key] = true

	e.links = append(e.links, parser.Link{
		URL:      url,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     context,
		Type:     parser.LinkTypeAutolink,
	})
}

// findURLPosition finds the line and column of a URL in the content.
func (e *linkExtractor) findURLPosition(url string) (line, col int) {
	idx := bytes.Index(e.content, []byte(url))
	if idx == -1 {
		return 1, 1
	}

	return parser.OffsetToLineCol(e.lines, idx)
}

// init registers the XML parser with the default registry.
func init() {
	parser.RegisterParser(New())
}
