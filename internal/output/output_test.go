package output

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/leonardomso/gone/internal/checker"
)

// =============================================================================
// Test Fixtures
// =============================================================================

func newTestReport() *Report {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	return &Report{
		GeneratedAt: now,
		Files:       []string{"README.md", "docs/guide.md"},
		TotalLinks:  10,
		UniqueURLs:  8,
		Summary: checker.Summary{
			Total:      10,
			UniqueURLs: 8,
			Alive:      5,
			Redirects:  2,
			Blocked:    1,
			Dead:       1,
			Errors:     1,
			Duplicates: 0,
		},
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://example.com", FilePath: "README.md", Line: 10, Text: "Example"},
				StatusCode: 200,
				Status:     checker.StatusAlive,
			},
			{
				Link: checker.Link{
					URL: "https://old.example.com", FilePath: "README.md", Line: 20, Text: "Old Link",
				},
				StatusCode: 301,
				Status:     checker.StatusRedirect,
				RedirectChain: []checker.Redirect{
					{URL: "https://old.example.com", StatusCode: 301},
				},
				FinalURL:    "https://new.example.com",
				FinalStatus: 200,
			},
			{
				Link: checker.Link{
					URL: "https://dead.example.com", FilePath: "docs/guide.md", Line: 5, Text: "Dead Link",
				},
				StatusCode: 404,
				Status:     checker.StatusDead,
			},
			{
				Link: checker.Link{
					URL: "https://error.example.com", FilePath: "docs/guide.md", Line: 15, Text: "Error Link",
				},
				Status: checker.StatusError,
				Error:  "connection refused",
			},
		},
		Ignored: []IgnoredURL{
			{
				URL:    "https://ignored.example.com",
				File:   "README.md",
				Line:   30,
				Reason: "domain",
				Rule:   "example.com",
			},
		},
	}
}

func newMinimalReport() *Report {
	return &Report{
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Files:       []string{},
		TotalLinks:  0,
		UniqueURLs:  0,
		Summary:     checker.Summary{},
		Results:     []checker.Result{},
	}
}

// =============================================================================
// Format Constants Tests
// =============================================================================

func TestValidFormats(t *testing.T) {
	t.Parallel()

	formats := ValidFormats()

	assert.Len(t, formats, 5)
	assert.Contains(t, formats, "json")
	assert.Contains(t, formats, "yaml")
	assert.Contains(t, formats, "xml")
	assert.Contains(t, formats, "junit")
	assert.Contains(t, formats, "markdown")
}

func TestIsValidFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format   string
		expected bool
	}{
		{"json", true},
		{"JSON", true},
		{"Json", true},
		{"yaml", true},
		{"YAML", true},
		{"xml", true},
		{"junit", true},
		{"markdown", true},
		{"md", false},
		{"txt", false},
		{"html", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsValidFormat(tt.format))
		})
	}
}

// =============================================================================
// GetFormatter Tests
// =============================================================================

func TestGetFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format       Format
		expectedType string
		hasError     bool
	}{
		{FormatJSON, "*output.JSONFormatter", false},
		{FormatYAML, "*output.YAMLFormatter", false},
		{FormatXML, "*output.XMLFormatter", false},
		{FormatJUnit, "*output.JUnitFormatter", false},
		{FormatMarkdown, "*output.MarkdownFormatter", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			t.Parallel()

			formatter, err := GetFormatter(tt.format)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, formatter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, formatter)
			}
		})
	}
}

// =============================================================================
// InferFormat Tests
// =============================================================================

func TestInferFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected Format
		hasError bool
	}{
		// JSON
		{"report.json", FormatJSON, false},
		{"REPORT.JSON", FormatJSON, false},
		{"path/to/report.json", FormatJSON, false},

		// YAML
		{"report.yaml", FormatYAML, false},
		{"report.yml", FormatYAML, false},
		{"REPORT.YAML", FormatYAML, false},

		// XML
		{"report.xml", FormatXML, false},
		{"REPORT.XML", FormatXML, false},

		// JUnit (special case)
		{"report.junit.xml", FormatJUnit, false},
		{"REPORT.JUNIT.XML", FormatJUnit, false},
		{"path/to/report.junit.xml", FormatJUnit, false},

		// Markdown
		{"report.md", FormatMarkdown, false},
		{"report.markdown", FormatMarkdown, false},
		{"REPORT.MD", FormatMarkdown, false},

		// Errors
		{"report.txt", "", true},
		{"report.html", "", true},
		{"report", "", true},
		{".json", FormatJSON, false}, // Edge case: just extension
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()

			format, err := InferFormat(tt.filename)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, format)
			}
		})
	}
}

