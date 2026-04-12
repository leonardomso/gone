package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/config"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/fixer"
	"github.com/leonardomso/gone/internal/output"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"
	"github.com/leonardomso/gone/internal/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChecker struct {
	results []checker.Result
}

func (s stubChecker) CheckAll([]checker.Link) []checker.Result {
	return append([]checker.Result(nil), s.results...)
}

type stubFixer struct {
	preview     string
	changes     []fixer.FileChanges
	applyAll    []fixer.FixResult
	applyToFile map[string]fixer.FixResult
}

func (*stubFixer) SetParserLinks([]parser.Link) {}

func (s *stubFixer) FindFixes([]checker.Result) []fixer.FileChanges {
	return append([]fixer.FileChanges(nil), s.changes...)
}

func (s *stubFixer) Preview([]fixer.FileChanges) string {
	return s.preview
}

func (s *stubFixer) ApplyAll([]fixer.FileChanges) []fixer.FixResult {
	return append([]fixer.FixResult(nil), s.applyAll...)
}

func (s *stubFixer) ApplyToFile(fc fixer.FileChanges) (*fixer.FixResult, error) {
	result := s.applyToFile[fc.FilePath]
	return &result, result.Error
}

func testCommandEnv() CommandEnv {
	return CommandEnv{
		LoadConfig: func(bool) (*LoadedConfig, error) {
			return &LoadedConfig{cfg: &config.Config{}}, nil
		},
		FindFiles: func(scanner.ScanOptions) ([]string, error) {
			return []string{"README.md"}, nil
		},
		ExtractLinks: func([]string, bool) ([]parser.Link, error) {
			return nil, nil
		},
		CreateFilterWithConfig: CreateFilterWithConfig,
		NewChecker: func(checker.Options) LinkChecker {
			return stubChecker{}
		},
		NewFixer: func() RedirectFixer {
			return &stubFixer{}
		},
		FormatReport: output.FormatReport,
		WriteToFile:  output.WriteToFile,
		NewStats:     stats.New,
		Now: func() time.Time {
			return time.Unix(123, 0)
		},
	}
}

func TestCheckRunner_Run_NoLinksStructuredOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := testCommandEnv()
	env.LoadConfig = func(bool) (*LoadedConfig, error) {
		return &LoadedConfig{cfg: &config.Config{
			Output: config.OutputConfig{Format: "json"},
		}}, nil
	}

	runner := newCheckRunner(CheckOptions{FileTypes: []string{"md"}}, env, IOStreams{
		Out:    &stdout,
		ErrOut: &stderr,
	})

	code := runner.Run(nil)
	require.Equal(t, 0, code)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), `"total_links": 0`)
	assert.Contains(t, stdout.String(), `"total_files": 1`)
}

func TestCheckRunner_Run_AllLinksIgnoredTextOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := testCommandEnv()
	env.ExtractLinks = func([]string, bool) ([]parser.Link, error) {
		return []parser.Link{{URL: "https://ignored.example/path", FilePath: "README.md", Line: 3}}, nil
	}

	runner := newCheckRunner(CheckOptions{
		FileTypes:     []string{"md"},
		ShowIgnored:   true,
		IgnoreDomains: []string{"ignored.example"},
	}, env, IOStreams{
		Out:    &stdout,
		ErrOut: &stderr,
	})

	code := runner.Run(nil)
	require.Equal(t, 0, code)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), "All links were ignored by filter rules.")
	assert.Contains(t, stdout.String(), "=== Ignored URLs (1) ===")
}

func TestCheckRunner_Run_FileOutputSummary(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := testCommandEnv()
	env.ExtractLinks = func([]string, bool) ([]parser.Link, error) {
		return []parser.Link{{URL: "https://alive.example", FilePath: "README.md", Line: 2}}, nil
	}
	env.NewChecker = func(checker.Options) LinkChecker {
		return stubChecker{
			results: []checker.Result{{
				Link:       checker.Link{URL: "https://alive.example", FilePath: "README.md", Line: 2},
				Status:     checker.StatusAlive,
				StatusCode: 200,
			}},
		}
	}

	outputPath := filepath.Join(t.TempDir(), "report.json")
	runner := newCheckRunner(CheckOptions{
		FileTypes:  []string{"md"},
		OutputFile: outputPath,
	}, env, IOStreams{
		Out:    &stdout,
		ErrOut: &stderr,
	})

	code := runner.Run(nil)
	require.Equal(t, 0, code)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), "Wrote report to")
	assert.Contains(t, stdout.String(), "Summary: 1 alive | 0 warnings | 0 dead | 0 duplicates")
}

