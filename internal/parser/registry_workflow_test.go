package parser_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/leonardomso/gone/internal/parser"
	_ "github.com/leonardomso/gone/internal/parser/json"
	_ "github.com/leonardomso/gone/internal/parser/markdown"
	_ "github.com/leonardomso/gone/internal/parser/toml"
	_ "github.com/leonardomso/gone/internal/parser/xml"
	_ "github.com/leonardomso/gone/internal/parser/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLinksWithRegistry(t *testing.T) {
	t.Run("UnsupportedExtension", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "README.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("https://example.com"), 0o644))

		links, err := parser.ExtractLinksWithRegistry(filePath, true)
		require.Error(t, err)
		assert.Nil(t, links)
		assert.ErrorContains(t, err, "no parser registered")
	})

	t.Run("NonStrictSkipsMalformedFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "broken.json")
		require.NoError(t, os.WriteFile(filePath, []byte(`{"url":`), 0o644))

		links, err := parser.ExtractLinksWithRegistry(filePath, false)
		require.NoError(t, err)
		assert.Nil(t, links)
	})

	t.Run("StrictReturnsParseError", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "broken.json")
		require.NoError(t, os.WriteFile(filePath, []byte(`{"url":`), 0o644))

		links, err := parser.ExtractLinksWithRegistry(filePath, true)
		require.Error(t, err)
		assert.Nil(t, links)
		assert.ErrorContains(t, err, "invalid JSON")
	})
}

func TestExtractLinksFromMultipleFilesWithRegistry(t *testing.T) {
	t.Run("SequentialMixedSupportedAndUnsupportedFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "README.md")
		txtPath := filepath.Join(tmpDir, "notes.txt")

		require.NoError(t, os.WriteFile(mdPath, []byte("[docs](https://example.com/docs)\n"), 0o644))
		require.NoError(t, os.WriteFile(txtPath, []byte("https://ignored.example"), 0o644))

		links, err := parser.ExtractLinksFromMultipleFilesWithRegistry([]string{mdPath, txtPath}, true)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs", links[0].URL)
	})

	t.Run("ParallelAggregatesAcrossFormats", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := map[string]string{
			filepath.Join(tmpDir, "README.md"):   `[docs](https://example.com/md)` + "\n",
			filepath.Join(tmpDir, "links.json"):  `{"url":"https://example.com/json"}`,
			filepath.Join(tmpDir, "config.yaml"): "url: https://example.com/yaml\n",
			filepath.Join(tmpDir, "config.toml"): "url = 'https://example.com/toml'\n",
		}

		for path, content := range files {
			require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		}

		paths := make([]string, 0, len(files))
		for path := range files {
			paths = append(paths, path)
		}

		links, err := parser.ExtractLinksFromMultipleFilesWithRegistry(paths, true)
		require.NoError(t, err)
		require.Len(t, links, 4)

		urls := make([]string, 0, len(links))
		for _, link := range links {
			urls = append(urls, link.URL)
		}
		sort.Strings(urls)

		assert.Equal(t, []string{
			"https://example.com/json",
			"https://example.com/md",
			"https://example.com/toml",
			"https://example.com/yaml",
		}, urls)
	})

	t.Run("ParallelStrictReturnsFirstParseError", func(t *testing.T) {
		tmpDir := t.TempDir()
		validMD := filepath.Join(tmpDir, "README.md")
		validYAML := filepath.Join(tmpDir, "config.yaml")
		brokenJSON := filepath.Join(tmpDir, "broken.json")

		require.NoError(t, os.WriteFile(validMD, []byte("[docs](https://example.com)\n"), 0o644))
		require.NoError(t, os.WriteFile(validYAML, []byte("url: https://example.org\n"), 0o644))
		require.NoError(t, os.WriteFile(brokenJSON, []byte(`{"url":`), 0o644))

		links, err := parser.ExtractLinksFromMultipleFilesWithRegistry([]string{validMD, validYAML, brokenJSON}, true)
		require.Error(t, err)
		assert.Nil(t, links)
		assert.ErrorContains(t, err, "broken.json")
	})
}
