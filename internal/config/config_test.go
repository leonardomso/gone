package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFrom(t *testing.T) {
	t.Parallel()

	t.Run("ValidFullConfig", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadFrom("testdata/valid_full.yaml")
		require.NoError(t, err)

		assert.Len(t, cfg.Ignore.Domains, 3)
		assert.Contains(t, cfg.Ignore.Domains, "example.com")
		assert.Contains(t, cfg.Ignore.Domains, "localhost")
		assert.Contains(t, cfg.Ignore.Domains, "internal.company.com")

		assert.Len(t, cfg.Ignore.Patterns, 2)
		assert.Contains(t, cfg.Ignore.Patterns, "*.local/*")
		assert.Contains(t, cfg.Ignore.Patterns, "*/internal/*")

		assert.Len(t, cfg.Ignore.Regex, 2)
		assert.Contains(t, cfg.Ignore.Regex, ".*\\.test$")
		assert.Contains(t, cfg.Ignore.Regex, ".*/draft/.*")
	})

	t.Run("ValidPartialConfig", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadFrom("testdata/valid_partial.yaml")
		require.NoError(t, err)

		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "example.com")
		assert.Empty(t, cfg.Ignore.Patterns)
		assert.Empty(t, cfg.Ignore.Regex)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadFrom("testdata/empty.yaml")
		require.NoError(t, err)

		assert.Empty(t, cfg.Ignore.Domains)
		assert.Empty(t, cfg.Ignore.Patterns)
		assert.Empty(t, cfg.Ignore.Regex)
		assert.True(t, cfg.IsEmpty())
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadFrom("testdata/invalid.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("FileNotExists", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadFrom("testdata/nonexistent.yaml")
		require.NoError(t, err) // Not an error, returns empty config
		assert.NotNil(t, cfg)
		assert.True(t, cfg.IsEmpty())
	})

	t.Run("ExtraFields", func(t *testing.T) {
		t.Parallel()
		// Should ignore unknown fields without error
		cfg, err := LoadFrom("testdata/extra_fields.yaml")
		require.NoError(t, err)

		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "example.com")
	})
}