// =============================================================================
// FormatReport Tests
// =============================================================================

func TestFormatReport(t *testing.T) {
	t.Parallel()

	report := newTestReport()

	tests := []struct {
		format   Format
		contains string
	}{
		{FormatJSON, `"url": "https://example.com"`},
		{FormatYAML, "url: https://example.com"},
		{FormatXML, "<url>https://example.com</url>"},
		{FormatJUnit, `name="gone-link-check"`},
		{FormatMarkdown, "# Gone Link Check Report"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			t.Parallel()

			data, err := FormatReport(report, tt.format)

			require.NoError(t, err)
			assert.Contains(t, string(data), tt.contains)
		})
	}
}

func TestFormatReport_UnknownFormat(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	_, err := FormatReport(report, "unknown")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
}

// =============================================================================
// WriteToFile Tests
// =============================================================================

func TestWriteToFile(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	tmpDir := t.TempDir()

	tests := []struct {
		filename string
		check    func(t *testing.T, data []byte)
	}{
		{
			filename: "report.json",
			check: func(t *testing.T, data []byte) {
				var obj map[string]any
				require.NoError(t, json.Unmarshal(data, &obj))
				assert.Equal(t, float64(10), obj["total_links"])
			},
		},
		{
			filename: "report.yaml",
			check: func(t *testing.T, data []byte) {
				var obj map[string]any
				require.NoError(t, yaml.Unmarshal(data, &obj))
				assert.Equal(t, 10, obj["total_links"])
			},
		},
		{
			filename: "report.xml",
			check: func(t *testing.T, data []byte) {
				assert.True(t, strings.HasPrefix(string(data), "<?xml"))
			},
		},
		{
			filename: "report.junit.xml",
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), "testsuites")
			},
		},
		{
			filename: "report.md",
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), "# Gone Link Check Report")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(tmpDir, tt.filename)
			err := WriteToFile(report, path)

			require.NoError(t, err)

			data, err := os.ReadFile(path)
			require.NoError(t, err)

			tt.check(t, data)
		})
	}
}

func TestWriteToFile_InvalidFormat(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.txt")

	err := WriteToFile(report, path)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot infer format")
}

func TestWriteToFile_InvalidPath(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	err := WriteToFile(report, "/nonexistent/path/report.json")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writing file")
}

// =============================================================================
// JSONFormatter Tests
// =============================================================================

func TestJSONFormatter_Format(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	formatter := &JSONFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Parse the JSON
	var output jsonOutput
	require.NoError(t, json.Unmarshal(data, &output))

	// Check basic fields
	assert.Equal(t, "2024-01-15T10:30:00Z", output.GeneratedAt)
	assert.Equal(t, 2, output.TotalFiles)
	assert.Equal(t, 10, output.TotalLinks)
	assert.Equal(t, 8, output.UniqueURLs)

	// Check summary
	assert.Equal(t, 5, output.Summary.Alive)
	assert.Equal(t, 2, output.Summary.Redirects)
	assert.Equal(t, 1, output.Summary.Blocked)
	assert.Equal(t, 1, output.Summary.Dead)
	assert.Equal(t, 1, output.Summary.Errors)
	assert.Equal(t, 1, output.Summary.Ignored)

	// Check results
	assert.Len(t, output.Results, 4)

	// Check redirect result
	var redirectResult *jsonResult
	for i := range output.Results {
		if output.Results[i].Status == "redirect" {
			redirectResult = &output.Results[i]
			break
		}
	}
	require.NotNil(t, redirectResult)
	assert.Len(t, redirectResult.RedirectChain, 1)
	assert.Equal(t, "https://new.example.com", redirectResult.FinalURL)
	assert.Equal(t, 200, redirectResult.FinalStatus)

	// Check ignored
	assert.Len(t, output.Ignored, 1)
	assert.Equal(t, "https://ignored.example.com", output.Ignored[0].URL)
}

