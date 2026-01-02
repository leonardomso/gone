// Package parser provides file parsers for extracting URLs from various file formats.
// It defines shared types, helpers, and multi-file processing functions used by all parsers.
package parser

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// LinkType represents the type of link found in a file.
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

// ParseError represents an error that occurred during file parsing.
// It includes the file path and the underlying error.
type ParseError struct {
	FilePath string
	Err      error
}

func (e *ParseError) Error() string {
	return e.FilePath + ": " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// fileResult holds the result of parsing a single file.
type fileResult struct {
	links []Link
	err   error
}

// =============================================================================
// Shared Helper Functions (used by all parsers)
// =============================================================================

// IsHTTPURL checks if a URL is an HTTP or HTTPS URL.
// Non-HTTP URLs (mailto, tel, file, anchors, etc.) are excluded from link checking.
// Exported for use by subpackage parsers.
func IsHTTPURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// URLRegex matches HTTP/HTTPS URLs.
// Exported for use by subpackage parsers.
var URLRegex = regexp.MustCompile(`https?://[^\s"'\]\}>,]+`)

// CleanURLTrailing removes trailing punctuation from URLs.
// Exported for use by subpackage parsers.
func CleanURLTrailing(url string) string {
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

// BuildLineIndex creates an index of byte offsets for the start of each line.
// This index enables O(log n) line/column lookups from byte offsets,
// which is more efficient than scanning from the start for each lookup.
// Exported for use by subpackage parsers.
func BuildLineIndex(content []byte) []int {
	// Estimate lines: assume avg 60 bytes per line, pre-allocate capacity
	estimatedLines := len(content)/60 + 1
	lines := make([]int, 1, estimatedLines)
	lines[0] = 0 // First line starts at offset 0

	for i, b := range content {
		if b == '\n' {
			lines = append(lines, i+1)
		}
	}

	return lines
}

// OffsetToLineCol converts a byte offset to 1-indexed line and column numbers
// using binary search for O(log n) performance.
// The lines parameter should be created by BuildLineIndex.
// Exported for use by subpackage parsers.
func OffsetToLineCol(lines []int, offset int) (lineNum, colNum int) {
	if len(lines) == 0 || offset < 0 {
		return 1, 1
	}

	// Binary search to find the line containing this offset
	// We want the largest index i where lines[i] <= offset
	lineIdx := sort.Search(len(lines), func(i int) bool {
		return lines[i] > offset
	}) - 1

	lineIdx = max(lineIdx, 0)

	lineNum = lineIdx + 1 // Convert to 1-indexed
	colNum = offset - lines[lineIdx] + 1

	return lineNum, colNum
}

// =============================================================================
// Multi-File Processing Functions
// =============================================================================

// ExtractLinksWithRegistry reads a file and returns all HTTP/HTTPS links found,
// using the appropriate parser from the registry based on file extension.
// If strict is true, validation errors will cause the function to return an error.
func ExtractLinksWithRegistry(filePath string, strict bool) ([]Link, error) {
	// Get parser for this file type
	p, ok := GetParserForFile(filePath)
	if !ok {
		return nil, &ParseError{
			FilePath: filePath,
			Err:      fmt.Errorf("no parser registered for file extension"),
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &ParseError{FilePath: filePath, Err: err}
	}

	// Validate and parse in a single pass for better performance
	links, err := p.ValidateAndParse(filePath, content)
	if err != nil {
		if strict {
			return nil, &ParseError{FilePath: filePath, Err: err}
		}
		// In non-strict mode, skip files with errors
		return nil, nil
	}

	return links, nil
}

// ExtractLinksFromMultipleFilesWithRegistry processes multiple files concurrently
// using the appropriate parser for each file type from the registry.
// If strict is true, validation errors will cause the function to return an error.
// Files with unsupported extensions are silently skipped.
func ExtractLinksFromMultipleFilesWithRegistry(filePaths []string, strict bool) ([]Link, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	// Filter to only supported files
	supportedFiles := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if _, ok := GetParserForFile(path); ok {
			supportedFiles = append(supportedFiles, path)
		}
	}

	if len(supportedFiles) == 0 {
		return nil, nil
	}

	// For small number of files, use sequential processing
	if len(supportedFiles) <= 2 {
		return extractLinksSequentialWithRegistry(supportedFiles, strict)
	}

	return extractLinksParallelWithRegistry(supportedFiles, strict)
}

// extractLinksSequentialWithRegistry processes files one at a time using the registry.
func extractLinksSequentialWithRegistry(filePaths []string, strict bool) ([]Link, error) {
	allLinks := make([]Link, 0, len(filePaths)*30)

	for _, path := range filePaths {
		links, err := ExtractLinksWithRegistry(path, strict)
		if err != nil {
			if strict {
				return nil, err
			}
			// In non-strict mode, skip files with errors
			continue
		}
		if links != nil {
			allLinks = append(allLinks, links...)
		}
	}

	return allLinks, nil
}

// extractLinksParallelWithRegistry processes files concurrently using the registry.
func extractLinksParallelWithRegistry(filePaths []string, strict bool) ([]Link, error) {
	numWorkers := min(runtime.NumCPU(), len(filePaths))

	type job struct {
		path   string
		strict bool
	}

	jobs := make(chan job, len(filePaths))
	results := make(chan fileResult, len(filePaths))

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for j := range jobs {
				links, err := ExtractLinksWithRegistry(j.path, j.strict)
				results <- fileResult{links: links, err: err}
			}
		})
	}

	// Send jobs
	for _, path := range filePaths {
		jobs <- job{path: path, strict: strict}
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	allLinks := make([]Link, 0, len(filePaths)*30)
	for result := range results {
		if result.err != nil {
			if strict {
				return nil, result.err
			}
			// In non-strict mode, skip files with errors
			continue
		}
		if result.links != nil {
			allLinks = append(allLinks, result.links...)
		}
	}

	return allLinks, nil
}
