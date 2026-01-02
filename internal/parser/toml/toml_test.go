package toml

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

	assert.Contains(t, exts, ".toml")
}

func TestParser_ValidateAndParse(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("SimpleKeyValue", func(t *testing.T) {
		t.Parallel()
		content := []byte(`url = "https://example.com"`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
		assert.Equal(t, "test.toml", links[0].FilePath)
	})

	t.Run("MultipleURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
homepage = "https://example.com"
repo = "https://github.com/test/repo"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedTables", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[project]
name = "test"

[project.links]
homepage = "https://example.com"
docs = "https://docs.example.com"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("Arrays", func(t *testing.T) {
		t.Parallel()
		content := []byte(`urls = ["https://one.com", "https://two.com"]`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("ArrayOfTables", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[[servers]]
url = "https://server1.com"

[[servers]]
url = "https://server2.com"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("InlineTables", func(t *testing.T) {
		t.Parallel()
		content := []byte(`link = { name = "Example", url = "https://example.com" }`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("EmbeddedURLsInStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`description = "Check out https://example.com for more info"`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("MultiLineStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`description = """
Visit https://example.com
and https://docs.example.com
"""`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("LiteralStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`path = 'https://literal.example.com'`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("NoURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`name = "test"
value = 42`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		links, err := p.ValidateAndParse("test.toml", []byte{})
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("SkipsNonHTTPURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
http = "https://example.com"
ftp = "ftp://files.example.com"
mailto = "mailto:test@example.com"
file = "file:///path/to/file"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("URLsAsKeys", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[bookmarks]
"https://example.com" = "Example site"
"https://github.com" = "GitHub"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})
}

func TestParser_ParseFromFile(t *testing.T) {
	t.Parallel()

	t.Run("SimpleFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/simple.toml", false)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/nested.toml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 6)
	})

	t.Run("ArraysFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/arrays.toml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 6)
	})

	t.Run("InlineTablesFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/inline_tables.toml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})

	t.Run("MultilineStringsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/multiline_strings.toml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})

	t.Run("NoURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/no_urls.toml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("InvalidFileStrict", func(t *testing.T) {
		t.Parallel()
		_, err := parser.ExtractLinksWithRegistry("testdata/invalid.toml", true)
		assert.Error(t, err)
	})

	t.Run("InvalidFileNonStrict", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/invalid.toml", false)
		require.NoError(t, err)
		assert.Nil(t, links)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/empty.toml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("URLsAsKeysFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/urls_as_keys.toml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})
}

func TestParser_LineNumbers(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("TracksLineNumbers", func(t *testing.T) {
		t.Parallel()
		content := []byte(`name = "test"
url1 = "https://line2.example.com"
url2 = "https://line3.example.com"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		require.Len(t, links, 2)

		// Line numbers should be reasonable
		for _, link := range links {
			assert.Greater(t, link.Line, 0)
		}
	})
}

// TestParser_EdgeCases tests edge cases for the TOML parser.
func TestParser_EdgeCases(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("DottedKeys", func(t *testing.T) {
		t.Parallel()
		content := []byte(`project.homepage = "https://example.com"`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("DeeplyNestedTables", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
[level1.level2.level3.level4]
url = "https://deep.example.com"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://deep.example.com", links[0].URL)
	})

	t.Run("MixedTypes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
string = "https://example.com"
integer = 42
float = 3.14
boolean = true
datetime = 2024-01-01T00:00:00Z
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("SpecialCharactersInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
query = "https://example.com/search?q=hello+world&lang=en"
fragment = "https://example.com/page#section"
encoded = "https://example.com/path%20with%20spaces"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("URLWithPortNumber", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
local = "https://localhost:8080/api"
custom = "https://example.com:3000/path"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("CommentsAreIgnored", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
# This is a comment with https://comment.example.com
url = "https://example.com" # inline comment with https://inline.example.com
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		// Only the actual value URL should be found
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("EscapedCharactersInStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`url = "https://example.com/path\"quoted\""`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("UnicodeInStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
japanese = "https://例え.jp"
emoji = "https://example.com/🎉"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("EmptyStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
empty = ""
url = "https://example.com"
`)
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("WhitespaceOnlyContent", func(t *testing.T) {
		t.Parallel()
		content := []byte("   \n  \n")
		links, err := p.ValidateAndParse("test.toml", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}
