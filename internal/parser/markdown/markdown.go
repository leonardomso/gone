// Package markdown implements a URL extractor for Markdown files.
// It supports various markdown link formats including inline links, reference links,
// images, autolinks, and HTML anchor tags. URLs inside code blocks are ignored.
package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/leonardomso/gone/internal/parser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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
	return ExtractLinksFromContent(content, filename)
}

// init registers the markdown parser with the default registry.
func init() {
	parser.RegisterParser(New())
}

// linkExtractor walks the AST and extracts links.
type linkExtractor struct {
	// Track reference definitions: name -> (url, line)
	refDefs  map[string]refDef
	filePath string
	links    []parser.Link
	source   []byte
	lines    []int // byte offset for start of each line

	// Track if we're inside a code block
	inCodeBlock bool
}

// refDef holds reference definition info.
type refDef struct {
	url  string
	line int
}

// htmlLinkRegex matches <a href="..."> tags.
var htmlLinkRegex = regexp.MustCompile(`<a\s+[^>]*href=["']([^"']+)["'][^>]*>([^<]*)</a>`)

// refDefRegex matches reference-style link definitions: [name]: url
// Compiled at package level to avoid recompilation on each call.
var refDefRegex = regexp.MustCompile(`^\s*\[([^\]]+)\]:\s*(\S+)`)

// ExtractLinksFromContent extracts links from markdown content.
func ExtractLinksFromContent(content []byte, filePath string) ([]parser.Link, error) {
	// Create goldmark parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify, // Auto-link bare URLs
		),
		goldmark.WithParserOptions(
			gmparser.WithAutoHeadingID(),
		),
	)

	// Parse the markdown into AST
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader)

	// Build line offset index for position calculation
	lines := parser.BuildLineIndex(content)

	// Extract reference definitions first
	refDefs := extractRefDefs(content)

	// Create extractor and walk the AST
	// Pre-allocate links slice - typical markdown files have ~10-30 links
	extractor := &linkExtractor{
		links:    make([]parser.Link, 0, 32),
		source:   content,
		filePath: filePath,
		lines:    lines,
		refDefs:  refDefs,
	}

	// Walk the AST
	_ = ast.Walk(doc, extractor.walk)

	// Also extract HTML links (goldmark doesn't parse these as links)
	extractor.extractHTMLLinks(content)

	return extractor.links, nil
}

// walk is the AST walker function.
func (e *linkExtractor) walk(n ast.Node, entering bool) (ast.WalkStatus, error) {
	// Track code block state
	if n.Kind() == ast.KindCodeBlock || n.Kind() == ast.KindFencedCodeBlock {
		e.inCodeBlock = entering
		return ast.WalkContinue, nil
	}

	// Skip if inside code block
	if e.inCodeBlock {
		return ast.WalkContinue, nil
	}

	// Only process on enter
	if !entering {
		return ast.WalkContinue, nil
	}

	switch node := n.(type) {
	case *ast.Link:
		e.handleLink(node)
	case *ast.Image:
		e.handleImage(node)
	case *ast.AutoLink:
		e.handleAutoLink(node)
	}

	return ast.WalkContinue, nil
}

// handleLink processes a standard markdown link.
func (e *linkExtractor) handleLink(node *ast.Link) {
	linkURL := string(node.Destination)

	// Skip non-HTTP links (anchors, mailto, tel, etc.)
	if !parser.IsHTTPURL(linkURL) {
		return
	}

	// Get link text from children
	linkText := e.getNodeText(node)

	// Calculate position
	line, col := e.getPosition(node)

	link := parser.Link{
		URL:      linkURL,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     linkText,
		Type:     parser.LinkTypeInline,
	}

	// Check reference definitions for this URL
	for refName, refDef := range e.refDefs {
		if refDef.url == linkURL && refDef.line != line {
			link.Type = parser.LinkTypeReference
			link.RefName = refName
			link.RefDefLine = refDef.line
			break
		}
	}

	e.links = append(e.links, link)
}

