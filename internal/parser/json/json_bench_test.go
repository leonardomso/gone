package jsonparser

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkValidateAndParse measures JSON parsing performance.
func BenchmarkValidateAndParse(b *testing.B) {
	content := createJSONContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ValidateAndParse("test.json", content)
	}
}

// createJSONContent creates a JSON document with the specified number of URLs.
func createJSONContent(numURLs int) []byte {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString("  \"name\": \"test-project\",\n")
	sb.WriteString("  \"version\": \"1.0.0\",\n")
	sb.WriteString("  \"links\": [\n")

	for i := range numURLs {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString("    {\n")
		sb.WriteString("      \"title\": \"Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\",\n")
		sb.WriteString("      \"url\": \"https://example.com/page/")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\"\n")
		sb.WriteString("    }")
	}

	sb.WriteString("\n  ],\n")
	sb.WriteString("  \"homepage\": \"https://github.com/example/project\",\n")
	sb.WriteString("  \"repository\": \"https://github.com/example/project.git\"\n")
	sb.WriteString("}\n")

	return []byte(sb.String())
}