func TestCheckRunner_Run_DeadLinksReturnExitCodeOne(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := testCommandEnv()
	env.ExtractLinks = func([]string, bool) ([]parser.Link, error) {
		return []parser.Link{{URL: "https://dead.example", FilePath: "README.md", Line: 2}}, nil
	}
	env.NewChecker = func(checker.Options) LinkChecker {
		return stubChecker{
			results: []checker.Result{{
				Link:       checker.Link{URL: "https://dead.example", FilePath: "README.md", Line: 2},
				Status:     checker.StatusDead,
				StatusCode: 404,
			}},
		}
	}

	runner := newCheckRunner(CheckOptions{
		FileTypes: []string{"md"},
	}, env, IOStreams{
		Out:    &stdout,
		ErrOut: &stderr,
	})

	code := runner.Run(nil)
	assert.Equal(t, 1, code)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), "Summary:")
}

func TestOutputTextAndPrintHelpers(t *testing.T) {
	t.Parallel()

	var outputBuf bytes.Buffer
	primary := checker.Result{
		Link:          checker.Link{URL: "https://redirect.example", FilePath: "README.md", Line: 1, Text: "Redirect"},
		Status:        checker.StatusRedirect,
		StatusCode:    301,
		FinalStatus:   200,
		FinalURL:      "https://final.example",
		RedirectChain: []checker.Redirect{{URL: "https://redirect.example", StatusCode: 301}},
	}
	results := []checker.Result{
		primary,
		{Link: checker.Link{URL: "https://dead.example", FilePath: "README.md", Line: 2}, Status: checker.StatusDead, StatusCode: 404},
		{Link: checker.Link{URL: "https://duplicate.example", FilePath: "README.md", Line: 3}, Status: checker.StatusDuplicate, DuplicateOf: &primary},
		{Link: checker.Link{URL: "https://alive.example", FilePath: "README.md", Line: 4}, Status: checker.StatusAlive, StatusCode: 200},
	}

	urlFilter, err := filter.New(filter.Config{Domains: []string{"ignored.example"}})
	require.NoError(t, err)
	require.True(t, urlFilter.ShouldIgnore("https://ignored.example/path", "README.md", 8))

	opts := checkRenderOptions{ShowAll: true, ShowIgnored: true}
	outputText(&outputBuf, results, checker.Summarize(results), urlFilter, opts)

	assert.Contains(t, outputBuf.String(), "=== Warnings (1) ===")
	assert.Contains(t, outputBuf.String(), "=== Dead Links (1) ===")
	assert.Contains(t, outputBuf.String(), "=== Duplicates (1) ===")
	assert.Contains(t, outputBuf.String(), "=== Alive (1) ===")
	assert.Contains(t, outputBuf.String(), "=== Ignored URLs (1) ===")

	outputBuf.Reset()
	printProgressMessage(&outputBuf, 5, 4, 3, 2, 1)
	assert.Contains(t, outputBuf.String(), "Found 5 link(s), checking 3 unique URLs (2 duplicates, 1 ignored)")
}

func TestOutputText_GroupedVsFlat(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{Link: checker.Link{URL: "https://redirect.example", FilePath: "README.md", Line: 1}, Status: checker.StatusRedirect, StatusCode: 301, FinalStatus: 200, FinalURL: "https://final.example"},
		{Link: checker.Link{URL: "https://dead.example", FilePath: "README.md", Line: 2}, Status: checker.StatusDead, StatusCode: 404},
	}
	summary := checker.Summarize(results)

	var grouped bytes.Buffer
	outputText(&grouped, results, summary, nil, checkRenderOptions{})
	assert.Contains(t, grouped.String(), "=== Warnings (1) ===")
	assert.Contains(t, grouped.String(), "=== Dead Links (1) ===")

	var flat bytes.Buffer
	outputText(&flat, results, summary, nil, checkRenderOptions{ShowDead: true})
	assert.NotContains(t, flat.String(), "=== Warnings")
	assert.NotContains(t, flat.String(), "=== Dead Links")
	assert.Contains(t, flat.String(), "[404]")
}

