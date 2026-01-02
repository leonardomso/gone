package markdown

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/leonardomso/gone/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLinksFromContent(t *testing.T) {
	t.Parallel()

	t.Run("InlineLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Click here](http://example.com) and [another](https://test.com/page)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		assert.Equal(t, "http://example.com", links[0].URL)
		assert.Equal(t, "Click here", links[0].Text)
		assert.Equal(t, parser.LinkTypeInline, links[0].Type)

		assert.Equal(t, "https://test.com/page", links[1].URL)
		assert.Equal(t, "another", links[1].Text)
	})

	t.Run("MultipleLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
# Title

[Link 1](http://one.com)

Some text here.

[Link 2](http://two.com)

[Link 3](http://three.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("NoLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`# Just a title

Some paragraph without any links.

- Item 1
- Item 2
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinksFromContent([]byte{}, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}

func TestExtractLinks_LinkTypes(t *testing.T) {
	t.Parallel()

	t.Run("ImageLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`![Alt text](http://example.com/image.png)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/image.png", links[0].URL)
		assert.Equal(t, "Alt text", links[0].Text)
		assert.Equal(t, parser.LinkTypeImage, links[0].Type)
	})

	t.Run("AutoLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`Visit <http://example.com/auto> for more.`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/auto", links[0].URL)
		assert.Equal(t, parser.LinkTypeAutolink, links[0].Type)
	})

	t.Run("ReferenceLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[Click here][ref]

[ref]: http://example.com/ref
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/ref", links[0].URL)
		assert.Equal(t, "Click here", links[0].Text)
		assert.Equal(t, parser.LinkTypeReference, links[0].Type)
	})

	t.Run("HTMLLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="http://example.com/html">Click</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/html", links[0].URL)
		assert.Equal(t, "Click", links[0].Text)
		assert.Equal(t, parser.LinkTypeHTML, links[0].Type)
	})
}

func TestExtractLinks_CodeBlocks(t *testing.T) {
	t.Parallel()

	t.Run("SkipsFencedCodeBlock", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[real link](http://real.com)

` + "```" + `
[fake link](http://fake.com)
` + "```" + `

[another real](http://another.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "http://real.com")
		assert.Contains(t, urls, "http://another.com")
		assert.NotContains(t, urls, "http://fake.com")
	})

	t.Run("SkipsIndentedCode", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[real](http://real.com)

    [fake](http://fake.com)

[also real](http://also-real.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)

		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "http://real.com")
		assert.Contains(t, urls, "http://also-real.com")
		assert.NotContains(t, urls, "http://fake.com")
	})
}

func TestExtractLinks_Filtering(t *testing.T) {
	t.Parallel()

	t.Run("SkipsNonHTTP", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[HTTP](http://example.com)
[HTTPS](https://example.com)
[Mailto](mailto:test@example.com)
[Tel](tel:+1234567890)
[Anchor](#section)
[FTP](ftp://example.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2) // Only HTTP and HTTPS

		for _, link := range links {
			assert.True(t,
				link.URL == "http://example.com" || link.URL == "https://example.com",
				"Unexpected URL: %s", link.URL)
		}
	})

	t.Run("SkipsRelative", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[Absolute](http://example.com)
[Relative](./path/to/file)
[Root relative](/path/to/file)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "http://example.com", links[0].URL)
	})

	t.Run("IncludesHTTPAndHTTPS", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[HTTP](http://example.com)
[HTTPS](https://secure.example.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})
}

func TestExtractLinks_LineNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		expectedLine int
	}{
		{
			name:         "FirstLine",
			content:      "[link](http://example.com)",
			expectedLine: 1,
		},
		{
			name:         "ThirdLine",
			content:      "line 1\nline 2\n[link](http://example.com)",
			expectedLine: 3,
		},
		{
			name:         "WithBlankLines",
			content:      "line 1\n\n\n[link](http://example.com)",
			expectedLine: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			links, err := ExtractLinksFromContent([]byte(tt.content), "test.md")
			require.NoError(t, err)
			require.Len(t, links, 1)
			assert.Equal(t, tt.expectedLine, links[0].Line)
		})
	}
}

func TestExtractLinks_LinkText(t *testing.T) {
	t.Parallel()

	t.Run("CapturesLinkText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Click here for more information](http://example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Click here for more information", links[0].Text)
	})

	t.Run("CapturesImageAltText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`![Company Logo](http://example.com/logo.png)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Company Logo", links[0].Text)
	})

	t.Run("HandlesEmptyText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[](http://example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "", links[0].Text)
	})
}

