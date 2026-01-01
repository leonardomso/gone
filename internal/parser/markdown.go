package parser

// MarkdownParser implements FileParser for markdown files.
type MarkdownParser struct{}

// NewMarkdownParser creates a new markdown parser.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Extensions returns the file extensions this parser handles.
func (*MarkdownParser) Extensions() []string {
	return []string{".md", ".mdx", ".markdown"}
}

// Validate checks if the content is valid markdown.
// Markdown is very permissive, so we just return nil.
// Any text content is valid markdown.
func (*MarkdownParser) Validate(_ []byte) error {
	return nil
}

// Parse extracts links from markdown content.
func (*MarkdownParser) Parse(filename string, content []byte) ([]Link, error) {
	return ExtractLinksFromContent(content, filename)
}

// init registers the markdown parser with the default registry.
func init() {
	RegisterParser(NewMarkdownParser())
}
