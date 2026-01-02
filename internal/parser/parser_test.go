package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestIsHTTPURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url      string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"HTTP://EXAMPLE.COM", false}, // Case sensitive
		{"ftp://example.com", false},
		{"mailto:test@example.com", false},
		{"tel:+1234567890", false},
		{"#section", false},
		{"./relative/path", false},
		{"/absolute/path", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsHTTPURL(tt.url))
		})
	}
}

func TestCleanURLTrailing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "http://example.com"},
		{"http://example.com.", "http://example.com"},
		{"http://example.com,", "http://example.com"},
		{"http://example.com;", "http://example.com"},
		{"http://example.com:", "http://example.com"},
		{"http://example.com)", "http://example.com"},
		{"http://example.com]", "http://example.com"},
		{"http://example.com}", "http://example.com"},
		{"http://example.com\"", "http://example.com"},
		{"http://example.com'", "http://example.com"},
		{"http://example.com.,;:)]}", "http://example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, CleanURLTrailing(tt.input))
		})
	}
}

func TestBuildLineIndex(t *testing.T) {
	t.Parallel()

	t.Run("EmptyContent", func(t *testing.T) {
		t.Parallel()
		lines := BuildLineIndex([]byte{})
		assert.Equal(t, []int{0}, lines)
	})

	t.Run("SingleLine", func(t *testing.T) {
		t.Parallel()
		lines := BuildLineIndex([]byte("hello"))
		assert.Equal(t, []int{0}, lines)
	})

	t.Run("MultipleLines", func(t *testing.T) {
		t.Parallel()
		content := []byte("line1\nline2\nline3")
		lines := BuildLineIndex(content)
		assert.Equal(t, []int{0, 6, 12}, lines)
	})

	t.Run("TrailingNewline", func(t *testing.T) {
		t.Parallel()
		content := []byte("line1\nline2\n")
		lines := BuildLineIndex(content)
		assert.Equal(t, []int{0, 6, 12}, lines)
	})

	t.Run("EmptyLines", func(t *testing.T) {
		t.Parallel()
		content := []byte("line1\n\n\nline4")
		lines := BuildLineIndex(content)
		assert.Equal(t, []int{0, 6, 7, 8}, lines)
	})
}

func TestURLRegex(t *testing.T) {
	t.Parallel()

	t.Run("MatchesHTTP", func(t *testing.T) {
		t.Parallel()
		matches := URLRegex.FindAllString("Visit http://example.com for more", -1)
		assert.Equal(t, []string{"http://example.com"}, matches)
	})

	t.Run("MatchesHTTPS", func(t *testing.T) {
		t.Parallel()
		matches := URLRegex.FindAllString("Visit https://secure.example.com/path", -1)
		assert.Equal(t, []string{"https://secure.example.com/path"}, matches)
	})

	t.Run("MatchesMultiple", func(t *testing.T) {
		t.Parallel()
		text := "See http://one.com and https://two.com"
		matches := URLRegex.FindAllString(text, -1)
		assert.Len(t, matches, 2)
	})

	t.Run("StopsAtWhitespace", func(t *testing.T) {
		t.Parallel()
		matches := URLRegex.FindAllString("http://example.com/path more text", -1)
		assert.Equal(t, []string{"http://example.com/path"}, matches)
	})
}

func TestParseError(t *testing.T) {
	t.Parallel()

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		err := &ParseError{FilePath: "test.md", Err: assert.AnError}
		assert.Contains(t, err.Error(), "test.md")
		assert.Contains(t, err.Error(), assert.AnError.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()
		err := &ParseError{FilePath: "test.md", Err: assert.AnError}
		assert.Equal(t, assert.AnError, err.Unwrap())
	})
}