// handleImage processes an image link.
func (e *linkExtractor) handleImage(node *ast.Image) {
	imageURL := string(node.Destination)

	// Skip non-HTTP URLs (data URLs, relative paths, etc.)
	if !parser.IsHTTPURL(imageURL) {
		return
	}

	// Get alt text from node children
	altText := e.getNodeText(node)

	line, col := e.getPosition(node)

	e.links = append(e.links, parser.Link{
		URL:      imageURL,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     altText,
		Type:     parser.LinkTypeImage,
	})
}

// handleAutoLink processes a bare URL that's auto-linked.
func (e *linkExtractor) handleAutoLink(node *ast.AutoLink) {
	url := string(node.URL(e.source))

	// Skip non-HTTP URLs
	if !parser.IsHTTPURL(url) {
		return
	}

	line, col := e.getPosition(node)

	e.links = append(e.links, parser.Link{
		URL:      url,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     "", // Auto-links don't have separate text
		Type:     parser.LinkTypeAutolink,
	})
}

// extractHTMLLinks finds <a href="..."> tags in the content.
func (e *linkExtractor) extractHTMLLinks(content []byte) {
	matches := htmlLinkRegex.FindAllSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		url := string(content[match[2]:match[3]])
		linkText := string(content[match[4]:match[5]])

		// Skip non-HTTP URLs
		if !parser.IsHTTPURL(url) {
			continue
		}

		// Calculate line and column from byte offset
		line, col := e.offsetToLineCol(match[0])

		e.links = append(e.links, parser.Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     line,
			Column:   col,
			Text:     linkText,
			Type:     parser.LinkTypeHTML,
		})
	}
}

// getNodeText extracts text content from a node's children.
func (e *linkExtractor) getNodeText(n ast.Node) string {
	var buf bytes.Buffer

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			buf.Write(textNode.Segment.Value(e.source))
		} else if child.HasChildren() {
			// Recursively get text from nested nodes
			buf.WriteString(e.getNodeText(child))
		}
	}

	return buf.String()
}

// getPosition returns the line and column for a node.
func (e *linkExtractor) getPosition(n ast.Node) (line, col int) {
	// For inline nodes (Link, Image, AutoLink), get position from child text
	if n.Type() == ast.TypeInline {
		// Check if it's a Text node directly
		if textNode, ok := n.(*ast.Text); ok {
			return e.offsetToLineCol(textNode.Segment.Start)
		}

		// Look for text children
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				return e.offsetToLineCol(textNode.Segment.Start)
			}
		}

		// For nodes without text children (like empty links), default to 1,1
		return 1, 1
	}

	// For block nodes, try to get position from the node's lines
	if lines := n.Lines(); lines != nil && lines.Len() > 0 {
		seg := lines.At(0)
		return e.offsetToLineCol(seg.Start)
	}

	return 1, 1 // Default if we can't determine position
}

// offsetToLineCol converts a byte offset to line and column numbers.
func (e *linkExtractor) offsetToLineCol(offset int) (lineNum, colNum int) {
	lineNum = 1
	colNum = 1

	for i, lineStart := range e.lines {
		if offset < lineStart {
			// The offset is on the previous line
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

// extractRefDefs extracts reference-style link definitions from markdown content.
// These are lines in the format: [refname]: url
// Reference names are normalized to lowercase for case-insensitive matching.
// Uses bytes.IndexByte to iterate lines without allocating a slice of all lines.
func extractRefDefs(content []byte) map[string]refDef {
	defs := make(map[string]refDef, 8) // Pre-allocate for typical case
	lineNum := 1
	start := 0

	for start < len(content) {
		// Find end of current line
		end := bytes.IndexByte(content[start:], '\n')
		var line []byte
		if end == -1 {
			line = content[start:]
			start = len(content) // Will exit loop
		} else {
			line = content[start : start+end]
			start = start + end + 1
		}

		match := refDefRegex.FindSubmatch(line)
		if match != nil {
			name := strings.ToLower(string(match[1]))
			url := string(match[2])
			defs[name] = refDef{
				url:  url,
				line: lineNum,
			}
		}
		lineNum++
	}

	return defs
}