func TestJSONFormatter_Format_EmptyReport(t *testing.T) {
	t.Parallel()

	report := newMinimalReport()
	formatter := &JSONFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output jsonOutput
	require.NoError(t, json.Unmarshal(data, &output))

	assert.Equal(t, 0, output.TotalFiles)
	assert.Empty(t, output.Results)
	assert.Empty(t, output.Ignored)
}

func TestJSONFormatter_Format_WithDuplicate(t *testing.T) {
	t.Parallel()

	primaryResult := checker.Result{
		Link:       checker.Link{URL: "https://example.com", FilePath: "a.md", Line: 1},
		Status:     checker.StatusAlive,
		StatusCode: 200,
	}

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			primaryResult,
			{
				Link:        checker.Link{URL: "https://example.com", FilePath: "b.md", Line: 5},
				Status:      checker.StatusDuplicate,
				DuplicateOf: &primaryResult,
			},
		},
	}

	formatter := &JSONFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output jsonOutput
	require.NoError(t, json.Unmarshal(data, &output))

	// Find duplicate result
	var dupResult *jsonResult
	for i := range output.Results {
		if output.Results[i].Status == "duplicate" {
			dupResult = &output.Results[i]
			break
		}
	}
	require.NotNil(t, dupResult)
	assert.Equal(t, "https://example.com", dupResult.DuplicateOf)
}

// =============================================================================
// YAMLFormatter Tests
// =============================================================================

func TestYAMLFormatter_Format(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	formatter := &YAMLFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Parse the YAML
	var output yamlOutput
	require.NoError(t, yaml.Unmarshal(data, &output))

	// Check basic fields
	assert.Equal(t, "2024-01-15T10:30:00Z", output.GeneratedAt)
	assert.Equal(t, 2, output.TotalFiles)
	assert.Equal(t, 10, output.TotalLinks)

	// Check results
	assert.Len(t, output.Results, 4)

	// Check ignored
	assert.Len(t, output.Ignored, 1)
}

func TestYAMLFormatter_Format_EmptyReport(t *testing.T) {
	t.Parallel()

	report := newMinimalReport()
	formatter := &YAMLFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output yamlOutput
	require.NoError(t, yaml.Unmarshal(data, &output))

	assert.Equal(t, 0, output.TotalFiles)
	assert.Empty(t, output.Results)
}

// =============================================================================
// XMLFormatter Tests
// =============================================================================

func TestXMLFormatter_Format(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	formatter := &XMLFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Check XML header
	assert.True(t, strings.HasPrefix(string(data), xml.Header))

	// Parse the XML (skip header)
	var output xmlOutput
	xmlData := strings.TrimPrefix(string(data), xml.Header)
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	// Check attributes
	assert.Equal(t, "2024-01-15T10:30:00Z", output.GeneratedAt)
	assert.Equal(t, 2, output.TotalFiles)
	assert.Equal(t, 10, output.TotalLinks)

	// Check results
	assert.Len(t, output.Results.Results, 4)

	// Check ignored
	require.NotNil(t, output.Ignored)
	assert.Len(t, output.Ignored.Items, 1)
}

func TestXMLFormatter_Format_EmptyReport(t *testing.T) {
	t.Parallel()

	report := newMinimalReport()
	formatter := &XMLFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(data), xml.Header))
}

func TestXMLFormatter_Format_NoIgnored(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	report.Ignored = nil
	formatter := &XMLFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Parse and verify ignored is nil
	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output xmlOutput
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))
	assert.Nil(t, output.Ignored)
}

// =============================================================================
// JUnitFormatter Tests
// =============================================================================

func TestJUnitFormatter_Format(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	formatter := &JUnitFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Check XML header
	assert.True(t, strings.HasPrefix(string(data), xml.Header))

	// Parse the XML
	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output junitTestSuites
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	// Check root element
	assert.Equal(t, "gone-link-check", output.Name)

	// JUnit only includes dead/error results
	assert.Equal(t, 2, output.Tests)
	assert.Equal(t, 1, output.Failures) // dead
	assert.Equal(t, 1, output.Errors)   // error

	// Find test suites - should be grouped by file
	assert.NotEmpty(t, output.TestSuite)
}