func TestGetEmptyResultsMessage_Variants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "No alive links found.", getEmptyResultsMessage(checker.Summary{}, checkRenderOptions{ShowAlive: true}))
	assert.Equal(t, "No warnings found.", getEmptyResultsMessage(checker.Summary{}, checkRenderOptions{ShowWarnings: true}))
	assert.Equal(t, "No dead links found.", getEmptyResultsMessage(checker.Summary{Redirects: 1}, checkRenderOptions{ShowDead: true}))
	assert.Equal(t, "All links are alive!", getEmptyResultsMessage(checker.Summary{Alive: 3}, checkRenderOptions{}))
}

func TestRunInteractiveFix_Branches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		changes      []fixer.FileChanges
		expectedCode int
		expectOut    []string
	}{
		{
			name:         "help then no",
			input:        "?\nn\n",
			changes:      []fixer.FileChanges{{FilePath: "README.md", TotalFixes: 1}},
			expectedCode: 0,
			expectOut:    []string{"Interactive mode options:", "Skipped README.md", "Skipped 1 file(s)."},
		},
		{
			name:         "invalid then yes",
			input:        "oops\ny\n",
			changes:      []fixer.FileChanges{{FilePath: "README.md", TotalFixes: 1}},
			expectedCode: 0,
			expectOut:    []string{"Invalid input.", "Fixed 1 redirect(s) in README.md", "Fixed 1 redirect(s) across 1 file(s)."},
		},
		{
			name:  "quit",
			input: "q\n",
			changes: []fixer.FileChanges{
				{FilePath: "README.md", TotalFixes: 1},
				{FilePath: "docs.md", TotalFixes: 1},
			},
			expectedCode: 2,
			expectOut:    []string{"Quitting. Remaining files were not modified.", "Skipped 2 file(s)."},
		},
		{
			name:  "all",
			input: "a\n",
			changes: []fixer.FileChanges{
				{FilePath: "README.md", TotalFixes: 1},
				{FilePath: "docs.md", TotalFixes: 1},
			},
			expectedCode: 0,
			expectOut:    []string{"Fixed 1 redirect(s) in README.md", "Fixed 2 redirect(s) across 2 file(s)."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			f := &stubFixer{
				applyToFile: map[string]fixer.FixResult{
					"README.md": {FilePath: "README.md", Applied: 1},
					"docs.md":   {FilePath: "docs.md", Applied: 1},
				},
			}

			code := runInteractiveFix(IOStreams{
				In:     strings.NewReader(tt.input),
				Out:    &stdout,
				ErrOut: &stderr,
			}, f, tt.changes)

			assert.Equal(t, tt.expectedCode, code)
			assert.Empty(t, stderr.String())
			for _, expected := range tt.expectOut {
				assert.Contains(t, stdout.String(), expected)
			}
		})
	}
}

func TestFixRunner_Run_DryRun(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := testCommandEnv()
	env.ExtractLinks = func([]string, bool) ([]parser.Link, error) {
		return []parser.Link{{URL: "https://old.example", FilePath: "README.md", Line: 2}}, nil
	}
	env.NewChecker = func(checker.Options) LinkChecker {
		return stubChecker{
			results: []checker.Result{{
				Link:        checker.Link{URL: "https://old.example", FilePath: "README.md", Line: 2},
				Status:      checker.StatusRedirect,
				StatusCode:  301,
				FinalStatus: 200,
				FinalURL:    "https://new.example",
			}},
		}
	}
	env.NewFixer = func() RedirectFixer {
		return &stubFixer{
			preview: "preview output\n",
			changes: []fixer.FileChanges{{FilePath: "README.md", TotalFixes: 1}},
		}
	}

	runner := newFixRunner(FixOptions{
		FileTypes: []string{"md"},
		DryRun:    true,
	}, env, IOStreams{
		Out:    &stdout,
		ErrOut: &stderr,
	})

	code := runner.Run(nil)
	require.Equal(t, 0, code)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), "preview output")
	assert.Contains(t, stdout.String(), "Dry-run mode: no files were modified.")
}
