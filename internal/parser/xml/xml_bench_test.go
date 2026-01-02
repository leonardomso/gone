package xml //nolint:revive // package name matches file type being parsed

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkValidateAndParse measures XML parsing performance.
func BenchmarkValidateAndParse(b *testing.B) {
	content := createXMLContent(50)
	p := New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ValidateAndParse("test.xml", content)
	}
}

// createXMLContent creates an XML document with the specified number of URLs.
func createXMLContent(numURLs int) []byte {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<project>\n")
	sb.WriteString("  <name>test-project</name>\n")
	sb.WriteString("  <version>1.0.0</version>\n")
	sb.WriteString("  <homepage href=\"https://github.com/example/project\"/>\n")
	sb.WriteString("  <links>\n")

	for i := range numURLs {
		sb.WriteString("    <link>\n")
		sb.WriteString("      <title>Link ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("</title>\n")
		sb.WriteString("      <url href=\"https://example.com/page/")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\"/>\n")
		sb.WriteString("    </link>\n")
	}

	sb.WriteString("  </links>\n")
	sb.WriteString("</project>\n")

	return []byte(sb.String())
}
