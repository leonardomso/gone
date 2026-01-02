package jsonparser

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Extensions(t *testing.T) {
	t.Parallel()

	p := New()
	exts := p.Extensions()

	assert.Contains(t, exts, ".json")
}

func TestParser_Validate(t *testing.T) {
	t.Parallel()

	p := New()

	t.Run("ValidJSON", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"key": "value"}`)
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("ValidJSONArray", func(t *testing.T) {
		t.Parallel()
		content := []byte(`["a", "b", "c"]`)
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		err := p.Validate([]byte{})
		assert.NoError(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"key": }`)
		err := p.Validate(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("TrailingComma", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"key": "value",}`)
		err := p.Validate(content)
		assert.Error(t, err)
	})
}

func TestParser_Parse(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("SimpleObject", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"url": "https://example.com"}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
		assert.Equal(t, "test.json", links[0].FilePath)
	})

	t.Run("MultipleURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"homepage": "https://example.com",
			"repo": "https://github.com/test/repo"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedObjects", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"project": {
				"links": {
					"homepage": "https://example.com",
					"docs": "https://docs.example.com"
				}
			}
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()
		content := []byte(`["https://one.com", "https://two.com"]`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("MixedArrayAndObjects", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"resources": [
				{"url": "https://api.example.com"},
				{"url": "https://cdn.example.com"}
			]
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("URLsAsKeys", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"https://example.com": "Example site",
			"https://github.com": "GitHub"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)

		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "https://example.com")
		assert.Contains(t, urls, "https://github.com")
	})

	t.Run("EmbeddedURLsInStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"description": "Check out https://example.com for more info"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("MultipleEmbeddedURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"message": "Visit https://example.com and https://github.com"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NoURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"name": "test", "value": 42}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		links, err := p.Parse("test.json", []byte{})
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("SkipsNonHTTPURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"http": "https://example.com",
			"ftp": "ftp://files.example.com",
			"mailto": "mailto:test@example.com",
			"file": "file:///path/to/file"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("CleansTrailingPunctuation", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"text": "Visit https://example.com. Or https://github.com, for code"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)

		// URLs should have trailing punctuation removed
		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "https://example.com")
		assert.Contains(t, urls, "https://github.com")
	})
}

func TestParser_ParseFromFile(t *testing.T) {
	t.Parallel()

	t.Run("SimpleFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/simple.json", false)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/nested.json", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 6)
	})

	t.Run("URLsAsKeysFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/urls_as_keys.json", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})

	t.Run("NoURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/no_urls.json", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("InvalidFileStrict", func(t *testing.T) {
		t.Parallel()
		_, err := parser.ExtractLinksWithRegistry("testdata/invalid.json", true)
		assert.Error(t, err)
	})

	t.Run("InvalidFileNonStrict", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/invalid.json", false)
		require.NoError(t, err)
		assert.Nil(t, links)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/empty.json", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("ArrayFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/array.json", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})

	t.Run("EmbeddedURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/embedded_urls.json", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})
}

func TestParser_LineNumbers(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("TracksLineNumbers", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
	"url1": "https://line2.example.com",
	"url2": "https://line3.example.com"
}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		require.Len(t, links, 2)

		// Line numbers should be reasonable
		for _, link := range links {
			assert.Greater(t, link.Line, 0)
		}
	})
}

func TestCleanURLTrailing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com.", "https://example.com"},
		{"https://example.com,", "https://example.com"},
		{"https://example.com)", "https://example.com"},
		{"https://example.com]", "https://example.com"},
		{"https://example.com}", "https://example.com"},
		{"https://example.com\"", "https://example.com"},
		{"https://example.com'", "https://example.com"},
		{"https://example.com.,;:", "https://example.com"},
		{"https://example.com/path", "https://example.com/path"},
		{"https://example.com/path?q=1", "https://example.com/path?q=1"},
	}

	for _, tt := range tests {
		result := parser.CleanURLTrailing(tt.input)
		assert.Equal(t, tt.expected, result, "CleanURLTrailing(%q)", tt.input)
	}
}

// TestParser_EdgeCases tests edge cases for the JSON parser.
func TestParser_EdgeCases(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("UnicodeURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"unicode": "https://例え.jp/path",
			"punycode": "https://xn--r8jz45g.jp/path",
			"emoji_path": "https://example.com/🎉"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("EscapedSlashesInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"url": "https:\/\/example.com\/path"}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		// The JSON decoder handles escape sequences, so this should parse correctly
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("DeeplyNestedObjects", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"level1": {
				"level2": {
					"level3": {
						"level4": {
							"level5": {
								"url": "https://deep.example.com"
							}
						}
					}
				}
			}
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://deep.example.com", links[0].URL)
	})

	t.Run("NullValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"url": "https://example.com",
			"nullable": null,
			"nested": {"value": null}
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("BooleanValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"url": "https://example.com",
			"enabled": true,
			"disabled": false
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("NumericValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"url": "https://example.com",
			"integer": 42,
			"float": 3.14,
			"negative": -100,
			"scientific": 1.23e10
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("EmptyArraysAndObjects", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"url": "https://example.com",
			"empty_array": [],
			"empty_object": {}
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("SpecialCharactersInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"query": "https://example.com/search?q=hello+world&lang=en",
			"fragment": "https://example.com/page#section",
			"encoded": "https://example.com/path%20with%20spaces"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("URLWithPortNumber", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"local": "https://localhost:8080/api",
			"custom": "https://example.com:3000/path"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("URLWithBasicAuth", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{"url": "https://user:pass@example.com/path"}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("LargeJSONArray", func(t *testing.T) {
		t.Parallel()
		// Generate a JSON array with 100 URLs
		content := []byte(`{"urls": [`)
		for i := range 100 {
			if i > 0 {
				content = append(content, ',')
			}
			content = append(content, []byte(`"https://example.com/page/`+string(rune('a'+i%26))+`"`)...)
		}
		content = append(content, []byte(`]}`)...)

		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 100)
	})

	t.Run("MixedValidAndInvalidURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{
			"valid1": "https://example.com",
			"noturl": "just some text",
			"valid2": "http://test.org",
			"partial": "example.com",
			"valid3": "https://final.io"
		}`)
		links, err := p.Parse("test.json", content)
		require.NoError(t, err)
		assert.Len(t, links, 3) // Only http/https URLs
	})

	t.Run("WhitespaceOnlyContent", func(t *testing.T) {
		t.Parallel()
		content := []byte("   \n\t  ")
		_, err := p.Parse("test.json", content)
		// Whitespace-only content is invalid JSON
		assert.Error(t, err)
	})
}
