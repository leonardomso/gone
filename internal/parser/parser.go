// Package parser extracts URLs from text content
package parser

import (
	"os"
	"regexp"
)

// Link represents a URL found in a file
type Link struct {
	URL      string // The actual URL
	FilePath string // Which file it was found in
	Line     int    // Line number (1-indexed)
}

// urlRegex matches HTTP and HTTPS URLs
// Breaking down the pattern:
//   - https?://           - matches "http://" or "https://"
//   - [^\s\)\]\>\"\'<]+   - matches any characters except whitespace and common URL terminators
//
// We compile this once at package load time (more efficient than compiling per-call)
var urlRegex = regexp.MustCompile(`https?://[^\s\)\]\>\"\'<]+`)

// ExtractLinks reads a file and returns all HTTP/HTTPS links found
func ExtractLinks(filePath string) ([]Link, error) {
	// Read the entire file into memory
	// os.ReadFile returns []byte (a byte slice)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Convert bytes to string for processing
	text := string(content)

	// Find all matches - returns []string of all URLs found
	matches := urlRegex.FindAllString(text, -1)

	// Build our Link structs
	// We're not tracking line numbers yet (would need more complex parsing)
	var links []Link
	for _, url := range matches {
		links = append(links, Link{
			URL:      cleanURL(url),
			FilePath: filePath,
			Line:     0, // TODO: implement line tracking if needed
		})
	}

	return links, nil
}

// ExtractLinksFromMultipleFiles processes multiple files and returns all links
func ExtractLinksFromMultipleFiles(filePaths []string) ([]Link, error) {
	var allLinks []Link

	for _, path := range filePaths {
		links, err := ExtractLinks(path)
		if err != nil {
			// We could skip files with errors, but let's fail fast for now
			return nil, err
		}
		// append with ... spreads the slice (like JavaScript spread operator)
		allLinks = append(allLinks, links...)
	}

	return allLinks, nil
}

// cleanURL removes trailing punctuation that might have been captured
// For example: "https://example.com." should become "https://example.com"
func cleanURL(url string) string {
	// Remove common trailing characters that aren't part of URLs
	for len(url) > 0 {
		last := url[len(url)-1]
		// Check if last character is punctuation we want to strip
		if last == '.' || last == ',' || last == ';' || last == ':' {
			url = url[:len(url)-1] // Slice off the last character
		} else {
			break
		}
	}
	return url
}
