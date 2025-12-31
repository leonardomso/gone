package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMarkdownFiles(t *testing.T) {
	t.Parallel()

	t.Run("SingleFile", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/single")
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/multiple")
		require.NoError(t, err)
		assert.Len(t, files, 2)

		// Check that .txt file is not included
		for _, f := range files {
			assert.True(t, filepath.Ext(f) == ".md", "expected .md extension, got %s", f)
		}
	})

	t.Run("NestedDirectories", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/nested")
		require.NoError(t, err)
		assert.Len(t, files, 2)

		// Should find both root.md and nested.md
		var names []string
		for _, f := range files {
			names = append(names, filepath.Base(f))
		}
		sort.Strings(names)
		assert.Equal(t, []string{"nested.md", "root.md"}, names)
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/empty")
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("SkipsHiddenDirectories", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/hidden")
		require.NoError(t, err)
		assert.Len(t, files, 1)

		// Should only find visible.md, not .hidden/ignored.md
		assert.Contains(t, files[0], "visible.md")
		for _, f := range files {
			assert.NotContains(t, f, ".hidden")
		}
	})

	t.Run("MixedCaseExtensions", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/mixed_case")
		require.NoError(t, err)
		assert.Len(t, files, 3)

		// Should find .md, .MD, and .Md
		var names []string
		for _, f := range files {
			names = append(names, filepath.Base(f))
		}
		sort.Strings(names)
		assert.Equal(t, []string{"Mixed.Md", "UPPER.MD", "lower.md"}, names)
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/nonexistent")
		assert.Error(t, err)
		assert.Nil(t, files)
	})

	t.Run("SingleFileAsRoot", func(t *testing.T) {
		t.Parallel()
		files, err := FindMarkdownFiles("testdata/single/readme.md")
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("CurrentDirectory", func(t *testing.T) {
		t.Parallel()
		// Create a temp directory with a markdown file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.md")
		err := os.WriteFile(testFile, []byte("# Test"), 0o644)
		require.NoError(t, err)

		files, err := FindMarkdownFiles(tmpDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("DeeplyNested", func(t *testing.T) {
		t.Parallel()
		// Create deeply nested structure
		tmpDir := t.TempDir()
		deepPath := filepath.Join(tmpDir, "a", "b", "c", "d")
		err := os.MkdirAll(deepPath, 0o755)
		require.NoError(t, err)

		testFile := filepath.Join(deepPath, "deep.md")
		err = os.WriteFile(testFile, []byte("# Deep"), 0o644)
		require.NoError(t, err)

		files, err := FindMarkdownFiles(tmpDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "deep.md")
	})
}
