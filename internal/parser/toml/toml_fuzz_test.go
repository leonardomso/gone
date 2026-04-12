package toml

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
)

func FuzzTOMLParserValidateAndParse(f *testing.F) {
	f.Add([]byte("url = 'https://example.com'\n"))
	f.Add([]byte("[links]\nprimary = 'https://example.com/a'\nsecondary = 'prefix https://example.com/b'\n"))
	f.Add([]byte("url = \n"))

	p := New()

	f.Fuzz(func(t *testing.T, content []byte) {
		links, err := p.ValidateAndParse("fuzz.toml", content)
		if err != nil {
			return
		}

		for _, link := range links {
			if !parser.IsHTTPURL(link.URL) {
				t.Fatalf("non-http URL returned: %#v", link)
			}
			if link.FilePath != "fuzz.toml" {
				t.Fatalf("unexpected file path: %#v", link)
			}
		}
	})
}