func TestJUnitFormatter_Format_AllLinksAlive(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://example.com", FilePath: "README.md", Line: 1},
				Status:     checker.StatusAlive,
				StatusCode: 200,
			},
		},
	}
	formatter := &JUnitFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	// Parse the XML
	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output junitTestSuites
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	// Should have empty test suite
	assert.Equal(t, 0, output.Tests)
	assert.Equal(t, 0, output.Failures)
	assert.Equal(t, 0, output.Errors)

	// Still should have a test suite (empty marker)
	assert.Len(t, output.TestSuite, 1)
	assert.Equal(t, "all-links", output.TestSuite[0].Name)
}

func TestJUnitFormatter_Format_DeadLinks(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://dead1.com", FilePath: "a.md", Line: 1, Text: "Link 1"},
				Status:     checker.StatusDead,
				StatusCode: 404,
			},
			{
				Link:       checker.Link{URL: "https://dead2.com", FilePath: "a.md", Line: 5, Text: "Link 2"},
				Status:     checker.StatusDead,
				StatusCode: 500,
			},
		},
	}
	formatter := &JUnitFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output junitTestSuites
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	assert.Equal(t, 2, output.Tests)
	assert.Equal(t, 2, output.Failures)
	assert.Equal(t, 0, output.Errors)

	// Check test case details
	require.Len(t, output.TestSuite, 1)
	suite := output.TestSuite[0]
	assert.Equal(t, "a.md", suite.Name)
	assert.Len(t, suite.TestCases, 2)

	// Check failure details
	tc := suite.TestCases[0]
	assert.Equal(t, "https://dead1.com", tc.Name)
	assert.Contains(t, tc.ClassName, "a.md:1")
	require.NotNil(t, tc.Failure)
	assert.Contains(t, tc.Failure.Message, "404")
}

func TestJUnitFormatter_Format_ErrorLinks(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:   checker.Link{URL: "https://error.com", FilePath: "b.md", Line: 10, Text: "Error Link"},
				Status: checker.StatusError,
				Error:  "connection timeout",
			},
		},
	}
	formatter := &JUnitFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output junitTestSuites
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	assert.Equal(t, 1, output.Tests)
	assert.Equal(t, 0, output.Failures)
	assert.Equal(t, 1, output.Errors)

	require.Len(t, output.TestSuite, 1)
	require.Len(t, output.TestSuite[0].TestCases, 1)

	tc := output.TestSuite[0].TestCases[0]
	require.NotNil(t, tc.Error)
	assert.Contains(t, tc.Error.Message, "connection timeout")
}

// =============================================================================
// MarkdownFormatter Tests
// =============================================================================

func TestMarkdownFormatter_Format(t *testing.T) {
	t.Parallel()

	report := newTestReport()
	formatter := &MarkdownFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)

	// Check header
	assert.Contains(t, content, "# Gone Link Check Report")
	assert.Contains(t, content, "**Generated:**")
	assert.Contains(t, content, "**Files Scanned:** 2")
	assert.Contains(t, content, "**Total Links:** 10")

	// Check summary table
	assert.Contains(t, content, "## Summary")
	assert.Contains(t, content, "| Status | Count |")
	assert.Contains(t, content, "| Alive | 5 |")

	// Check dead links section
	assert.Contains(t, content, "## Dead Links (2)")
	assert.Contains(t, content, "https://dead.example.com")
	assert.Contains(t, content, "https://error.example.com")

	// Check warnings section
	assert.Contains(t, content, "## Warnings")
	assert.Contains(t, content, "https://old.example.com")

	// Check ignored section
	assert.Contains(t, content, "## Ignored URLs (1)")
	assert.Contains(t, content, "https://ignored.example.com")
}

func TestMarkdownFormatter_Format_NoDeadLinks(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Files:       []string{"README.md"},
		TotalLinks:  1,
		UniqueURLs:  1,
		Summary:     checker.Summary{Alive: 1},
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://example.com", FilePath: "README.md", Line: 1},
				Status:     checker.StatusAlive,
				StatusCode: 200,
			},
		},
	}
	formatter := &MarkdownFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "## Dead Links")
}

func TestMarkdownFormatter_Format_NoWarnings(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Files:       []string{"README.md"},
		TotalLinks:  1,
		UniqueURLs:  1,
		Summary:     checker.Summary{Dead: 1},
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://dead.com", FilePath: "README.md", Line: 1},
				Status:     checker.StatusDead,
				StatusCode: 404,
			},
		},
	}
	formatter := &MarkdownFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "## Dead Links")
	assert.NotContains(t, content, "## Warnings")
}

