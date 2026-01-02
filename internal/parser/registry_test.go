package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParser is a test helper that implements FileParser.
type mockParser struct {
	extensions []string
}

func newMockParser(exts ...string) *mockParser {
	return &mockParser{extensions: exts}
}

func (m *mockParser) Extensions() []string  { return m.extensions }
func (*mockParser) Validate(_ []byte) error { return nil }
func (*mockParser) Parse(_ string, _ []byte) ([]Link, error) {
	return nil, nil
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.parsers)
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("RegistersParser", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		p := newMockParser(".md", ".mdx", ".markdown")

		r.Register(p)

		for _, ext := range p.Extensions() {
			got, ok := r.Get(ext)
			assert.True(t, ok)
			assert.Equal(t, p, got)
		}
	})

	t.Run("OverwritesExisting", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		p1 := newMockParser(".md")
		p2 := newMockParser(".md")

		r.Register(p1)
		r.Register(p2)

		got, ok := r.Get(".md")
		assert.True(t, ok)
		assert.Equal(t, p2, got)
	})

	t.Run("NormalizesExtension", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		p := newMockParser(".md")

		r.Register(p)

		// Should work with various formats
		testCases := []string{".md", "md", ".MD", "MD"}
		for _, ext := range testCases {
			got, ok := r.Get(ext)
			assert.True(t, ok, "Failed for extension: %s", ext)
			assert.Equal(t, p, got)
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsRegisteredParser", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		p := newMockParser(".json")
		r.Register(p)

		got, ok := r.Get(".json")
		assert.True(t, ok)
		assert.Equal(t, p, got)
	})

	t.Run("ReturnsFalseForUnregistered", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		got, ok := r.Get(".unknown")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		p := newMockParser(".json")
		r.Register(p)

		_, ok := r.Get(".JSON")
		assert.True(t, ok)

		_, ok = r.Get(".Json")
		assert.True(t, ok)
	})
}

func TestRegistry_GetForFile(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsParserForFilename", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))
		r.Register(newMockParser(".yaml", ".yml"))

		tests := []struct {
			filename string
			wantOK   bool
		}{
			{"readme.md", true},
			{"README.MD", true},
			{"data.json", true},
			{"config.yaml", true},
			{"config.yml", true},
			{"unknown.xyz", false},
			{"noextension", false},
		}

		for _, tt := range tests {
			_, ok := r.GetForFile(tt.filename)
			assert.Equal(t, tt.wantOK, ok, "GetForFile(%q)", tt.filename)
		}
	})
}

func TestRegistry_SupportedTypes(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsAllTypes", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md", ".mdx", ".markdown"))
		r.Register(newMockParser(".json"))
		r.Register(newMockParser(".yaml", ".yml"))

		types := r.SupportedTypes()

		// Should have md, mdx, markdown, json, yaml, yml
		assert.GreaterOrEqual(t, len(types), 3)
		assert.Contains(t, types, "md")
		assert.Contains(t, types, "json")
		assert.Contains(t, types, "yaml")
	})

	t.Run("EmptyForEmptyRegistry", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		types := r.SupportedTypes()
		assert.Empty(t, types)
	})
}

func TestRegistry_SupportedExtensions(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsAllExtensions", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))

		exts := r.SupportedExtensions()

		assert.Contains(t, exts, ".md")
		assert.Contains(t, exts, ".json")
	})
}

func TestRegistry_HasParser(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(newMockParser(".md"))

	assert.True(t, r.HasParser(".md"))
	assert.True(t, r.HasParser("md"))
	assert.False(t, r.HasParser(".xyz"))
}

func TestRegistry_ExtensionsForTypes(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsExtensions", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md", ".mdx", ".markdown"))
		r.Register(newMockParser(".json"))
		r.Register(newMockParser(".yaml", ".yml"))

		exts, err := r.ExtensionsForTypes([]string{"md", "json"})
		require.NoError(t, err)
		assert.Contains(t, exts, ".md")
		assert.Contains(t, exts, ".json")
	})

	t.Run("ErrorsOnUnsupportedType", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))

		_, err := r.ExtensionsForTypes([]string{"md", "unknown"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported file type")
	})
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsGlobalRegistry", func(t *testing.T) {
		t.Parallel()
		r := DefaultRegistry()
		assert.NotNil(t, r)
	})

	// Note: TestDefaultRegistry_HasBuiltinParsers is skipped here
	// because the subpackages need to be imported to register.
	// This test should be in an integration test package.
}

