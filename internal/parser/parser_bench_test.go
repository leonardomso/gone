package parser

import (
	"strings"
	"testing"
)

// BenchmarkBuildLineIndex measures line index construction.
func BenchmarkBuildLineIndex(b *testing.B) {
	content := []byte(strings.Repeat("This is a line of approximately eighty characters for testing purposes.\n", 1000))

	b.ResetTimer()
	for b.Loop() {
		BuildLineIndex(content)
	}
}

// BenchmarkBuildLineIndex_Small benchmarks with smaller content.
func BenchmarkBuildLineIndex_Small(b *testing.B) {
	content := []byte(strings.Repeat("Short line\n", 100))

	b.ResetTimer()
	for b.Loop() {
		BuildLineIndex(content)
	}
}

// BenchmarkIsHTTPURL measures URL protocol checking.
func BenchmarkIsHTTPURL(b *testing.B) {
	urls := []string{
		"http://example.com",
		"https://example.com",
		"mailto:test@example.com",
		"ftp://example.com",
		"#anchor",
		"/relative/path",
	}

	b.ResetTimer()
	for b.Loop() {
		for _, u := range urls {
			IsHTTPURL(u)
		}
	}
}

// BenchmarkCleanURLTrailing measures URL trailing cleanup.
func BenchmarkCleanURLTrailing(b *testing.B) {
	urls := []string{
		"http://example.com",
		"http://example.com.",
		"http://example.com,",
		"http://example.com.,;:)]}",
	}

	b.ResetTimer()
	for b.Loop() {
		for _, u := range urls {
			CleanURLTrailing(u)
		}
	}
}

// BenchmarkURLRegex measures URL regex matching.
func BenchmarkURLRegex(b *testing.B) {
	text := "Visit http://example.com and https://another.com for more info"

	b.ResetTimer()
	for b.Loop() {
		URLRegex.FindAllString(text, -1)
	}
}