func TestMarkdownFormatter_Format_Duplicates(t *testing.T) {
	t.Parallel()

	primaryResult := checker.Result{
		Link:       checker.Link{URL: "https://example.com", FilePath: "a.md", Line: 1},
		Status:     checker.StatusAlive,
		StatusCode: 200,
	}

	report := &Report{
		GeneratedAt: time.Now(),
		Summary:     checker.Summary{Duplicates: 1},
		Results: []checker.Result{
			primaryResult,
			{
				Link:        checker.Link{URL: "https://example.com", FilePath: "b.md", Line: 5},
				Status:      checker.StatusDuplicate,
				DuplicateOf: &primaryResult,
			},
		},
	}
	formatter := &MarkdownFormatter{}

	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "## Duplicates (1)")
	assert.Contains(t, content, "b.md")
	assert.Contains(t, content, "a.md:1")
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestTruncateText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long text", 10, "this is..."},
		{"", 10, ""},
		{"   spaced   ", 10, "spaced"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateText(tt.input, tt.maxLen))
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text|with|pipes", "text\\|with\\|pipes"},
		{"text`with`backticks", "text\\`with\\`backticks"},
		{"|`both`|", "\\|\\`both\\`\\|"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, escapeMarkdown(tt.input))
		})
	}
}

func TestTruncateForXML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is very long text", 10, "this is..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateForXML(tt.input, tt.maxLen))
		})
	}
}

func TestFilterByStatus(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{Status: checker.StatusAlive},
		{Status: checker.StatusDead},
		{Status: checker.StatusError},
		{Status: checker.StatusRedirect},
		{Status: checker.StatusDead},
	}

	// Single status
	dead := filterByStatus(results, checker.StatusDead)
	assert.Len(t, dead, 2)

	// Multiple statuses
	problems := filterByStatus(results, checker.StatusDead, checker.StatusError)
	assert.Len(t, problems, 3)

	// No matches
	blocked := filterByStatus(results, checker.StatusBlocked)
	assert.Empty(t, blocked)
}

func TestFormatStatusForMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result   checker.Result
		expected string
	}{
		{
			result:   checker.Result{Status: checker.StatusDead, StatusCode: 404},
			expected: "`404`",
		},
		{
			result:   checker.Result{Status: checker.StatusDead, StatusCode: 0},
			expected: "DEAD",
		},
		{
			result:   checker.Result{Status: checker.StatusError},
			expected: "ERROR",
		},
		{
			result:   checker.Result{Status: checker.StatusAlive},
			expected: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatStatusForMarkdown(tt.result))
		})
	}
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestJSONFormatter_Format_LongRedirectChain(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://start.com", FilePath: "test.md", Line: 1},
				Status:     checker.StatusRedirect,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://start.com", StatusCode: 301},
					{URL: "https://hop1.com", StatusCode: 302},
					{URL: "https://hop2.com", StatusCode: 307},
				},
				FinalURL:    "https://final.com",
				FinalStatus: 200,
			},
		},
	}

	formatter := &JSONFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output jsonOutput
	require.NoError(t, json.Unmarshal(data, &output))

	require.Len(t, output.Results, 1)
	assert.Len(t, output.Results[0].RedirectChain, 3)
}

func TestMarkdownFormatter_Format_SpecialCharacters(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Summary:     checker.Summary{Dead: 1},
		Results: []checker.Result{
			{
				Link: checker.Link{
					URL: "https://example.com?foo=bar|baz", FilePath: "test.md", Line: 1, Text: "Link with `code`",
				},
				Status:     checker.StatusDead,
				StatusCode: 404,
			},
		},
	}

	formatter := &MarkdownFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	// Pipes should be escaped
	assert.Contains(t, content, "\\|")
}

// =============================================================================
// Additional Output Tests for Coverage
// =============================================================================

func TestJUnitFormatter_Format_DeadLinkNoStatusCode(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://dead.com", FilePath: "test.md", Line: 1},
				Status:     checker.StatusDead,
				StatusCode: 0, // No status code
			},
		},
	}

	formatter := &JUnitFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Link is dead")
}

