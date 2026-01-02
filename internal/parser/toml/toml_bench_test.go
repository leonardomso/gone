package toml

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkParse measures TOML parsing performance.
func BenchmarkParse(b *testing.B) {
	content := createTOMLContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.Parse("test.toml", content)
	}
}

// BenchmarkValidateAndParse measures combined validation and parsing.
func BenchmarkValidateAndParse(b *testing.B) {
	content := createTOMLContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_ = p.Validate(content)
		_, _ = p.Parse("test.toml", content)
	}
}

// createTOMLContent creates a TOML document with the specified number of URLs.
func createTOMLContent(numURLs int) []byte {
	var sb strings.Builder
	sb.WriteString("[project]\n")
	sb.WriteString("name = \"test-project\"\n")
	sb.WriteString("version = \"1.0.0\"\n")
	sb.WriteString("homepage = \"https://github.com/example/project\"\n")
	sb.WriteString("repository = \"https://github.com/example/project.git\"\n\n")

	for i := range numURLs {
		sb.WriteString("[[links]]\n")
		sb.WriteString("title = \"Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\"\n")
		sb.WriteString("url = \"https://example.com/page/")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\"\n\n")
	}

	return []byte(sb.String())
}
