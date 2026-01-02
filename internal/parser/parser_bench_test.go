package parser

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkExtractRefDefs measures reference definition extraction.
// This includes regex matching and line splitting.
func BenchmarkExtractRefDefs(b *testing.B) {
	content := createContentWithRefDefs(100)

	b.ResetTimer()
	for b.Loop() {
		extractRefDefs(content)
	}
}

// BenchmarkExtractRefDefs_Small benchmarks with fewer definitions.
func BenchmarkExtractRefDefs_Small(b *testing.B) {
	content := createContentWithRefDefs(10)

	b.ResetTimer()
	for b.Loop() {
		extractRefDefs(content)
	}
}

// BenchmarkExtractRefDefs_Large benchmarks with many definitions.
func BenchmarkExtractRefDefs_Large(b *testing.B) {
	content := createContentWithRefDefs(500)

	b.ResetTimer()
	for b.Loop() {
		extractRefDefs(content)
	}
}

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

// BenchmarkExtractLinksFromContent measures the full parsing pipeline.
func BenchmarkExtractLinksFromContent(b *testing.B) {
	content := createMarkdownContent(50)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ExtractLinksFromContent(content, "test.md")
	}
}

// BenchmarkExtractLinksFromContent_Small benchmarks with fewer links.
func BenchmarkExtractLinksFromContent_Small(b *testing.B) {
	content := createMarkdownContent(10)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ExtractLinksFromContent(content, "test.md")
	}
}

// BenchmarkExtractLinksFromContent_Large benchmarks with many links.
func BenchmarkExtractLinksFromContent_Large(b *testing.B) {
	content := createMarkdownContent(200)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ExtractLinksFromContent(content, "test.md")
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

// createContentWithRefDefs creates content with reference definitions.
func createContentWithRefDefs(n int) []byte {
	var sb strings.Builder
	for range n {
		sb.WriteString("[ref")
		sb.WriteString(strconv.Itoa(n))
		sb.WriteString("]: http://example.com/page")
		sb.WriteString(strconv.Itoa(n))
		sb.WriteString("\n")
		// Add some regular text lines
		sb.WriteString("Some regular text content here.\n")
		sb.WriteString("Another line of text.\n")
	}
	return []byte(sb.String())
}

// createMarkdownContent creates realistic markdown with links.
func createMarkdownContent(numLinks int) []byte {
	var sb strings.Builder
	sb.WriteString("# Test Document\n\n")
	sb.WriteString("This is a test document with multiple links.\n\n")

	for i := range numLinks {
		sb.WriteString("## Section ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\n\n")
		sb.WriteString("[Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("](http://example.com/page")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(")\n\n")
		sb.WriteString("Some text explaining this section. ")
		sb.WriteString("It contains multiple sentences to simulate real content.\n\n")
	}

	// Add some reference-style links
	sb.WriteString("\n[Reference Link][ref1]\n")
	sb.WriteString("[ref1]: http://example.com/reference\n")

	return []byte(sb.String())
}
