// Package output provides formatting and file writing for link check reports.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gone/internal/checker"
)

// Format represents an output format type.
type Format string

const (
	// FormatJSON outputs as JSON.
	FormatJSON Format = "json"
	// FormatYAML outputs as YAML.
	FormatYAML Format = "yaml"
	// FormatXML outputs as generic XML.
	FormatXML Format = "xml"
	// FormatJUnit outputs as JUnit XML for CI/CD integration.
	FormatJUnit Format = "junit"
	// FormatMarkdown outputs as a Markdown report.
	FormatMarkdown Format = "markdown"
)

// ValidFormats returns all valid format strings.
func ValidFormats() []string {
	return []string{
		string(FormatJSON),
		string(FormatYAML),
		string(FormatXML),
		string(FormatJUnit),
		string(FormatMarkdown),
	}
}

// IsValidFormat checks if a format string is valid.
func IsValidFormat(s string) bool {
	switch Format(strings.ToLower(s)) {
	case FormatJSON, FormatYAML, FormatXML, FormatJUnit, FormatMarkdown:
		return true
	default:
		return false
	}
}

// IgnoredURL represents a URL that was ignored by filter rules.
type IgnoredURL struct {
	URL    string
	File   string
	Line   int
	Reason string // "domain", "pattern", or "regex"
	Rule   string // The rule that matched
}

// Report contains all data needed for output formatting.
type Report struct {
	GeneratedAt time.Time
	Files       []string
	TotalLinks  int
	UniqueURLs  int
	Summary     checker.Summary
	Results     []checker.Result
	Ignored     []IgnoredURL
}

// Formatter is the interface that output formatters implement.
type Formatter interface {
	Format(report *Report) ([]byte, error)
}

// GetFormatter returns the appropriate formatter for a format.
func GetFormatter(format Format) (Formatter, error) {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}, nil
	case FormatYAML:
		return &YAMLFormatter{}, nil
	case FormatXML:
		return &XMLFormatter{}, nil
	case FormatJUnit:
		return &JUnitFormatter{}, nil
	case FormatMarkdown:
		return &MarkdownFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// FormatReport formats a report using the specified format.
func FormatReport(report *Report, format Format) ([]byte, error) {
	formatter, err := GetFormatter(format)
	if err != nil {
		return nil, err
	}
	return formatter.Format(report)
}

// InferFormat determines the output format from a filename extension.
func InferFormat(filename string) (Format, error) {
	// Handle special case for JUnit
	if strings.HasSuffix(strings.ToLower(filename), ".junit.xml") {
		return FormatJUnit, nil
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return FormatJSON, nil
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".xml":
		return FormatXML, nil
	case ".md", ".markdown":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf(
			"cannot infer format from extension %q (supported: .json, .yaml, .yml, .xml, .junit.xml, .md, .markdown)",
			ext,
		)
	}
}

// WriteToFile writes a formatted report to a file.
func WriteToFile(report *Report, filename string) error {
	format, err := InferFormat(filename)
	if err != nil {
		return err
	}

	data, err := FormatReport(report, format)
	if err != nil {
		return fmt.Errorf("formatting report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
