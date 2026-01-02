package markdown

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkValidateAndParse measures Markdown parsing performance.
func BenchmarkValidateAndParse(b *testing.B) {
	content := createMarkdownContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ValidateAndParse("test.md", content)
	}
}

// BenchmarkExtractLinksFromContent measures direct link extraction.
func BenchmarkExtractLinksFromContent(b *testing.B) {
	content := createMarkdownContent(50)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ExtractLinksFromContent(content, "test.md")
	}
}

// createMarkdownContent creates a Markdown document with the specified number of URLs.
func createMarkdownContent(numURLs int) []byte {
	var sb strings.Builder
	sb.WriteString("# Test Project\n\n")
	sb.WriteString("This is a test project with multiple links.\n\n")
	sb.WriteString("## Links\n\n")

	for i := range numURLs {
		sb.WriteString("- [Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("](https://example.com/page/")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(")\n")
	}

	sb.WriteString("\n## Images\n\n")
	sb.WriteString("![Logo](https://example.com/logo.png)\n\n")
	sb.WriteString("## References\n\n")
	sb.WriteString("[Homepage][home]\n\n")
	sb.WriteString("[home]: https://github.com/example/project\n")

	return []byte(sb.String())
}
