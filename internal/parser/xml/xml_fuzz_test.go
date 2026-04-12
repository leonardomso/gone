package xml

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
)

func FuzzXMLParserValidateAndParse(f *testing.F) {
	f.Add([]byte(`<root><a href="https://example.com">docs</a></root>`))
	f.Add([]byte(`<root>prefix https://example.com/path</root>`))
	f.Add([]byte(`<root><a href="https://example.com">`))

	p := New()

	f.Fuzz(func(t *testing.T, content []byte) {
		links, err := p.ValidateAndParse("fuzz.xml", content)
		if err != nil {
			return
		}

		for _, link := range links {
			if !parser.IsHTTPURL(link.URL) {
				t.Fatalf("non-http URL returned: %#v", link)
			}
			if link.FilePath != "fuzz.xml" {
				t.Fatalf("unexpected file path: %#v", link)
			}
		}
	})
}