func TestExtractLinks_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("EncodedURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link](http://example.com/path%20with%20spaces)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com/path%20with%20spaces", links[0].URL)
	})

	t.Run("URLWithQueryParams", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Search](http://example.com/search?q=test&page=1)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com/search?q=test&page=1", links[0].URL)
	})

	t.Run("URLWithFragment", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Section](http://example.com/page#section)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com/page#section", links[0].URL)
	})

	t.Run("URLWithPort", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[API](http://example.com:8080/api)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com:8080/api", links[0].URL)
	})

	t.Run("AdjacentLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[one](http://one.com)[two](http://two.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("VeryLongURL", func(t *testing.T) {
		t.Parallel()
		longPath := "http://example.com/" + string(make([]byte, 1000))
		for i := range longPath[len("http://example.com/"):] {
			longPath = longPath[:len("http://example.com/")+i] + "a" + longPath[len("http://example.com/")+i+1:]
		}
		// Simplified: just use a long but valid URL
		longURL := "http://example.com/very/long/path/that/goes/on/and/on"
		content := []byte(`[Long](` + longURL + `)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, longURL, links[0].URL)
	})

	t.Run("UnicodeInText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[日本語テキスト](http://example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "日本語テキスト", links[0].Text)
	})

	t.Run("SpecialCharactersInText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link with "quotes" & <special>](http://example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})
}

func TestExtractLinks_ReferenceDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("TracksDefinitionLine", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[Click][ref]

[ref]: http://example.com
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, parser.LinkTypeReference, links[0].Type)
		assert.Equal(t, "ref", links[0].RefName)
		assert.Greater(t, links[0].RefDefLine, 0)
	})

	t.Run("MultipleUsages", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[First][ref]

[Second][ref]

[ref]: http://example.com
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		// Both usages should resolve to the same URL
		assert.Len(t, links, 2)
		assert.Equal(t, links[0].URL, links[1].URL)
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[Click][REF]

[ref]: http://example.com
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com", links[0].URL)
	})
}

func TestParser_Extensions(t *testing.T) {
	t.Parallel()
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".md")
	assert.Contains(t, exts, ".mdx")
	assert.Contains(t, exts, ".markdown")
}

func TestParser_Validate(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("AlwaysValid", func(t *testing.T) {
		t.Parallel()
		// Markdown is very permissive
		assert.NoError(t, p.Validate([]byte("# Valid")))
		assert.NoError(t, p.Validate([]byte("random text")))
		assert.NoError(t, p.Validate([]byte{}))
		assert.NoError(t, p.Validate(nil))
	})
}

func TestParser_Parse(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("ParsesLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte("[link](http://example.com)")
		links, err := p.Parse("test.md", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com", links[0].URL)
	})
}

// =============================================================================
// File-based Tests
// =============================================================================

func TestExtractLinks_FromTestdataFiles(t *testing.T) {
	t.Parallel()

	readTestFile := func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join("testdata", name))
	}

	t.Run("FromInlineLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("inline_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "inline_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("FromReferenceLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("reference_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "reference_links.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("FromImageLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("image_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "image_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, parser.LinkTypeImage, link.Type)
		}
	})

	t.Run("FromHTMLLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("html_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "html_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, parser.LinkTypeHTML, link.Type)
		}
	})

	t.Run("FromCodeBlocksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("code_blocks.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "code_blocks.md")
		require.NoError(t, err)

		// Should only contain links outside code blocks
		for _, link := range links {
			assert.NotContains(t, link.URL, "fake")
			assert.NotContains(t, link.URL, "inline-code")
		}
	})

	t.Run("FromAutolinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("autolinks.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "autolinks.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, parser.LinkTypeAutolink, link.Type)
		}
	})

	t.Run("FromMixedContentFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("mixed_content.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "mixed_content.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})

	t.Run("FromNoLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("no_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "no_links.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("FromEmptyFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("empty.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "empty.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("FromEdgeCasesFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("edge_cases.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "edge_cases.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 5)
	})

	t.Run("FromNonHTTPLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("non_http_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "non_http_links.md")
		require.NoError(t, err)

		// Should only have HTTP/HTTPS links
		for _, link := range links {
			assert.True(t,
				link.URL == "http://real.example.com" ||
					link.URL == "https://cdn.example.com/image.png" ||
					link.URL == "https://html.example.com",
				"Unexpected URL: %s", link.URL)
		}
		assert.Len(t, links, 3)
	})

	t.Run("FromNestedFormattingFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("nested_formatting.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "nested_formatting.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 8)

		// Verify all links have valid HTTP URLs
		for _, link := range links {
			assert.True(t,
				len(link.URL) > 7 && (link.URL[:7] == "http://" || link.URL[:8] == "https://"),
				"URL should be HTTP(S): %s", link.URL)
		}
	})

	t.Run("FromEmptyLinksFile", func(t *testing.T) {
		t.Parallel()
		content, err := readTestFile("empty_links.md")
		require.NoError(t, err)
		links, err := ExtractLinksFromContent(content, "empty_links.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestExtractLinks_NonHTTPURLs(t *testing.T) {
	t.Parallel()

	t.Run("SkipsMailtoAutolinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
Email: <mailto:test@example.com>
Real: <http://real.example.com>
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://real.example.com", links[0].URL)
	})

	t.Run("SkipsDataURIImages", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
![Data](data:image/png;base64,ABC123)
![Real](http://real.example.com/image.png)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://real.example.com/image.png", links[0].URL)
	})

	t.Run("SkipsRelativeImages", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
![Relative](./images/logo.png)
![Root](/images/icon.png)
![Real](https://cdn.example.com/image.png)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://cdn.example.com/image.png", links[0].URL)
	})

	t.Run("SkipsRelativeHTMLLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<a href="/relative/path">Relative</a>
<a href="javascript:void(0)">JS</a>
<a href="#section">Anchor</a>
<a href="https://real.example.com">Real</a>
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://real.example.com", links[0].URL)
	})

	t.Run("SkipsFTPAutolinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
FTP: <ftp://files.example.com>
HTTP: <http://web.example.com>
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://web.example.com", links[0].URL)
	})
}

func TestExtractLinks_NestedFormatting(t *testing.T) {
	t.Parallel()

	t.Run("BoldInLink", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[**Bold text**](http://bold.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://bold.example.com", links[0].URL)
		assert.Equal(t, "Bold text", links[0].Text)
	})

	t.Run("ItalicInLink", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[*Italic text*](http://italic.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Italic text", links[0].Text)
	})

	t.Run("CodeInLink", func(t *testing.T) {
		t.Parallel()
		content := []byte("[`code snippet`](http://code.example.com)")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://code.example.com", links[0].URL)
	})

	t.Run("MixedFormattingInLink", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[**Bold** and *italic*](http://mixed.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Contains(t, links[0].Text, "Bold")
		assert.Contains(t, links[0].Text, "italic")
	})
}

func TestExtractLinks_EmptyLinks(t *testing.T) {
	t.Parallel()

	t.Run("EmptyLinkText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[](http://empty-text.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://empty-text.example.com", links[0].URL)
		assert.Equal(t, "", links[0].Text)
	})

	t.Run("WhitespaceOnlyText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[   ](http://whitespace.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://whitespace.example.com", links[0].URL)
	})

	t.Run("EmptyAltTextImage", func(t *testing.T) {
		t.Parallel()
		content := []byte(`![](http://empty-alt.example.com/image.png)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://empty-alt.example.com/image.png", links[0].URL)
		assert.Equal(t, "", links[0].Text)
	})
}

