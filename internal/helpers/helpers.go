// Package helpers provides shared utility functions used across the application.
// These are generic helpers that don't belong to a specific domain package.
package helpers

import "strings"

// TruncateText shortens text to the specified maximum length, adding "..." if truncated.
// Returns empty string if input is empty or only whitespace.
func TruncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// TruncateURL shortens a URL to the specified maximum length for display purposes.
// Adds "..." suffix if the URL exceeds maxLen.
func TruncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// CountUniqueStrings returns the number of unique strings in a slice.
// Useful for counting unique URLs or other string collections.
func CountUniqueStrings(items []string) int {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		seen[item] = true
	}
	return len(seen)
}
