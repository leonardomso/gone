package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("EmptyConfig", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{})
		require.NoError(t, err)
		assert.NotNil(t, f)
		assert.False(t, f.HasRules())
	})

	t.Run("ValidDomains", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains: []string{"example.com", "test.org"},
		})
		require.NoError(t, err)
		assert.NotNil(t, f)
		assert.True(t, f.HasRules())

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 2, domains)
		assert.Equal(t, 0, globs)
		assert.Equal(t, 0, regexes)
	})

	t.Run("ValidGlobs", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			GlobPatterns: []string{"*.local/*", "*/internal/*"},
		})
		require.NoError(t, err)
		assert.NotNil(t, f)

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 0, domains)
		assert.Equal(t, 2, globs)
		assert.Equal(t, 0, regexes)
	})

	t.Run("ValidRegex", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			RegexPatterns: []string{".*\\.test$", ".*/draft/.*"},
		})
		require.NoError(t, err)
		assert.NotNil(t, f)

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 0, domains)
		assert.Equal(t, 0, globs)
		assert.Equal(t, 2, regexes)
	})

	t.Run("InvalidGlob", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			GlobPatterns: []string{"[invalid"},
		})
		assert.Error(t, err)
		assert.Nil(t, f)
		assert.Contains(t, err.Error(), "invalid glob pattern")
	})

	t.Run("InvalidRegex", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			RegexPatterns: []string{"[invalid"},
		})
		assert.Error(t, err)
		assert.Nil(t, f)
		assert.Contains(t, err.Error(), "invalid regex pattern")
	})

	t.Run("NormalizesDomains", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains: []string{"  EXAMPLE.COM  ", "TEST.org"},
		})
		require.NoError(t, err)

		// Should match lowercase versions
		assert.True(t, f.ShouldIgnore("http://example.com/path", "file.md", 1))
		assert.True(t, f.ShouldIgnore("http://test.org/path", "file.md", 1))
	})

	t.Run("SkipsEmptyStrings", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains:       []string{"", "example.com", "  "},
			GlobPatterns:  []string{"", "*.local/*"},
			RegexPatterns: []string{"", ".*\\.test$"},
		})
		require.NoError(t, err)

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 1, domains)
		assert.Equal(t, 1, globs)
		assert.Equal(t, 1, regexes)
	})

	t.Run("AllRuleTypes", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains:       []string{"example.com"},
			GlobPatterns:  []string{"*.local/*"},
			RegexPatterns: []string{".*\\.test$"},
		})
		require.NoError(t, err)

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 1, domains)
		assert.Equal(t, 1, globs)
		assert.Equal(t, 1, regexes)
	})
}

func TestShouldIgnore_Domain(t *testing.T) {
	t.Parallel()

	f, err := New(Config{
		Domains: []string{"example.com", "localhost", "internal.corp"},
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Exact matches
		{"ExactMatch", "http://example.com/page", true},
		{"ExactMatchHTTPS", "https://example.com/page", true},
		{"ExactMatchWithPath", "http://example.com/some/deep/path?query=1", true},
		{"LocalhostMatch", "http://localhost/api", true},
		{"LocalhostWithPort", "http://localhost:8080/api", true},

		// Subdomain matches
		{"SubdomainWWW", "http://www.example.com/page", true},
		{"SubdomainAPI", "http://api.example.com/v1", true},
		{"SubdomainDeep", "http://dev.api.example.com/test", true},
		{"SubdomainInternal", "http://app.internal.corp/dashboard", true},

		// Non-matches
		{"DifferentDomain", "http://other.com/page", false},
		{"PartialNoMatch", "http://notexample.com/page", false},
		{"SuffixNoMatch", "http://myexample.com/page", false},
		{"DifferentTLD", "http://example.org/page", false},
		{"EmptyHost", "http:///path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := f.ShouldIgnore(tt.url, "test.md", 1)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

func TestShouldIgnore_Glob(t *testing.T) {
	t.Parallel()

	f, err := New(Config{
		GlobPatterns: []string{
			"*://localhost*",
			"*://*/internal/*",
			"*.local",
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"LocalhostHTTP", "http://localhost/page", true},
		{"LocalhostHTTPS", "https://localhost:3000/api", true},
		{"InternalPath", "http://example.com/internal/doc", true},
		{"InternalPathHTTPS", "https://api.example.com/internal/secret", true},
		{"DotLocal", "http://myapp.local", true},

		{"NoMatch", "http://example.com/page", false},
		{"PublicPath", "http://example.com/public/doc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := f.ShouldIgnore(tt.url, "test.md", 1)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

func TestShouldIgnore_Regex(t *testing.T) {
	t.Parallel()

	f, err := New(Config{
		RegexPatterns: []string{
			`.*\.test$`,
			`.*/v[0-9]+/draft/.*`,
			`^https://private\.`,
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"DotTestSuffix", "http://example.test", true},
		{"DotTestWithPath", "http://api.example.test", true},
		{"DraftV1", "http://example.com/v1/draft/doc", true},
		{"DraftV2", "https://api.example.com/v2/draft/page", true},
		{"PrivateHTTPS", "https://private.example.com/secret", true},

		{"NoMatchDotTest", "http://example.com/test", false},
		{"NoMatchDraft", "http://example.com/draft/doc", false}, // Missing version
		{"NoMatchPrivate", "http://private.example.com", false}, // HTTP not HTTPS
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := f.ShouldIgnore(tt.url, "test.md", 1)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

func TestShouldIgnore_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NilFilter", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		assert.False(t, f.ShouldIgnore("http://example.com", "test.md", 1))
	})

	t.Run("InvalidURL", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		// Malformed URL - shouldn't crash
		assert.False(t, f.ShouldIgnore("not-a-valid-url", "test.md", 1))
		assert.False(t, f.ShouldIgnore("://missing-scheme", "test.md", 1))
	})

	t.Run("EmptyURL", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		assert.False(t, f.ShouldIgnore("", "test.md", 1))
	})

	t.Run("URLWithUserInfo", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		assert.True(t, f.ShouldIgnore("http://user:pass@example.com/page", "test.md", 1))
	})

	t.Run("URLWithFragment", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		assert.True(t, f.ShouldIgnore("http://example.com/page#section", "test.md", 1))
	})

	t.Run("IPv4Address", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"127.0.0.1"}})
		require.NoError(t, err)

		assert.True(t, f.ShouldIgnore("http://127.0.0.1:8080/api", "test.md", 1))
	})
}

