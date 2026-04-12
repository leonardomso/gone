package jsonparser

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
)

func FuzzJSONParserValidateAndParse(f *testing.F) {
	f.Add([]byte(`{"url":"https://example.com"}`))
	f.Add([]byte(`{"nested":{"items":["https://example.com/a","prefix https://example.com/b"]}}`))
	f.Add([]byte(`{"broken":`))

	p := New()

	f.Fuzz(func(t *testing.T, content []byte) {
		links, err := p.ValidateAndParse("fuzz.json", content)
		if err != nil {
			return
		}

		for _, link := range links {
			if !parser.IsHTTPURL(link.URL) {
				t.Fatalf("non-http URL returned: %#v", link)
			}
			if link.FilePath != "fuzz.json" {
				t.Fatalf("unexpected file path: %#v", link)
			}
		}
	})
}
