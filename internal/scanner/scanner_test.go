package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFiles(t *testing.T) {
	t.Parallel()

	t.Run("SingleFile", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/single", []string{".md"})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/multiple", []string{".md"})
		require.NoError(t, err)
		assert.Len(t, files, 2)

		// Check that .txt file is not included
		for _, f := range files {
			assert.True(t, filepath.Ext(f) == ".md", "expected .md extension, got %s", f)
		}
	})

	t.Run("NestedDirectories", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/nested", []string{".md"})
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
		files, err := FindFiles("testdata/empty", []string{".md"})
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("SkipsHiddenDirectories", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/hidden", []string{".md"})
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
		files, err := FindFiles("testdata/mixed_case", []string{".md"})
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
		files, err := FindFiles("testdata/nonexistent", []string{".md"})
		assert.Error(t, err)
		assert.Nil(t, files)
	})

	t.Run("SingleFileAsRoot", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/single/readme.md", []string{".md"})
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

		files, err := FindFiles(tmpDir, []string{".md"})
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

		files, err := FindFiles(tmpDir, []string{".md"})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "deep.md")
	})

	t.Run("EmptyExtensions", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/single", nil)
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("MultipleExtensions", func(t *testing.T) {
		t.Parallel()
		files, err := FindFiles("testdata/multiple", []string{".md", ".txt"})
		require.NoError(t, err)
		assert.Len(t, files, 3) // 2 .md + 1 .txt
	})
}

func TestFindFilesByTypes(t *testing.T) {
	t.Parallel()

	t.Run("MDTypeIncludesMDXAndMarkdown", func(t *testing.T) {
		t.Parallel()
		// Test that "md" type finds .md, .mdx, and .markdown files
		files, err := FindFilesByTypes("testdata/mdx", []string{"md"})
		require.NoError(t, err)
		assert.Len(t, files, 3)

		// Collect extensions found
		var extensions []string
		for _, f := range files {
			extensions = append(extensions, filepath.Ext(f))
		}
		sort.Strings(extensions)
		assert.Equal(t, []string{".markdown", ".md", ".mdx"}, extensions)
	})

	t.Run("YAMLTypeIncludesYML", func(t *testing.T) {
		t.Parallel()
		// Create temp directory with .yaml and .yml files
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("key: value"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "other.yml"), []byte("key: value"), 0o644)
		require.NoError(t, err)

		files, err := FindFilesByTypes(tmpDir, []string{"yaml"})
		require.NoError(t, err)
		assert.Len(t, files, 2)

		var extensions []string
		for _, f := range files {
			extensions = append(extensions, filepath.Ext(f))
		}
		sort.Strings(extensions)
		assert.Equal(t, []string{".yaml", ".yml"}, extensions)
	})

	t.Run("MultipleTypes", func(t *testing.T) {
		t.Parallel()
		// Create temp directory with various file types
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "component.mdx"), []byte("# MDX"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("ignored"), 0o644)
		require.NoError(t, err)

		files, err := FindFilesByTypes(tmpDir, []string{"md", "json"})
		require.NoError(t, err)
		assert.Len(t, files, 3) // readme.md, component.mdx, config.json
	})

	t.Run("EmptyTypes", func(t *testing.T) {
		t.Parallel()
		files, err := FindFilesByTypes("testdata/single", []string{})
		require.NoError(t, err)
		assert.Nil(t, files)
	})

	t.Run("JSONType", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Test"), 0o644)
		require.NoError(t, err)

		files, err := FindFilesByTypes(tmpDir, []string{"json"})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "data.json")
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "readme.MD"), []byte("# Test"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "component.MDX"), []byte("# MDX"), 0o644)
		require.NoError(t, err)

		files, err := FindFilesByTypes(tmpDir, []string{"md"})
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})
}

func TestFindFilesWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("BasicScan", func(t *testing.T) {
		t.Parallel()
		files, err := FindFilesWithOptions(ScanOptions{
			Root:  "testdata/nested",
			Types: []string{"md"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("WithIncludePattern", func(t *testing.T) {
		t.Parallel()
		// Create temp structure: docs/readme.md, src/code.md
		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		srcDir := filepath.Join(tmpDir, "src")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.MkdirAll(srcDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "readme.md"), []byte("# Docs"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "code.md"), []byte("# Code"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.md"), []byte("# Root"), 0o644))

		// Only include files in docs/**
		files, err := FindFilesWithOptions(ScanOptions{
			Root:    tmpDir,
			Types:   []string{"md"},
			Include: []string{"docs/**"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("WithExcludePattern", func(t *testing.T) {
		t.Parallel()
		// Create temp structure with vendor directory
		tmpDir := t.TempDir()
		vendorDir := filepath.Join(tmpDir, "vendor")
		require.NoError(t, os.MkdirAll(vendorDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Main"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "dep.md"), []byte("# Dep"), 0o644))

		// Exclude vendor/**
		files, err := FindFilesWithOptions(ScanOptions{
			Root:    tmpDir,
			Types:   []string{"md"},
			Exclude: []string{"vendor/**"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")

		// Verify vendor file was excluded
		for _, f := range files {
			assert.NotContains(t, f, "vendor")
		}
	})

	t.Run("WithBothIncludeAndExclude", func(t *testing.T) {
		t.Parallel()
		// Create temp structure: docs/readme.md, docs/internal/secret.md
		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		internalDir := filepath.Join(docsDir, "internal")
		require.NoError(t, os.MkdirAll(internalDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "readme.md"), []byte("# Docs"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(internalDir, "secret.md"), []byte("# Secret"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.md"), []byte("# Other"), 0o644))

		// Include docs/**, but exclude **/internal/**
		files, err := FindFilesWithOptions(ScanOptions{
			Root:    tmpDir,
			Types:   []string{"md"},
			Include: []string{"docs/**"},
			Exclude: []string{"**/internal/**"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("MultipleExcludePatterns", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		vendorDir := filepath.Join(tmpDir, "vendor")
		nodeModules := filepath.Join(tmpDir, "node_modules")
		require.NoError(t, os.MkdirAll(vendorDir, 0o755))
		require.NoError(t, os.MkdirAll(nodeModules, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Main"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "dep.md"), []byte("# Vendor"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(nodeModules, "pkg.md"), []byte("# NPM"), 0o644))

		files, err := FindFilesWithOptions(ScanOptions{
			Root:    tmpDir,
			Types:   []string{"md"},
			Exclude: []string{"vendor/**", "node_modules/**"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "readme.md")
	})

	t.Run("EmptyIncludeReturnsAll", func(t *testing.T) {
		t.Parallel()
		files, err := FindFilesWithOptions(ScanOptions{
			Root:    "testdata/nested",
			Types:   []string{"md"},
			Include: []string{}, // Empty means no filtering
		})
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("SpecificFilePattern", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Readme"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.md"), []byte("# Other"), 0o644))

		// Include only README.md
		files, err := FindFilesWithOptions(ScanOptions{
			Root:    tmpDir,
			Types:   []string{"md"},
			Include: []string{"README.md"},
		})
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], "README.md")
	})

	t.Run("InvalidIncludePattern", func(t *testing.T) {
		t.Parallel()
		_, err := FindFilesWithOptions(ScanOptions{
			Root:    "testdata/single",
			Types:   []string{"md"},
			Include: []string{"[invalid"},
		})
		assert.Error(t, err)
	})

	t.Run("InvalidExcludePattern", func(t *testing.T) {
		t.Parallel()
		_, err := FindFilesWithOptions(ScanOptions{
			Root:    "testdata/single",
			Types:   []string{"md"},
			Exclude: []string{"[invalid"},
		})
		assert.Error(t, err)
	})
}
