// Package yaml implements a URL extractor for YAML files.
package yaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/leonardomso/gone/internal/parser"
	"gopkg.in/yaml.v3"
)

// Parser implements parser.FileParser for YAML files.
type Parser struct{}

// New creates a new YAML parser.
func New() *Parser {
	return &Parser{}
}

// Extensions returns the file extensions this parser handles.
func (*Parser) Extensions() []string {
	return []string{".yaml", ".yml"}
}

// ValidateAndParse validates the content and extracts links in a single pass.
func (*Parser) ValidateAndParse(filename string, content []byte) ([]parser.Link, error) {
	if len(content) == 0 {
		return nil, nil
	}

	extractor := &linkExtractor{
		filePath: filename,
		links:    make([]parser.Link, 0, 32),
	}

	// Parse all YAML documents in the file (single pass - validates and parses)
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		extractor.extractFromNode(&node, "")
	}

	return extractor.links, nil
}

// linkExtractor extracts URLs from YAML nodes.
type linkExtractor struct {
	filePath string
	links    []parser.Link
}

// extractFromNode recursively extracts URLs from a YAML node.
func (e *linkExtractor) extractFromNode(node *yaml.Node, path string) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node contains content nodes
		for _, content := range node.Content {
			e.extractFromNode(content, path)
		}

	case yaml.MappingNode:
		// Mapping nodes have alternating key/value content
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Check if key is a URL
			if keyNode.Kind == yaml.ScalarNode && parser.IsHTTPURL(keyNode.Value) {
				e.links = append(e.links, parser.Link{
					URL:      keyNode.Value,
					FilePath: e.filePath,
					Line:     keyNode.Line,
					Column:   keyNode.Column,
					Text:     path + ".<key>",
					Type:     parser.LinkTypeAutolink,
				})
			}

			// Build path for value
			keyStr := ""
			if keyNode.Kind == yaml.ScalarNode {
				keyStr = keyNode.Value
			}
			childPath := keyStr
			if path != "" {
				childPath = path + "." + keyStr
			}

			// Recurse into value
			e.extractFromNode(valueNode, childPath)
		}

	case yaml.SequenceNode:
		// Sequence (array) nodes
		for i, item := range node.Content {
			// Use string concatenation with strconv.Itoa instead of fmt.Sprintf for performance
			childPath := path + "[" + strconv.Itoa(i) + "]"
			e.extractFromNode(item, childPath)
		}

	case yaml.ScalarNode:
		// Check if scalar value is a URL or contains URLs
		e.extractURLsFromScalar(node, path)

	case yaml.AliasNode:
		// Follow alias to actual node
		if node.Alias != nil {
			e.extractFromNode(node.Alias, path)
		}
	}
}

// extractURLsFromScalar extracts URLs from a scalar (string) node.
func (e *linkExtractor) extractURLsFromScalar(node *yaml.Node, path string) {
	value := node.Value

	// Check if the entire value is a URL
	if parser.IsHTTPURL(value) {
		e.links = append(e.links, parser.Link{
			URL:      value,
			FilePath: e.filePath,
			Line:     node.Line,
			Column:   node.Column,
			Text:     path,
			Type:     parser.LinkTypeAutolink,
		})
		return
	}

	// Check if the value contains URLs (for multi-line strings or embedded URLs)
	matches := parser.URLRegex.FindAllStringIndex(value, -1)
	for _, match := range matches {
		url := value[match[0]:match[1]]
		url = parser.CleanURLTrailing(url)
		if !parser.IsHTTPURL(url) {
			continue
		}

		e.links = append(e.links, parser.Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     node.Line,
			Column:   node.Column,
			Text:     path,
			Type:     parser.LinkTypeAutolink,
		})
	}
}

// init registers the YAML parser with the default registry.
func init() {
	parser.RegisterParser(New())
}