func TestGetParser(t *testing.T) {
	t.Parallel()

	// Register a mock parser first
	RegisterParser(newMockParser(".test"))

	p, ok := GetParser(".test")
	assert.True(t, ok)
	assert.NotNil(t, p)
}

func TestGetParserForFile(t *testing.T) {
	t.Parallel()

	// Register mock parsers
	RegisterParser(newMockParser(".md"))
	RegisterParser(newMockParser(".json"))

	p, ok := GetParserForFile("readme.md")
	assert.True(t, ok)
	assert.NotNil(t, p)

	p, ok = GetParserForFile("data.json")
	assert.True(t, ok)
	assert.NotNil(t, p)
}

func TestSupportedFileTypes(t *testing.T) {
	t.Parallel()

	// Register mock parsers to ensure types exist
	RegisterParser(newMockParser(".md"))
	RegisterParser(newMockParser(".json"))
	RegisterParser(newMockParser(".yaml"))

	types := SupportedFileTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "md")
	assert.Contains(t, types, "json")
	assert.Contains(t, types, "yaml")
}

func TestNormalizeExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{".md", ".md"},
		{"md", ".md"},
		{".MD", ".md"},
		{"MD", ".md"},
		{".Json", ".json"},
		{"YAML", ".yaml"},
	}

	for _, tt := range tests {
		result := normalizeExtension(tt.input)
		assert.Equal(t, tt.expected, result, "normalizeExtension(%q)", tt.input)
	}
}

// TestRegistry_EdgeCases tests edge cases for the Registry.
func TestRegistry_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("ConcurrentAccess", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		// Pre-register some parsers
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))
		r.Register(newMockParser(".yaml"))

		// Run concurrent reads and writes
		done := make(chan bool)
		for range 10 {
			go func() {
				for range 100 {
					// Concurrent reads
					r.Get(".md")
					r.Get(".json")
					r.GetForFile("test.yaml")
					r.SupportedTypes()
					r.SupportedExtensions()
					r.HasParser(".md")
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}
	})

	t.Run("SupportedTypesIsSorted", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".yaml"))
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))

		types := r.SupportedTypes()

		// Verify the result is sorted
		for i := 1; i < len(types); i++ {
			assert.LessOrEqual(t, types[i-1], types[i],
				"SupportedTypes() should be sorted, but %q > %q", types[i-1], types[i])
		}
	})

	t.Run("EmptyExtension", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))

		_, ok := r.Get("")
		assert.False(t, ok)
	})

	t.Run("GetForFileWithNoExtension", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))

		_, ok := r.GetForFile("Makefile")
		assert.False(t, ok)

		_, ok = r.GetForFile("README")
		assert.False(t, ok)
	})

	t.Run("GetForFileWithPath", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))

		_, ok := r.GetForFile("/path/to/docs/readme.md")
		assert.True(t, ok)

		_, ok = r.GetForFile("./relative/path/data.json")
		assert.True(t, ok)
	})

	t.Run("ExtensionsForTypesEmpty", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))

		exts, err := r.ExtensionsForTypes([]string{})
		require.NoError(t, err)
		assert.Empty(t, exts)
	})

	t.Run("ExtensionsForTypesCaseInsensitive", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))

		exts, err := r.ExtensionsForTypes([]string{"MD", "JSON"})
		require.NoError(t, err)
		assert.Contains(t, exts, ".md")
		assert.Contains(t, exts, ".json")
	})

	t.Run("MultipleParsersWithSameExtension", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		p1 := newMockParser(".md")
		p2 := newMockParser(".md")

		r.Register(p1)
		r.Register(p2)

		// The second registration should overwrite the first
		got, ok := r.Get(".md")
		assert.True(t, ok)
		assert.Equal(t, p2, got)
	})

	t.Run("SupportedExtensionsNoDuplicates", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))
		r.Register(newMockParser(".json"))
		r.Register(newMockParser(".yaml"))

		exts := r.SupportedExtensions()

		// Check for duplicates
		seen := map[string]bool{}
		for _, ext := range exts {
			assert.False(t, seen[ext], "Duplicate extension found: %s", ext)
			seen[ext] = true
		}
	})

	t.Run("HasParserWithVariousFormats", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		r.Register(newMockParser(".md"))

		// All these should work
		assert.True(t, r.HasParser(".md"))
		assert.True(t, r.HasParser("md"))
		assert.True(t, r.HasParser(".MD"))
		assert.True(t, r.HasParser("MD"))
		assert.True(t, r.HasParser(".Md"))
	})
}