func TestJUnitFormatter_Format_DeadLinkWithRedirectChain(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link: checker.Link{
					URL: "https://start.com", FilePath: "test.md", Line: 1, Text: "Link Text",
				},
				Status:     checker.StatusDead,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://start.com", StatusCode: 301},
				},
				FinalURL:    "https://dead.com",
				FinalStatus: 404,
			},
		},
	}

	formatter := &JUnitFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Final URL")
	assert.Contains(t, content, "Link text")
}

func TestJUnitFormatter_Format_ErrorLinkNoText(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:   checker.Link{URL: "https://error.com", FilePath: "test.md", Line: 1, Text: ""},
				Status: checker.StatusError,
				Error:  "connection timeout",
			},
		},
	}

	formatter := &JUnitFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "connection timeout")
	assert.NotContains(t, content, "Link text")
}

func TestYAMLFormatter_Format_WithRedirectChain(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 1},
				Status:     checker.StatusRedirect,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://old.com", StatusCode: 301},
					{URL: "https://middle.com", StatusCode: 302},
				},
				FinalURL:    "https://new.com",
				FinalStatus: 200,
			},
		},
	}

	formatter := &YAMLFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output yamlOutput
	require.NoError(t, yaml.Unmarshal(data, &output))

	require.Len(t, output.Results, 1)
	assert.Len(t, output.Results[0].RedirectChain, 2)
	assert.Equal(t, "https://new.com", output.Results[0].FinalURL)
	assert.Equal(t, 200, output.Results[0].FinalStatus)
}

func TestYAMLFormatter_Format_WithDuplicate(t *testing.T) {
	t.Parallel()

	primaryResult := checker.Result{
		Link:       checker.Link{URL: "https://example.com", FilePath: "a.md", Line: 1},
		Status:     checker.StatusAlive,
		StatusCode: 200,
	}

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			primaryResult,
			{
				Link:        checker.Link{URL: "https://example.com", FilePath: "b.md", Line: 5},
				Status:      checker.StatusDuplicate,
				DuplicateOf: &primaryResult,
			},
		},
	}

	formatter := &YAMLFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	var output yamlOutput
	require.NoError(t, yaml.Unmarshal(data, &output))

	require.Len(t, output.Results, 2)

	// Find duplicate result
	var dupResult *yamlResult
	for i := range output.Results {
		if output.Results[i].Status == "duplicate" {
			dupResult = &output.Results[i]
			break
		}
	}
	require.NotNil(t, dupResult)
	assert.Equal(t, "https://example.com", dupResult.DuplicateOf)
}

func TestXMLFormatter_Format_WithRedirectChain(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 1, Text: "Old Link"},
				Status:     checker.StatusRedirect,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://old.com", StatusCode: 301},
				},
				FinalURL:    "https://new.com",
				FinalStatus: 200,
			},
		},
	}

	formatter := &XMLFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "redirect_chain")
	assert.Contains(t, content, "final_url")
	assert.Contains(t, content, "https://new.com")
}

func TestXMLFormatter_Format_WithDuplicate(t *testing.T) {
	t.Parallel()

	primaryResult := checker.Result{
		Link:       checker.Link{URL: "https://example.com", FilePath: "a.md", Line: 1},
		Status:     checker.StatusAlive,
		StatusCode: 200,
	}

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			primaryResult,
			{
				Link:        checker.Link{URL: "https://example.com", FilePath: "b.md", Line: 5},
				Status:      checker.StatusDuplicate,
				DuplicateOf: &primaryResult,
			},
		},
	}

	formatter := &XMLFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "duplicate_of")
}

func TestMarkdownFormatter_Format_BlockedLinks(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Summary:     checker.Summary{Blocked: 1, Redirects: 1},
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://blocked.com", FilePath: "test.md", Line: 1},
				Status:     checker.StatusBlocked,
				StatusCode: 403,
			},
			{
				Link:       checker.Link{URL: "https://redirect.com", FilePath: "test.md", Line: 5, Text: "Redirect"},
				Status:     checker.StatusRedirect,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://redirect.com", StatusCode: 301},
				},
				FinalURL:    "https://final.com",
				FinalStatus: 200,
			},
		},
	}

	formatter := &MarkdownFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "## Warnings")
	assert.Contains(t, content, "BLOCKED")
	assert.Contains(t, content, "REDIRECT")
	assert.Contains(t, content, "### Redirect Details")
}