func TestShouldIgnore_Priority(t *testing.T) {
	t.Parallel()

	// Filter with all rule types that could match the same URL
	f, err := New(Config{
		Domains:       []string{"example.com"},
		GlobPatterns:  []string{"*example*"},
		RegexPatterns: []string{".*example.*"},
	})
	require.NoError(t, err)

	// Should match (domain check happens first)
	result := f.ShouldIgnore("http://example.com/page", "test.md", 1)
	assert.True(t, result)

	// Check that it was matched by domain (fastest path)
	ignored := f.IgnoredURLs()
	require.Len(t, ignored, 1)
	assert.Equal(t, "domain", ignored[0].Type)
}

func TestFilter_IgnoredTracking(t *testing.T) {
	t.Parallel()

	t.Run("IgnoredCount", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		assert.Equal(t, 0, f.IgnoredCount())

		f.ShouldIgnore("http://example.com/1", "file1.md", 1)
		assert.Equal(t, 1, f.IgnoredCount())

		f.ShouldIgnore("http://example.com/2", "file2.md", 5)
		assert.Equal(t, 2, f.IgnoredCount())

		// Non-matching URL shouldn't increase count
		f.ShouldIgnore("http://other.com/page", "file3.md", 10)
		assert.Equal(t, 2, f.IgnoredCount())
	})

	t.Run("IgnoredURLs", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains:      []string{"example.com"},
			GlobPatterns: []string{"*localhost*"},
		})
		require.NoError(t, err)

		f.ShouldIgnore("http://example.com/page", "doc.md", 10)
		f.ShouldIgnore("http://localhost:3000/api", "test.md", 20)

		ignored := f.IgnoredURLs()
		require.Len(t, ignored, 2)

		// First ignored URL
		assert.Equal(t, "http://example.com/page", ignored[0].URL)
		assert.Equal(t, "doc.md", ignored[0].File)
		assert.Equal(t, 10, ignored[0].Line)
		assert.Equal(t, "domain", ignored[0].Type)
		assert.Equal(t, "example.com", ignored[0].Rule)

		// Second ignored URL
		assert.Equal(t, "http://localhost:3000/api", ignored[1].URL)
		assert.Equal(t, "test.md", ignored[1].File)
		assert.Equal(t, 20, ignored[1].Line)
		assert.Equal(t, "pattern", ignored[1].Type)
	})

	t.Run("Reset", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)

		f.ShouldIgnore("http://example.com/1", "file.md", 1)
		f.ShouldIgnore("http://example.com/2", "file.md", 2)
		assert.Equal(t, 2, f.IgnoredCount())

		f.Reset()
		assert.Equal(t, 0, f.IgnoredCount())
		assert.Empty(t, f.IgnoredURLs())
	})

	t.Run("NilFilterIgnoredCount", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		assert.Equal(t, 0, f.IgnoredCount())
	})

	t.Run("NilFilterIgnoredURLs", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		assert.Nil(t, f.IgnoredURLs())
	})

	t.Run("NilFilterReset", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		// Should not panic
		f.Reset()
	})
}

func TestFilter_HasRules(t *testing.T) {
	t.Parallel()

	t.Run("NoRules", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{})
		require.NoError(t, err)
		assert.False(t, f.HasRules())
	})

	t.Run("WithDomains", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{Domains: []string{"example.com"}})
		require.NoError(t, err)
		assert.True(t, f.HasRules())
	})

	t.Run("WithGlobs", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{GlobPatterns: []string{"*.local"}})
		require.NoError(t, err)
		assert.True(t, f.HasRules())
	})

	t.Run("WithRegex", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{RegexPatterns: []string{".*test.*"}})
		require.NoError(t, err)
		assert.True(t, f.HasRules())
	})

	t.Run("NilFilter", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		assert.False(t, f.HasRules())
	})
}

func TestFilter_Stats(t *testing.T) {
	t.Parallel()

	t.Run("AllTypes", func(t *testing.T) {
		t.Parallel()
		f, err := New(Config{
			Domains:       []string{"a.com", "b.com", "c.com"},
			GlobPatterns:  []string{"*.local", "*.test"},
			RegexPatterns: []string{".*pattern.*"},
		})
		require.NoError(t, err)

		domains, globs, regexes := f.Stats()
		assert.Equal(t, 3, domains)
		assert.Equal(t, 2, globs)
		assert.Equal(t, 1, regexes)
	})

	t.Run("NilFilter", func(t *testing.T) {
		t.Parallel()
		var f *Filter
		domains, globs, regexes := f.Stats()
		assert.Equal(t, 0, domains)
		assert.Equal(t, 0, globs)
		assert.Equal(t, 0, regexes)
	})
}
