package markdown

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
)

func FuzzExtractLinksFromContent(f *testing.F) {
	f.Add([]byte("[docs](https://example.com/docs)\n"))
	f.Add([]byte("Inline URL: https://example.com/path\n"))
	f.Add([]byte("```go\nhttps://ignored.example\n```\n<a href=\"https://html.example\">html</a>\n"))

	f.Fuzz(func(t *testing.T, content []byte) {
		links, err := ExtractLinksFromContent(content, "fuzz.md")
		if err != nil {
			t.Fatalf("markdown parsing should not error: %v", err)
		}

		for _, link := range links {
			if !parser.IsHTTPURL(link.URL) {
				t.Fatalf("non-http URL returned: %#v", link)
			}
			if link.FilePath != "fuzz.md" {
				t.Fatalf("unexpected file path: %#v", link)
			}
		}
	})
}