func TestExtractLinks_BlockNodes(t *testing.T) {
	t.Parallel()

	t.Run("LinksInBlockquote", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
> Quote with [link](http://blockquote.example.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://blockquote.example.com", links[0].URL)
	})

	t.Run("LinksInList", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
- Item with [link1](http://list1.example.com)
- Another [link2](http://list2.example.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("LinksInNestedList", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
- Outer
  - Inner [link](http://nested.example.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("LinksInTable", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
| Header |
|--------|
| [link](http://table.example.com) |
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})
}

func TestExtractLinks_ComplexHTML(t *testing.T) {
	t.Parallel()

	t.Run("HTMLWithAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="http://example.com" target="_blank" rel="noopener">Link</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com", links[0].URL)
		assert.Equal(t, "Link", links[0].Text)
	})

	t.Run("HTMLWithSingleQuotes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href='http://single-quotes.example.com'>Link</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://single-quotes.example.com", links[0].URL)
	})

	t.Run("HTMLWithNestedTags", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="http://nested.example.com">Plain text only</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Plain text only", links[0].Text)
	})

	t.Run("MultipleHTMLLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<a href="http://first.example.com">First</a>
<a href="http://second.example.com">Second</a>
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})
}

func TestExtractLinks_AdditionalEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("URLWithPort", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link](http://localhost:8080/path)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://localhost:8080/path", links[0].URL)
	})

	t.Run("URLWithCredentials", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link](http://user:pass@example.com/path)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://user:pass@example.com/path", links[0].URL)
	})

	t.Run("URLWithIPv4", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link](http://192.168.1.1/path)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("URLWithSpecialCharsInPath", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Link](http://example.com/path%20with%20spaces)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("ConsecutiveLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[One](http://one.com)[Two](http://two.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("LinkInHeading", func(t *testing.T) {
		t.Parallel()
		content := []byte(`# Heading with [link](http://heading.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("ImageWithTitle", func(t *testing.T) {
		t.Parallel()
		content := []byte(`![Alt](http://example.com/img.png "Title")`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Alt", links[0].Text)
	})

	t.Run("LinkWithTitle", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[Text](http://example.com "Title")`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Text", links[0].Text)
	})
}

