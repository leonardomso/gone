package yaml

import (
	"testing"

	"github.com/leonardomso/gone/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLParser_Extensions(t *testing.T) {
	t.Parallel()

	p := New()
	exts := p.Extensions()

	assert.Contains(t, exts, ".yaml")
	assert.Contains(t, exts, ".yml")
}

func TestYAMLParser_Validate(t *testing.T) {
	t.Parallel()

	p := New()

	t.Run("ValidYAML", func(t *testing.T) {
		t.Parallel()
		content := []byte("key: value\n")
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("ValidYAMLList", func(t *testing.T) {
		t.Parallel()
		content := []byte("- item1\n- item2\n")
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		err := p.Validate([]byte{})
		assert.NoError(t, err)
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		t.Parallel()
		content := []byte("key:\n  - item\n invalid indentation")
		err := p.Validate(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid YAML")
	})

	t.Run("MultiDocument", func(t *testing.T) {
		t.Parallel()
		content := []byte("---\nkey1: value1\n---\nkey2: value2\n")
		err := p.Validate(content)
		assert.NoError(t, err)
	})
}

func TestYAMLParser_Parse(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("SimpleMapping", func(t *testing.T) {
		t.Parallel()
		content := []byte("url: https://example.com\n")
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
		assert.Equal(t, "test.yaml", links[0].FilePath)
	})

	t.Run("MultipleURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
homepage: https://example.com
repo: https://github.com/test/repo
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedMappings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
project:
  links:
    homepage: https://example.com
    docs: https://docs.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("Sequence", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
urls:
  - https://one.com
  - https://two.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("MixedSequenceAndMappings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
resources:
  - name: API
    url: https://api.example.com
  - name: CDN
    url: https://cdn.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("URLsAsKeys", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
https://example.com: Example site
https://github.com: GitHub
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)

		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "https://example.com")
		assert.Contains(t, urls, "https://github.com")
	})

	t.Run("MultiDocument", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
url: https://doc1.example.com
---
url: https://doc2.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("EmbeddedURLsInStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
description: Check out https://example.com for more info
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("NoURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte("name: test\nvalue: 42\n")
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		links, err := p.Parse("test.yaml", []byte{})
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("SkipsNonHTTPURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
http: https://example.com
ftp: ftp://files.example.com
mailto: mailto:test@example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("Anchors", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
defaults: &defaults
  url: https://example.com
  timeout: 30

production:
  <<: *defaults
  url: https://prod.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		// Should find both the default URL and the production URL
		assert.GreaterOrEqual(t, len(links), 2)
	})
}

func TestYAMLParser_ParseFromFile(t *testing.T) {
	t.Parallel()

	t.Run("SimpleFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/simple.yaml", false)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/nested.yaml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 6)
	})

	t.Run("MultiDocumentFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/multi_document.yaml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})

	t.Run("URLsAsKeysFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/urls_as_keys.yaml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})

	t.Run("NoURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/no_urls.yaml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("InvalidFileStrict", func(t *testing.T) {
		t.Parallel()
		_, err := parser.ExtractLinksWithRegistry("testdata/invalid.yaml", true)
		assert.Error(t, err)
	})

	t.Run("InvalidFileNonStrict", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/invalid.yaml", false)
		require.NoError(t, err)
		assert.Nil(t, links)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/empty.yaml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("AnchorsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/anchors.yaml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})
}

func TestYAMLParser_LineNumbers(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("TracksLineNumbers", func(t *testing.T) {
		t.Parallel()
		content := []byte(`name: test
url1: https://line2.example.com
url2: https://line3.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 2)

		// Line numbers should be accurate
		assert.Equal(t, 2, links[0].Line)
		assert.Equal(t, 3, links[1].Line)
	})

	t.Run("NestedLineNumbers", func(t *testing.T) {
		t.Parallel()
		content := []byte(`root:
  child:
    url: https://line3.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 3, links[0].Line)
	})
}

func TestYAMLParser_PathTracking(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("TracksPath", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
project:
  links:
    homepage: https://example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		// The path should be in the Text field
		assert.Contains(t, links[0].Text, "homepage")
	})

	t.Run("ArrayPath", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
urls:
  - https://example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		// The path should include array index
		assert.Contains(t, links[0].Text, "[0]")
	})
}

// TestYAMLParser_EdgeCases tests edge cases for the YAML parser.
func TestYAMLParser_EdgeCases(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("FlowStyleMapping", func(t *testing.T) {
		t.Parallel()
		content := []byte(`{url: https://example.com, name: test}`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("FlowStyleSequence", func(t *testing.T) {
		t.Parallel()
		content := []byte(`urls: [https://one.com, https://two.com]`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("BlockLiteralScalar", func(t *testing.T) {
		t.Parallel()
		content := []byte(`description: |
  This is a multi-line description.
  Visit https://example.com for more info.
  Also check https://docs.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("BlockFoldedScalar", func(t *testing.T) {
		t.Parallel()
		content := []byte(`description: >
  This is a folded description
  with https://example.com embedded
  in the text content.
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("QuotedStrings", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
single_quoted: 'https://example.com'
double_quoted: "https://double.example.com"
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("QuotedStringsWithEscapes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
escaped: "https://example.com/path\"quoted\""
newline: "https://example.com\nmore text"
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("UnicodeKeys", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
日本語: https://example.jp
中文: https://example.cn
emoji🎉: https://example.com/party
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("NullValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
url: https://example.com
nullable: null
tilde_null: ~
empty_value:
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("BooleanValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
url: https://example.com
enabled: true
disabled: false
yes_val: yes
no_val: no
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("NumericValues", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
url: https://example.com
integer: 42
float: 3.14
hex: 0x1A
octal: 0o755
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("SpecialCharactersInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
query: https://example.com/search?q=hello+world&lang=en
fragment: https://example.com/page#section
encoded: https://example.com/path%20with%20spaces
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("URLWithPortNumber", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
local: https://localhost:8080/api
custom: https://example.com:3000/path
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("DeeplyNested", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
level1:
  level2:
    level3:
      level4:
        level5:
          url: https://deep.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://deep.example.com", links[0].URL)
	})

	t.Run("MixedContent", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
string: just text
url: https://example.com
number: 42
bool: true
null_val: null
list:
  - item1
  - https://list.example.com
map:
  key: value
  link: https://map.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("CommentsAreIgnored", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
# This is a comment with https://comment.example.com
url: https://example.com  # inline comment with https://inline.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		// Only the actual value URL should be found, not URLs in comments
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("EmptyDocuments", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
---
url: https://example.com
---
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("WhitespaceOnlyContent", func(t *testing.T) {
		t.Parallel()
		// Tab characters in YAML content can cause parsing issues
		// Using only spaces and newlines
		content := []byte("   \n  \n")
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("ExplicitTypeTags", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
explicit_string: !!str https://example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("AnchorWithOverride", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
base: &base
  url: https://base.example.com
  timeout: 30

derived:
  <<: *base
  url: https://derived.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		// Should find both URLs
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("ComplexKeyMapping", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
? complex key
: https://example.com
simple: https://simple.example.com
`)
		links, err := p.Parse("test.yaml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 1)
	})
}
