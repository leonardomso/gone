// Package parser extracts URLs from markdown content using goldmark.
package parser

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// LinkType represents the type of link found in markdown.
type LinkType int

const (
	// LinkTypeInline represents a standard markdown link: [text](url).
	LinkTypeInline LinkType = iota
	// LinkTypeReference represents a reference-style link: [text][ref] with [ref]: url.
	LinkTypeReference
	// LinkTypeImage represents an image: ![alt](url).
	LinkTypeImage
	// LinkTypeAutolink represents a bare URL that's auto-linked.
	LinkTypeAutolink
	// LinkTypeHTML represents a link in HTML: <a href="url">.
	LinkTypeHTML
)

// String returns the string representation of a LinkType.
func (t LinkType) String() string {
	switch t {
	case LinkTypeInline:
		return "inline"
	case LinkTypeReference:
		return "reference"
	case LinkTypeImage:
		return "image"
	case LinkTypeAutolink:
		return "autolink"
	case LinkTypeHTML:
		return "html"
	default:
		return "unknown"
	}
}

// Link represents a URL found in a file.
type Link struct {
	URL      string // The actual URL
	FilePath string // Which file it was found in
	Text     string // Link text or alt text for images

	// For reference links.
	RefName string   // Reference name (e.g., "myref" in [text][myref])
	Line    int      // Line number (1-indexed)
	Column  int      // Column position (1-indexed)
	Type    LinkType // Type of link

	RefDefLine int // Line where [ref]: url is defined (0 if not reference)
}

// linkExtractor walks the AST and extracts links.
type linkExtractor struct {

	// Track reference definitions: name -> (url, line)
	refDefs  map[string]refDef
	filePath string
	links    []Link
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

// ExtractLinks reads a file and returns all HTTP/HTTPS links found.
func ExtractLinks(filePath string) ([]Link, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ExtractLinksFromContent(content, filePath)
}

// ExtractLinksFromContent extracts links from markdown content.
func ExtractLinksFromContent(content []byte, filePath string) ([]Link, error) {
	// Create goldmark parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify, // Auto-link bare URLs
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// Parse the markdown into AST
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader)

	// Build line offset index for position calculation
	lines := buildLineIndex(content)

	// Extract reference definitions first
	refDefs := extractRefDefs(content)

	// Create extractor and walk the AST
	extractor := &linkExtractor{
		links:    []Link{},
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
	if !isHTTPURL(linkURL) {
		return
	}

	// Get link text from children
	linkText := e.getNodeText(node)

	// Calculate position
	line, col := e.getPosition(node)

	link := Link{
		URL:      linkURL,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     linkText,
		Type:     LinkTypeInline,
	}

	// Check reference definitions for this URL
	for refName, refDef := range e.refDefs {
		if refDef.url == linkURL && refDef.line != line {
			link.Type = LinkTypeReference
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
	if !isHTTPURL(imageURL) {
		return
	}

	// Get alt text from node children
	altText := e.getNodeText(node)

	line, col := e.getPosition(node)

	e.links = append(e.links, Link{
		URL:      imageURL,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     altText,
		Type:     LinkTypeImage,
	})
}

// handleAutoLink processes a bare URL that's auto-linked.
func (e *linkExtractor) handleAutoLink(node *ast.AutoLink) {
	url := string(node.URL(e.source))

	// Skip non-HTTP URLs
	if !isHTTPURL(url) {
		return
	}

	line, col := e.getPosition(node)

	e.links = append(e.links, Link{
		URL:      url,
		FilePath: e.filePath,
		Line:     line,
		Column:   col,
		Text:     "", // Auto-links don't have separate text
		Type:     LinkTypeAutolink,
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
		if !isHTTPURL(url) {
			continue
		}

		// Calculate line and column from byte offset
		line, col := e.offsetToLineCol(match[0])

		e.links = append(e.links, Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     line,
			Column:   col,
			Text:     linkText,
			Type:     LinkTypeHTML,
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

// buildLineIndex creates an index of byte offsets for the start of each line.
func buildLineIndex(content []byte) []int {
	lines := []int{0} // First line starts at offset 0

	for i, b := range content {
		if b == '\n' {
			lines = append(lines, i+1)
		}
	}

	return lines
}

// extractRefDefs extracts reference definitions from the content.
// Format: [refname]: url.
func extractRefDefs(content []byte) map[string]refDef {
	defs := map[string]refDef{}
	lines := bytes.Split(content, []byte("\n"))

	refDefRegex := regexp.MustCompile(`^\s*\[([^\]]+)\]:\s*(\S+)`)

	for i, line := range lines {
		match := refDefRegex.FindSubmatch(line)
		if match != nil {
			name := strings.ToLower(string(match[1]))
			url := string(match[2])
			defs[name] = refDef{
				url:  url,
				line: i + 1, // 1-indexed
			}
		}
	}

	return defs
}

// isHTTPURL checks if a URL is an HTTP or HTTPS URL.
func isHTTPURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// ExtractLinksFromMultipleFiles processes multiple files and returns all links.
func ExtractLinksFromMultipleFiles(filePaths []string) ([]Link, error) {
	var allLinks []Link

	for _, path := range filePaths {
		links, err := ExtractLinks(path)
		if err != nil {
			return nil, err
		}
		allLinks = append(allLinks, links...)
	}

	return allLinks, nil
}