func TestMarkdownFormatter_Format_DeadLinkWithRedirectChain(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Summary:     checker.Summary{Dead: 1},
		Results: []checker.Result{
			{
				Link: checker.Link{
					URL: "https://start.com", FilePath: "test.md", Line: 1, Text: "Link",
				},
				Status:     checker.StatusDead,
				StatusCode: 301,
				RedirectChain: []checker.Redirect{
					{URL: "https://start.com", StatusCode: 301},
					{URL: "https://middle.com", StatusCode: 302},
				},
				FinalURL:    "https://dead.com",
				FinalStatus: 404,
			},
		},
	}

	formatter := &MarkdownFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "## Dead Links")
	assert.Contains(t, content, "Redirect Chain")
	assert.Contains(t, content, "Final:")
}

func TestMarkdownFormatter_Format_ErrorWithMessage(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Summary:     checker.Summary{Errors: 1},
		Results: []checker.Result{
			{
				Link:   checker.Link{URL: "https://error.com", FilePath: "test.md", Line: 1},
				Status: checker.StatusError,
				Error:  "DNS lookup failed",
			},
		},
	}

	formatter := &MarkdownFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "DNS lookup failed")
	assert.Contains(t, content, "ERROR")
}

func TestJUnitFormatter_Format_MultipleFilesWithErrors(t *testing.T) {
	t.Parallel()

	report := &Report{
		GeneratedAt: time.Now(),
		Results: []checker.Result{
			{
				Link:       checker.Link{URL: "https://dead1.com", FilePath: "a.md", Line: 1},
				Status:     checker.StatusDead,
				StatusCode: 404,
			},
			{
				Link:       checker.Link{URL: "https://dead2.com", FilePath: "a.md", Line: 5},
				Status:     checker.StatusDead,
				StatusCode: 500,
			},
			{
				Link:   checker.Link{URL: "https://error.com", FilePath: "b.md", Line: 1},
				Status: checker.StatusError,
				Error:  "timeout",
			},
		},
	}

	formatter := &JUnitFormatter{}
	data, err := formatter.Format(report)
	require.NoError(t, err)

	xmlData := strings.TrimPrefix(string(data), xml.Header)
	var output junitTestSuites
	require.NoError(t, xml.Unmarshal([]byte(xmlData), &output))

	assert.Equal(t, 3, output.Tests)
	assert.Equal(t, 2, output.Failures)
	assert.Equal(t, 1, output.Errors)

	// Should have 2 test suites (a.md and b.md)
	assert.Len(t, output.TestSuite, 2)
}

func TestBuildFailureContent_NoLinkText(t *testing.T) {
	t.Parallel()

	result := checker.Result{
		Link:       checker.Link{URL: "https://example.com", Text: ""},
		Status:     checker.StatusDead,
		StatusCode: 404,
	}

	content := buildFailureContent(result)
	assert.NotContains(t, content, "Link text")
	assert.Contains(t, content, "Status: 404")
}

func TestBuildFailureContent_NoStatusCode(t *testing.T) {
	t.Parallel()

	result := checker.Result{
		Link:       checker.Link{URL: "https://example.com", Text: "My Link"},
		Status:     checker.StatusDead,
		StatusCode: 0,
	}

	content := buildFailureContent(result)
	assert.Contains(t, content, "Link text")
	assert.NotContains(t, content, "Status:")
}

func TestBuildErrorContent_WithLinkText(t *testing.T) {
	t.Parallel()

	result := checker.Result{
		Link:   checker.Link{URL: "https://example.com", Text: "My Link"},
		Status: checker.StatusError,
		Error:  "connection refused",
	}

	content := buildErrorContent(result)
	assert.Contains(t, content, "Link text")
	assert.Contains(t, content, "connection refused")
}

func TestBuildErrorContent_NoLinkText(t *testing.T) {
	t.Parallel()

	result := checker.Result{
		Link:   checker.Link{URL: "https://example.com", Text: ""},
		Status: checker.StatusError,
		Error:  "timeout",
	}

	content := buildErrorContent(result)
	assert.NotContains(t, content, "Link text")
	assert.Contains(t, content, "timeout")
}