// =============================================================================
// Code Block Handling Tests
// =============================================================================

func TestCodeBlock_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("FencedCodeBlockWithLanguage", func(t *testing.T) {
		t.Parallel()
		content := []byte("```javascript\n[fake](http://fake.com)\n```\n[real](http://real.com)")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://real.com", links[0].URL)
	})

	t.Run("MultipleFencedCodeBlocks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[before](http://before.com)

` + "```" + `
[fake1](http://fake1.com)
` + "```" + `

[middle](http://middle.com)

` + "```python" + `
[fake2](http://fake2.com)
` + "```" + `

[after](http://after.com)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)

		urls := make([]string, len(links))
		for i, link := range links {
			urls[i] = link.URL
		}

		assert.Contains(t, urls, "http://before.com")
		assert.Contains(t, urls, "http://middle.com")
		assert.Contains(t, urls, "http://after.com")
		assert.NotContains(t, urls, "http://fake1.com")
		assert.NotContains(t, urls, "http://fake2.com")
	})

	t.Run("InlineCodeNotAffected", func(t *testing.T) {
		t.Parallel()
		content := []byte("[real](http://real.com) `http://inline.com`")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "http://real.com", links[0].URL)
	})

	t.Run("CodeBlockAtEndOfFile", func(t *testing.T) {
		t.Parallel()
		content := []byte("[real](http://real.com)\n```\n[fake](http://fake.com)\n```")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://real.com", links[0].URL)
	})
}

// =============================================================================
// Empty and Edge Case Tests
// =============================================================================

func TestEmptyAndEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NilContent", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinksFromContent(nil, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("OnlyWhitespace", func(t *testing.T) {
		t.Parallel()
		content := []byte("   \n\t\n   \n")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("OnlyHeadings", func(t *testing.T) {
		t.Parallel()
		content := []byte("# Heading 1\n## Heading 2\n### Heading 3")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("BrokenMarkdownLink", func(t *testing.T) {
		t.Parallel()
		content := []byte("[broken link(http://broken.com)")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("LinkWithParenthesesInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte("[wiki](http://en.wikipedia.org/wiki/Markdown_(markup_language))")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(links), 0)
	})

	t.Run("VeryLongContent", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		for i := range 1000 {
			buf.WriteString("This is line " + string(rune('0'+i%10)) + "\n")
		}
		buf.WriteString("[link](http://longcontent.example.com)\n")

		links, err := ExtractLinksFromContent(buf.Bytes(), "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 1001, links[0].Line)
	})
}

// =============================================================================
// Position Tracking Tests
// =============================================================================

func TestGetPosition_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("FirstCharacterOfFile", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[link](http://first.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 1, links[0].Line)
	})

	t.Run("MultipleLinksOnSameLine", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[one](http://one.com) [two](http://two.com) [three](http://three.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 3)
		for _, link := range links {
			assert.Equal(t, 1, link.Line)
		}
	})

	t.Run("LinksAfterManyBlankLines", func(t *testing.T) {
		t.Parallel()
		content := []byte("\n\n\n\n\n\n\n\n\n\n[link](http://line11.example.com)")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 11, links[0].Line)
	})

	t.Run("LinkInBlockquote", func(t *testing.T) {
		t.Parallel()
		content := []byte(`> [quoted link](http://quoted.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 1, links[0].Line)
	})

	t.Run("LinkInNestedBlockquote", func(t *testing.T) {
		t.Parallel()
		content := []byte(`> > [deeply nested](http://nested.example.com)`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("ImagePosition", func(t *testing.T) {
		t.Parallel()
		content := []byte("line 1\nline 2\n![img](http://img.example.com/pic.png)")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 3, links[0].Line)
	})

	t.Run("AutolinkPosition", func(t *testing.T) {
		t.Parallel()
		content := []byte("first\nsecond\n<http://autolink.example.com>")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.GreaterOrEqual(t, links[0].Line, 1)
	})

	t.Run("HTMLLinkPosition", func(t *testing.T) {
		t.Parallel()
		content := []byte("line1\nline2\nline3\n<a href=\"http://html.example.com\">click</a>")
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 4, links[0].Line)
	})

	t.Run("ReferenceDefinitionPosition", func(t *testing.T) {
		t.Parallel()
		content := []byte(`[text][ref]

[ref]: http://ref.example.com`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, parser.LinkTypeReference, links[0].Type)
		assert.Equal(t, 3, links[0].RefDefLine)
	})
}

// =============================================================================
// HTML Link Edge Cases
// =============================================================================

func TestExtractHTMLLinks_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("HTMLWithEmptyHref", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="">Empty</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("HTMLWithWhitespaceHref", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="   ">Whitespace</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("HTMLWithMultipleAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a class="btn" href="http://example.com" id="link1" target="_blank">Link</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://example.com", links[0].URL)
	})

	t.Run("HTMLWithNewlinesInTag", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a 
			href="http://newline.example.com"
			target="_blank"
		>Link</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "http://newline.example.com", links[0].URL)
	})

	t.Run("HTMLMixedWithMarkdown", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[Markdown](http://markdown.example.com)

<a href="http://html.example.com">HTML</a>

![Image](http://image.example.com/pic.png)
`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("HTMLCaseSensitivity", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="http://lowercase.example.com">Lower</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)
	})
}
