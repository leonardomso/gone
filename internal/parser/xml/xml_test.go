package xml //nolint:revive // package name matches file type being parsed

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

	assert.Contains(t, exts, ".xml")
}

func TestParser_Validate(t *testing.T) {
	t.Parallel()

	p := New()

	t.Run("ValidXML", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<?xml version="1.0"?><root><item/></root>`)
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("ValidXMLWithAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root attr="value"><item id="1"/></root>`)
		err := p.Validate(content)
		assert.NoError(t, err)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		err := p.Validate([]byte{})
		assert.NoError(t, err)
	})

	t.Run("InvalidXML", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root><unclosed>`)
		err := p.Validate(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid XML")
	})
}

func TestParser_Parse(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("SimpleHref", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="https://example.com">Link</a>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
		assert.Equal(t, "test.xml", links[0].FilePath)
	})

	t.Run("MultipleLinkAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a href="https://one.example.com">One</a>
	<img src="https://two.example.com/image.png"/>
	<link href="https://three.example.com/style.css"/>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 3)
	})

	t.Run("URLInTextContent", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<description>Visit https://example.com for info</description>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("MultipleURLsInText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<text>Check https://one.com and https://two.com</text>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedElements", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<level1>
		<level2>
			<a href="https://deep.example.com">Deep</a>
		</level2>
	</level1>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("SelfClosingElements", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<img src="https://example.com/image.png"/>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("VariousURLAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a href="https://href.example.com">Href</a>
	<img src="https://src.example.com/img.png"/>
	<form action="https://action.example.com/submit"/>
	<video poster="https://poster.example.com/thumb.jpg"/>
	<blockquote cite="https://cite.example.com/source"/>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 5)
	})

	t.Run("NoURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root><item id="1">Text</item></root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		links, err := p.Parse("test.xml", []byte{})
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("SkipsNonHTTPURLs", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a href="https://example.com">HTTP</a>
	<a href="ftp://files.example.com">FTP</a>
	<a href="mailto:test@example.com">Email</a>
	<a href="#anchor">Anchor</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("MixedAttributesAndText", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a href="https://attr.example.com">Visit https://text.example.com</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})
}

func TestParser_ParseFromFile(t *testing.T) {
	t.Parallel()

	t.Run("SimpleFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/simple.xml", false)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("NestedFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/nested.xml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 3)
	})

	t.Run("TextURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/text_urls.xml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 4)
	})

	t.Run("AttributesFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/attributes.xml", false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 8)
	})

	t.Run("NoURLsFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/no_urls.xml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("InvalidFileStrict", func(t *testing.T) {
		t.Parallel()
		_, err := parser.ExtractLinksWithRegistry("testdata/invalid.xml", true)
		assert.Error(t, err)
	})

	t.Run("InvalidFileNonStrict", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/invalid.xml", false)
		require.NoError(t, err)
		assert.Nil(t, links)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()
		links, err := parser.ExtractLinksWithRegistry("testdata/empty.xml", false)
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}

func TestParser_LineNumbers(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("TracksLineNumbers", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root>
<a href="https://line2.example.com">Line 2</a>
<a href="https://line3.example.com">Line 3</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		require.Len(t, links, 2)

		// Line numbers should be reasonable
		for _, link := range links {
			assert.Greater(t, link.Line, 0)
		}
	})
}

// TestParser_EdgeCases tests edge cases for the XML parser.
func TestParser_EdgeCases(t *testing.T) {
	t.Parallel()
	p := New()

	t.Run("XMLDeclaration", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root href="https://example.com"/>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("CDATA", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root><![CDATA[Visit https://cdata.example.com]]></root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("Comments", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root>
<!-- Comment with https://comment.example.com -->
<a href="https://real.example.com">Real</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		// Should only find the real link, not the one in comment
		assert.Len(t, links, 1)
		assert.Equal(t, "https://real.example.com", links[0].URL)
	})

	t.Run("Namespaces", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<root xmlns:xlink="http://www.w3.org/1999/xlink">
<a xlink:href="https://xlink.example.com">XLink</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("WhitespaceInAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="  https://example.com  ">Link</a>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "https://example.com", links[0].URL)
	})

	t.Run("SpecialCharactersInURL", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a href="https://example.com/search?q=hello&amp;lang=en">Query</a>
	<a href="https://example.com/path#section">Fragment</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(links), 2)
	})

	t.Run("URLWithPortNumber", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="https://localhost:8080/api">API</a>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("MixedCaseAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<root>
	<a HREF="https://uppercase.example.com">Upper</a>
	<a Href="https://mixed.example.com">Mixed</a>
</root>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("MultipleAttributesOnElement", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<video src="https://video.example.com/v.mp4" poster="https://poster.example.com/p.jpg"/>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 2)
	})

	t.Run("EmptyAttributes", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a href="">Empty</a><a href="https://example.com">Real</a>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("VeryDeeplyNested", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<a><b><c><d><e><f><g><h><i>` +
			`<j href="https://deep.example.com"/></i></h></g></f></e></d></c></b></a>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 1)
	})

	t.Run("ProcessingInstruction", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<?xml version="1.0"?>` +
			`<?my-pi href="https://pi.example.com"?><root href="https://example.com"/>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		// Should find the one in root, PI attributes are not standard
		assert.GreaterOrEqual(t, len(links), 1)
	})

	t.Run("HTMLLikeElements", func(t *testing.T) {
		t.Parallel()
		content := []byte(`
<html>
	<head>
		<link href="https://styles.example.com/main.css" rel="stylesheet"/>
		<script src="https://scripts.example.com/app.js"/>
	</head>
	<body>
		<a href="https://link.example.com">Link</a>
		<img src="https://images.example.com/img.png"/>
	</body>
</html>`)
		links, err := p.Parse("test.xml", content)
		require.NoError(t, err)
		assert.Len(t, links, 4)
	})
}
