package yaml

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
)

func FuzzYAMLParserValidateAndParse(f *testing.F) {
	f.Add([]byte("url: https://example.com\n"))
	f.Add([]byte("items:\n  - prefix https://example.com/a\n  - https://example.com/b\n"))
	f.Add([]byte(":\n  - invalid"))

	p := New()

	f.Fuzz(func(t *testing.T, content []byte) {
		links, err := p.ValidateAndParse("fuzz.yaml", content)
		if err != nil {
			return
		}

		for _, link := range links {
			if !parser.IsHTTPURL(link.URL) {
				t.Fatalf("non-http URL returned: %#v", link)
			}
			if link.FilePath != "fuzz.yaml" {
				t.Fatalf("unexpected file path: %#v", link)
			}
		}
	})
}
