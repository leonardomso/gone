package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "EmptyString",
			text:     "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "WhitespaceOnly",
			text:     "   ",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "ShorterThanMax",
			text:     "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "ExactlyMaxLength",
			text:     "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "LongerThanMax",
			text:     "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "MaxLenLessThan4",
			text:     "test",
			maxLen:   3,
			expected: "test",
		},
		{
			name:     "MaxLenZero",
			text:     "test",
			maxLen:   0,
			expected: "test",
		},
		{
			name:     "MaxLenNegative",
			text:     "test",
			maxLen:   -5,
			expected: "test",
		},
		{
			name:     "Unicode",
			text:     "héllo",
			maxLen:   10,
			expected: "héllo",
		},
		{
			name:     "TrimsWhitespace",
			text:     "  hello  ",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "ExactlyMaxLenOf4",
			text:     "testing",
			maxLen:   4,
			expected: "t...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := TruncateText(tt.text, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		maxLen   int
		expected string
	}{
		{
			name:     "EmptyURL",
			url:      "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "ShorterThanMax",
			url:      "http://x.co",
			maxLen:   20,
			expected: "http://x.co",
		},
		{
			name:     "ExactlyMaxLength",
			url:      "http://x.co",
			maxLen:   11,
			expected: "http://x.co",
		},
		{
			name:     "LongerThanMax",
			url:      "http://example.com/path",
			maxLen:   15,
			expected: "http://examp...",
		},
		{
			name:     "VeryLongURL",
			url:      "https://example.com/very/long/path/to/resource?query=param&foo=bar",
			maxLen:   30,
			expected: "https://example.com/very/lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := TruncateURL(tt.url, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountUniqueStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		items    []string
		expected int
	}{
		{
			name:     "NilSlice",
			items:    nil,
			expected: 0,
		},
		{
			name:     "EmptySlice",
			items:    []string{},
			expected: 0,
		},
		{
			name:     "SingleItem",
			items:    []string{"a"},
			expected: 1,
		},
		{
			name:     "AllUnique",
			items:    []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "AllDuplicates",
			items:    []string{"a", "a", "a"},
			expected: 1,
		},
		{
			name:     "MixedDuplicates",
			items:    []string{"a", "b", "a", "c", "b"},
			expected: 3,
		},
		{
			name:     "EmptyStrings",
			items:    []string{"", "", "a"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := CountUniqueStrings(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}
