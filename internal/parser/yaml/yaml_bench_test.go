package yaml

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkValidateAndParse measures YAML parsing performance.
func BenchmarkValidateAndParse(b *testing.B) {
	content := createYAMLContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ValidateAndParse("test.yaml", content)
	}
}

// createYAMLContent creates a YAML document with the specified number of URLs.
func createYAMLContent(numURLs int) []byte {
	var sb strings.Builder
	sb.WriteString("name: test-project\n")
	sb.WriteString("version: 1.0.0\n")
	sb.WriteString("links:\n")

	for i := range numURLs {
		sb.WriteString("  - title: Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\n")
		sb.WriteString("    url: https://example.com/page/")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\n")
	}

	sb.WriteString("homepage: https://github.com/example/project\n")
	sb.WriteString("repository: https://github.com/example/project.git\n")

	return []byte(sb.String())
}
