package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLParser implements FileParser for YAML files.
type YAMLParser struct{}

// NewYAMLParser creates a new YAML parser.
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Extensions returns the file extensions this parser handles.
func (*YAMLParser) Extensions() []string {
	return []string{".yaml", ".yml"}
}

// Validate checks if the content is valid YAML.
func (*YAMLParser) Validate(content []byte) error {
	if len(content) == 0 {
		return nil // Empty file is valid (no links to extract)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("invalid YAML: %w", err)
		}
	}
	return nil
}

// Parse extracts links from YAML content.
// It extracts URLs from both string values and mapping keys.
// Supports multi-document YAML files.
func (*YAMLParser) Parse(filename string, content []byte) ([]Link, error) {
	if len(content) == 0 {
		return nil, nil
	}

	extractor := &yamlLinkExtractor{
		filePath: filename,
		links:    make([]Link, 0, 32),
	}

	// Parse all YAML documents in the file
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

// yamlLinkExtractor extracts URLs from YAML nodes.
type yamlLinkExtractor struct {
	filePath string
	links    []Link
}

// extractFromNode recursively extracts URLs from a YAML node.
func (e *yamlLinkExtractor) extractFromNode(node *yaml.Node, path string) {
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
			if keyNode.Kind == yaml.ScalarNode && isHTTPURL(keyNode.Value) {
				e.links = append(e.links, Link{
					URL:      keyNode.Value,
					FilePath: e.filePath,
					Line:     keyNode.Line,
					Column:   keyNode.Column,
					Text:     path + ".<key>",
					Type:     LinkTypeAutolink,
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
			childPath := fmt.Sprintf("%s[%d]", path, i)
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
func (e *yamlLinkExtractor) extractURLsFromScalar(node *yaml.Node, path string) {
	value := node.Value

	// Check if the entire value is a URL
	if isHTTPURL(value) {
		e.links = append(e.links, Link{
			URL:      value,
			FilePath: e.filePath,
			Line:     node.Line,
			Column:   node.Column,
			Text:     path,
			Type:     LinkTypeAutolink,
		})
		return
	}

	// Check if the value contains URLs (for multi-line strings or embedded URLs)
	matches := urlRegex.FindAllStringIndex(value, -1)
	for _, match := range matches {
		url := value[match[0]:match[1]]
		url = cleanURLTrailing(url)
		if !isHTTPURL(url) {
			continue
		}

		e.links = append(e.links, Link{
			URL:      url,
			FilePath: e.filePath,
			Line:     node.Line,
			Column:   node.Column,
			Text:     path,
			Type:     LinkTypeAutolink,
		})
	}
}

// init registers the YAML parser with the default registry.
func init() {
	RegisterParser(NewYAMLParser())
}
