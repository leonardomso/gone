package cmd

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/fixer"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"
)

var (
	benchParserLinks  []parser.Link
	benchCheckerLinks []checker.Link
	benchCheckResults []checker.Result
	benchFixChanges   []fixer.FileChanges
	benchPreview      string
)

func BenchmarkPipeline_ScanParseOnly(b *testing.B) {
	for _, fixture := range []string{"small", "medium", "large"} {
		b.Run(fixture, func(b *testing.B) {
			root := benchmarkFixturePath(fixture)
			opts := benchmarkScanOptions(root)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				files, err := scanner.FindFilesWithOptions(opts)
				if err != nil {
					b.Fatal(err)
				}

				links, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, false)
				if err != nil {
					b.Fatal(err)
				}

				benchParserLinks = links
			}
		})
	}
}

func BenchmarkPipeline_ParseFilterOnly(b *testing.B) {
	root := benchmarkFixturePath("duplicate-heavy")
	files := mustScanBenchmarkFiles(b, root)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		parserLinks, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, false)
		if err != nil {
			b.Fatal(err)
		}

		urlFilter, err := filter.New(filter.Config{
			Domains: []string{"ignored.example"},
		})
		if err != nil {
			b.Fatal(err)
		}

		benchCheckerLinks = FilterParserLinks(parserLinks, urlFilter)
	}
}

func BenchmarkPipeline_FullCheck(b *testing.B) {
	server := newBenchmarkHTTPServer()
	defer server.Close()

	root := materializeBenchmarkFixture(b, "http-scenarios", map[string]string{
		"{{ALIVE_URL}}":    server.URL + "/alive",
		"{{REDIRECT_URL}}": server.URL + "/redirect",
		"{{DEAD_URL}}":     server.URL + "/dead",
		"{{ERROR_URL}}":    server.URL + "/error",
	})
	opts := benchmarkScanOptions(root)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		files, err := scanner.FindFilesWithOptions(opts)
		if err != nil {
			b.Fatal(err)
		}

		parserLinks, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, false)
		if err != nil {
			b.Fatal(err)
		}

		c := checker.New(
			checker.DefaultOptions().
				WithConcurrency(16).
				WithTimeout(2 * time.Second).
				WithMaxRetries(0),
		)
		benchCheckResults = c.CheckAll(ConvertParserLinks(parserLinks))
	}
}

func BenchmarkPipeline_FixDryRun(b *testing.B) {
	server := newBenchmarkHTTPServer()
	defer server.Close()

	root := materializeBenchmarkFixture(b, "http-scenarios", map[string]string{
		"{{ALIVE_URL}}":    server.URL + "/alive",
		"{{REDIRECT_URL}}": server.URL + "/redirect",
		"{{DEAD_URL}}":     server.URL + "/dead",
		"{{ERROR_URL}}":    server.URL + "/error",
	})
	opts := benchmarkScanOptions(root)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		files, err := scanner.FindFilesWithOptions(opts)
		if err != nil {
			b.Fatal(err)
		}

		parserLinks, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, false)
		if err != nil {
			b.Fatal(err)
		}

		c := checker.New(
			checker.DefaultOptions().
				WithConcurrency(16).
				WithTimeout(2 * time.Second).
				WithMaxRetries(0),
		)
		results := c.CheckAll(ConvertParserLinks(parserLinks))

		f := fixer.New()
		f.SetParserLinks(parserLinks)
		benchFixChanges = f.FindFixes(results)
		benchPreview = f.Preview(benchFixChanges)
	}
}

func benchmarkFixturePath(name string) string {
	return filepath.Join("..", "testdata", "bench", name)
}

func benchmarkScanOptions(root string) scanner.ScanOptions {
	return scanner.ScanOptions{
		Root:  root,
		Types: []string{"md", "json", "yaml", "toml", "xml"},
	}
}

func mustScanBenchmarkFiles(b *testing.B, root string) []string {
	b.Helper()

	files, err := scanner.FindFilesWithOptions(benchmarkScanOptions(root))
	if err != nil {
		b.Fatal(err)
	}
	return files
}

func materializeBenchmarkFixture(b *testing.B, name string, replacements map[string]string) string {
	b.Helper()

	srcRoot := benchmarkFixturePath(name)
	dstRoot := b.TempDir()

	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstRoot, relPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rewritten := string(content)
		for oldValue, newValue := range replacements {
			rewritten = strings.ReplaceAll(rewritten, oldValue, newValue)
		}

		return os.WriteFile(dstPath, []byte(rewritten), 0o644)
	})
	if err != nil {
		b.Fatal(err)
	}

	return dstRoot
}

func newBenchmarkHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alive":
			w.WriteHeader(http.StatusOK)
		case "/redirect":
			http.Redirect(w, r, "/alive", http.StatusMovedPermanently)
		case "/dead":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
