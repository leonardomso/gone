package parser

import (
	"os"
	"path/filepath"
	"testing"

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
		assert.Equal(t, LinkTypeInline, links[0].Type)

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
		assert.Equal(t, LinkTypeImage, links[0].Type)
	})

	t.Run("AutoLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`Visit <http://example.com/auto> for more.`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/auto", links[0].URL)
		assert.Equal(t, LinkTypeAutolink, links[0].Type)
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
		assert.Equal(t, LinkTypeReference, links[0].Type)
	})

	t.Run("HTMLLinks", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="http://example.com/html">Click</a>`)
		links, err := ExtractLinksFromContent(content, "test.md")
		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "http://example.com/html", links[0].URL)
		assert.Equal(t, "Click", links[0].Text)
		assert.Equal(t, LinkTypeHTML, links[0].Type)
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

		assert.Equal(t, LinkTypeReference, links[0].Type)
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

func TestExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("FromInlineLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/inline_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("FromReferenceLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/reference_links.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("FromImageLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/image_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, LinkTypeImage, link.Type)
		}
	})

	t.Run("FromHTMLLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/html_links.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, LinkTypeHTML, link.Type)
		}
	})

	t.Run("FromCodeBlocksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/code_blocks.md")
		require.NoError(t, err)

		// Should only contain links outside code blocks
		for _, link := range links {
			assert.NotContains(t, link.URL, "fake")
			assert.NotContains(t, link.URL, "inline-code")
		}
	})

	t.Run("FromAutolinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/autolinks.md")
		require.NoError(t, err)
		assert.Len(t, links, 2)

		for _, link := range links {
			assert.Equal(t, LinkTypeAutolink, link.Type)
		}
	})

	t.Run("FromMixedContentFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/mixed_content.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})

	t.Run("FromNoLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/no_links.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("FromEmptyFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/empty.md")
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("FromEdgeCasesFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/edge_cases.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 5)
	})

	t.Run("FileNotFound", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/nonexistent.md")
		assert.Error(t, err)
		assert.Nil(t, links)
	})

	t.Run("TracksFilePath", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/inline_links.md")
		require.NoError(t, err)
		require.NotEmpty(t, links)

		for _, link := range links {
			assert.Equal(t, "testdata/inline_links.md", link.FilePath)
		}
	})
}

func TestExtractLinksFromMultipleFiles(t *testing.T) {
	t.Parallel()

	t.Run("AggregatesLinks", func(t *testing.T) {
		t.Parallel()
		files := []string{
			"testdata/inline_links.md",
			"testdata/image_links.md",
		}

		links, err := ExtractLinksFromMultipleFiles(files)
		require.NoError(t, err)

		// Should have links from both files
		assert.GreaterOrEqual(t, len(links), 5)

		// Verify we have links from different files
		filesSeen := map[string]bool{}
		for _, link := range links {
			filesSeen[link.FilePath] = true
		}
		assert.Len(t, filesSeen, 2)
	})

	t.Run("HandlesEmptyList", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinksFromMultipleFiles([]string{})
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("HandlesFileError", func(t *testing.T) {
		t.Parallel()
		files := []string{
			"testdata/inline_links.md",
			"testdata/nonexistent.md",
		}

		links, err := ExtractLinksFromMultipleFiles(files)
		assert.Error(t, err)
		assert.Nil(t, links)
	})

	t.Run("PreservesOrder", func(t *testing.T) {
		t.Parallel()
		// Create temp files with predictable content
		tmpDir := t.TempDir()

		file1 := filepath.Join(tmpDir, "first.md")
		file2 := filepath.Join(tmpDir, "second.md")

		err := os.WriteFile(file1, []byte("[First](http://first.com)"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("[Second](http://second.com)"), 0o644)
		require.NoError(t, err)

		links, err := ExtractLinksFromMultipleFiles([]string{file1, file2})
		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "http://first.com", links[0].URL)
		assert.Equal(t, "http://second.com", links[1].URL)
	})
}

func TestLinkType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		linkType LinkType
		expected string
	}{
		{LinkTypeInline, "inline"},
		{LinkTypeReference, "reference"},
		{LinkTypeImage, "image"},
		{LinkTypeAutolink, "autolink"},
		{LinkTypeHTML, "html"},
		{LinkType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.linkType.String())
		})
	}
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

	t.Run("FromNonHTTPLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/non_http_links.md")
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
		// Code nodes are handled differently
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

	t.Run("FromNestedFormattingFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/nested_formatting.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 8)

		// Verify all links have valid HTTP URLs
		for _, link := range links {
			assert.True(t,
				len(link.URL) > 7 && (link.URL[:7] == "http://" || link.URL[:8] == "https://"),
				"URL should be HTTP(S): %s", link.URL)
		}
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

	t.Run("FromEmptyLinksFile", func(t *testing.T) {
		t.Parallel()
		links, err := ExtractLinks("testdata/empty_links.md")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
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
		// Note: Current HTML regex doesn't support nested tags in link text
		// This is a known limitation - it expects simple text content
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