func TestLoad(t *testing.T) {
	t.Run("LoadsDefaultFile", func(t *testing.T) {
		// This test runs in the config package directory
		// where there's no .gonerc.yaml, so it should return empty config
		cfg, err := Load()
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

func TestLoadFrom_DirectoryInsteadOfFile(t *testing.T) {
	t.Parallel()
	// Trying to read a directory should return an error (not ErrNotExist)
	tmpDir := t.TempDir()

	cfg, err := LoadFrom(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadFrom_PermissionDenied(t *testing.T) {
	// Skip on Windows where permission handling is different
	if os.Getenv("CI") != "" {
		t.Skip("Skipping permission test in CI")
	}

	// Create a file with no read permissions
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gonerc.yaml")
	err := os.WriteFile(configPath, []byte("ignore:\n  domains:\n    - test.com\n"), 0o000)
	require.NoError(t, err)

	// Ensure cleanup restores permissions so temp dir can be removed
	t.Cleanup(func() {
		_ = os.Chmod(configPath, 0o644)
	})

	cfg, err := LoadFrom(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestFindAndLoad(t *testing.T) {
	t.Parallel()

	t.Run("FindsInCurrentDir", func(t *testing.T) {
		t.Parallel()
		// Create temp directory with config
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, DefaultConfigFileName)
		configContent := []byte("ignore:\n  domains:\n    - test.com\n")
		err := os.WriteFile(configPath, configContent, 0o644)
		require.NoError(t, err)

		cfg, err := FindAndLoad(tmpDir)
		require.NoError(t, err)
		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "test.com")
	})

	t.Run("FindsInParentDir", func(t *testing.T) {
		t.Parallel()
		// Create temp directory structure: parent/child
		tmpDir := t.TempDir()
		childDir := filepath.Join(tmpDir, "child")
		err := os.MkdirAll(childDir, 0o755)
		require.NoError(t, err)

		// Put config in parent
		configPath := filepath.Join(tmpDir, DefaultConfigFileName)
		configContent := []byte("ignore:\n  domains:\n    - parent.com\n")
		err = os.WriteFile(configPath, configContent, 0o644)
		require.NoError(t, err)

		// Search from child
		cfg, err := FindAndLoad(childDir)
		require.NoError(t, err)
		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "parent.com")
	})

	t.Run("NotFoundReturnsEmpty", func(t *testing.T) {
		t.Parallel()
		// Create temp directory with no config
		tmpDir := t.TempDir()

		cfg, err := FindAndLoad(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.True(t, cfg.IsEmpty())
	})

	t.Run("CloserConfigTakesPrecedence", func(t *testing.T) {
		t.Parallel()
		// Create temp directory structure: parent/child, both with configs
		tmpDir := t.TempDir()
		childDir := filepath.Join(tmpDir, "child")
		err := os.MkdirAll(childDir, 0o755)
		require.NoError(t, err)

		// Put config in parent
		parentConfig := filepath.Join(tmpDir, DefaultConfigFileName)
		err = os.WriteFile(parentConfig, []byte("ignore:\n  domains:\n    - parent.com\n"), 0o644)
		require.NoError(t, err)

		// Put config in child
		childConfig := filepath.Join(childDir, DefaultConfigFileName)
		err = os.WriteFile(childConfig, []byte("ignore:\n  domains:\n    - child.com\n"), 0o644)
		require.NoError(t, err)

		// Search from child - should find child config first
		cfg, err := FindAndLoad(childDir)
		require.NoError(t, err)
		assert.Contains(t, cfg.Ignore.Domains, "child.com")
		assert.NotContains(t, cfg.Ignore.Domains, "parent.com")
	})
}

func TestConfig_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "EmptyConfig",
			config:   Config{},
			expected: true,
		},
		{
			name: "WithDomains",
			config: Config{
				Ignore: IgnoreConfig{
					Domains: []string{"example.com"},
				},
			},
			expected: false,
		},
		{
			name: "WithPatterns",
			config: Config{
				Ignore: IgnoreConfig{
					Patterns: []string{"*.local/*"},
				},
			},
			expected: false,
		},
		{
			name: "WithRegex",
			config: Config{
				Ignore: IgnoreConfig{
					Regex: []string{".*\\.test$"},
				},
			},
			expected: false,
		},
		{
			name: "WithAll",
			config: Config{
				Ignore: IgnoreConfig{
					Domains:  []string{"example.com"},
					Patterns: []string{"*.local/*"},
					Regex:    []string{".*\\.test$"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.config.IsEmpty())
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	t.Parallel()

	t.Run("MergesBothConfigs", func(t *testing.T) {
		t.Parallel()
		cfg1 := &Config{
			Ignore: IgnoreConfig{
				Domains:  []string{"domain1.com"},
				Patterns: []string{"pattern1"},
				Regex:    []string{"regex1"},
			},
		}

		cfg2 := &Config{
			Ignore: IgnoreConfig{
				Domains:  []string{"domain2.com"},
				Patterns: []string{"pattern2"},
				Regex:    []string{"regex2"},
			},
		}

		cfg1.Merge(cfg2)

		assert.Len(t, cfg1.Ignore.Domains, 2)
		assert.Contains(t, cfg1.Ignore.Domains, "domain1.com")
		assert.Contains(t, cfg1.Ignore.Domains, "domain2.com")

		assert.Len(t, cfg1.Ignore.Patterns, 2)
		assert.Len(t, cfg1.Ignore.Regex, 2)
	})

	t.Run("MergeNilOther", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Ignore: IgnoreConfig{
				Domains: []string{"domain.com"},
			},
		}

		cfg.Merge(nil)

		// Should not panic and remain unchanged
		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "domain.com")
	})

	t.Run("MergeEmptyOther", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Ignore: IgnoreConfig{
				Domains: []string{"domain.com"},
			},
		}

		cfg.Merge(&Config{})

		assert.Len(t, cfg.Ignore.Domains, 1)
	})

	t.Run("MergeIntoEmpty", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{}

		other := &Config{
			Ignore: IgnoreConfig{
				Domains: []string{"domain.com"},
			},
		}

		cfg.Merge(other)

		assert.Len(t, cfg.Ignore.Domains, 1)
		assert.Contains(t, cfg.Ignore.Domains, "domain.com")
	})
}
