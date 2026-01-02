// Package markdown implements a URL extractor for Markdown files.
package markdown

import (
	"github.com/leonardomso/gone/internal/parser"
)

// Parser implements parser.FileParser for markdown files.
type Parser struct{}

// New creates a new markdown parser.
func New() *Parser {
	return &Parser{}
}

// Extensions returns the file extensions this parser handles.
func (*Parser) Extensions() []string {
	return []string{".md", ".mdx", ".markdown"}
}

// Validate checks if the content is valid markdown.
// Markdown is very permissive, so we just return nil.
// Any text content is valid markdown.
func (*Parser) Validate(_ []byte) error {
	return nil
}

// Parse extracts links from markdown content.
func (*Parser) Parse(filename string, content []byte) ([]parser.Link, error) {
	return parser.ExtractLinksFromContent(content, filename)
}

// init registers the markdown parser with the default registry.
func init() {
	parser.RegisterParser(New())
}
