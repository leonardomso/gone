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

func TestOffsetToLineCol(t *testing.T) {
	t.Parallel()

	// Content: "line1\nline2\nline3"
	// Line 1: bytes 0-5 (line1\n)
	// Line 2: bytes 6-11 (line2\n)
	// Line 3: bytes 12-16 (line3)
	content := []byte("line1\nline2\nline3")
	lines := BuildLineIndex(content)

	tests := []struct {
		name         string
		offset       int
		expectedLine int
		expectedCol  int
	}{
		{"FirstCharacter", 0, 1, 1},
		{"MiddleOfLine1", 2, 1, 3},
		{"EndOfLine1BeforeNewline", 4, 1, 5},
		{"FirstCharOfLine2", 6, 2, 1},
		{"MiddleOfLine2", 8, 2, 3},
		{"FirstCharOfLine3", 12, 3, 1},
		{"LastCharOfLine3", 16, 3, 5},
		{"NegativeOffset", -1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			line, col := OffsetToLineCol(lines, tt.offset)
			assert.Equal(t, tt.expectedLine, line, "line mismatch")
			assert.Equal(t, tt.expectedCol, col, "column mismatch")
		})
	}

	t.Run("EmptyLinesSlice", func(t *testing.T) {
		t.Parallel()
		line, col := OffsetToLineCol([]int{}, 5)
		assert.Equal(t, 1, line)
		assert.Equal(t, 1, col)
	})

	t.Run("SingleLine", func(t *testing.T) {
		t.Parallel()
		singleLines := BuildLineIndex([]byte("hello"))
		line, col := OffsetToLineCol(singleLines, 3)
		assert.Equal(t, 1, line)
		assert.Equal(t, 4, col)
	})

	t.Run("EmptyLines", func(t *testing.T) {
		t.Parallel()
		// Content: "a\n\nb" - line 1 has 'a', line 2 is empty, line 3 has 'b'
		emptyContent := []byte("a\n\nb")
		emptyLines := BuildLineIndex(emptyContent)
		// Line starts: 0, 2, 3
		line, col := OffsetToLineCol(emptyLines, 2) // Start of line 2 (empty line)
		assert.Equal(t, 2, line)
		assert.Equal(t, 1, col)

		line, col = OffsetToLineCol(emptyLines, 3) // Start of line 3
		assert.Equal(t, 3, line)
		assert.Equal(t, 1, col)
	})
}
