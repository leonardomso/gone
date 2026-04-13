package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/config"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/output"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	origWD, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, config.DefaultConfigFileName)
	configContent := `
types: [json, yaml]
scan:
  include: ["docs/**"]
  exclude: ["vendor/**"]
check:
  concurrency: 17
  timeout: 9
  retries: 3
  strict: true
output:
  format: yaml
  showAlive: true
ignore:
  domains: [example.com]
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	loaded, err := LoadConfig(false)
	require.NoError(t, err)
	assert.Equal(t, []string{"json", "yaml"}, loaded.Config().Types)
	assert.Equal(t, []string{"docs/**"}, loaded.Config().Scan.Include)
	assert.Equal(t, 17, loaded.Config().Check.Concurrency)
	assert.Equal(t, output.FormatYAML, output.Format(loaded.GetOutputFormat("")))

	noCfg, err := LoadConfig(true)
	require.NoError(t, err)
	assert.True(t, noCfg.Config().IsEmpty())
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	origWD, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, config.DefaultConfigFileName)
	require.NoError(t, os.WriteFile(configPath, []byte("output:\n  format: invalid\n"), 0o644))
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	loaded, err := LoadConfig(false)
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestLoadedConfig_GettersAndBuilders(t *testing.T) {
	t.Parallel()

	showWarnings := false
	showDead := false

	loaded := &LoadedConfig{
		cfg: &config.Config{
			Types: []string{"json", "xml"},
			Scan: config.ScanConfig{
				Include: []string{"docs/**"},
				Exclude: []string{"vendor/**"},
			},
			Check: config.CheckConfig{
				Concurrency: 12,
				Timeout:     7,
				Retries:     4,
				Strict:      true,
			},
			Output: config.OutputConfig{
				Format:       "json",
				ShowAlive:    true,
				ShowWarnings: &showWarnings,
				ShowDead:     &showDead,
				ShowStats:    true,
			},
		},
	}

	assert.Equal(t, []string{"json", "xml"}, loaded.GetTypes([]string{"md"}, []string{"md"}))
	assert.Equal(t, []string{"md"}, loaded.GetTypes([]string{"md"}, []string{"json"}))
	assert.Equal(t, []string{"yaml"}, loaded.GetTypes([]string{"yaml"}, []string{"md"}))

	opts := loaded.BuildCheckerOptions(
		checker.DefaultConcurrency,
		int(checker.DefaultTimeout.Seconds()),
		checker.DefaultMaxRetries,
	)
	assert.Equal(t, 12, opts.Concurrency)
	assert.Equal(t, 7*time.Second, opts.Timeout)
	assert.Equal(t, 4, opts.MaxRetries)

	scanOpts := loaded.BuildScanOptions("/tmp/docs", []string{"md"}, []string{"md"})
	assert.Equal(t, "/tmp/docs", scanOpts.Root)
	assert.Equal(t, []string{"json", "xml"}, scanOpts.Types)
	assert.Equal(t, []string{"docs/**"}, scanOpts.Include)
	assert.Equal(t, []string{"vendor/**"}, scanOpts.Exclude)

	assert.True(t, loaded.GetStrict(false))
	assert.Equal(t, "json", loaded.GetOutputFormat(""))
	assert.True(t, loaded.GetShowAlive(false))
	assert.False(t, loaded.GetShowWarnings(false))
	assert.False(t, loaded.GetShowDead(false))
	assert.True(t, loaded.GetShowStats(false))
}

func TestCreateFilter(t *testing.T) {
	origWD, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, config.DefaultConfigFileName)
	configContent := `
ignore:
  domains: [config.example]
  patterns: ["*.internal/*"]
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	urlFilter, err := CreateFilter(FilterOptions{
		Domains:  []string{"cli.example"},
		Patterns: []string{"*.cli/*"},
		Regex:    []string{`example\.org`},
	})
	require.NoError(t, err)
	require.NotNil(t, urlFilter)

	assert.True(t, urlFilter.ShouldIgnore("https://config.example/path", "README.md", 1))
	assert.True(t, urlFilter.ShouldIgnore("https://foo.cli/bar", "README.md", 2))
	assert.True(t, urlFilter.ShouldIgnore("https://service.internal/path", "README.md", 3))
	assert.True(t, urlFilter.ShouldIgnore("https://example.org/path", "README.md", 4))

	noRules, err := CreateFilter(FilterOptions{NoConfig: true})
	require.NoError(t, err)
	assert.Nil(t, noRules)
}

func TestCreateFilterWithConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Ignore: config.IgnoreConfig{
			Domains:  []string{"config.example"},
			Patterns: []string{"*.internal/*"},
			Regex:    []string{`ignored\.org`},
		},
	}

	urlFilter, err := CreateFilterWithConfig(
		cfg,
		[]string{"cli.example"},
		[]string{"*.cli/*"},
		[]string{`other\.org`},
	)
	require.NoError(t, err)
	require.NotNil(t, urlFilter)

	assert.True(t, urlFilter.ShouldIgnore("https://config.example/path", "a.md", 1))
	assert.True(t, urlFilter.ShouldIgnore("https://service.cli/path", "a.md", 2))
	assert.True(t, urlFilter.ShouldIgnore("https://service.internal/path", "a.md", 3))
	assert.True(t, urlFilter.ShouldIgnore("https://ignored.org/path", "a.md", 4))
	assert.True(t, urlFilter.ShouldIgnore("https://other.org/path", "a.md", 5))
}

func TestCreateFilterWithConfig_NoRules(t *testing.T) {
	t.Parallel()

	urlFilter, err := CreateFilterWithConfig(&config.Config{}, nil, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, urlFilter)
}

func TestLinkConversionAndFiltering(t *testing.T) {
	t.Parallel()

	parserLinks := []parser.Link{
		{URL: "https://allowed.example", FilePath: "README.md", Line: 3, Text: "Allowed"},
		{URL: "https://ignored.example", FilePath: "README.md", Line: 5, Text: "Ignored"},
		{URL: "https://allowed.example", FilePath: "README.md", Line: 8, Text: "Duplicate"},
	}

	converted := ConvertParserLinks(parserLinks)
	require.Len(t, converted, len(parserLinks))
	assert.Equal(t, parserLinks[0].URL, converted[0].URL)
	assert.Equal(t, parserLinks[1].FilePath, converted[1].FilePath)

	urlFilter, err := filter.New(filter.Config{Domains: []string{"ignored.example"}})
	require.NoError(t, err)

	filtered := FilterParserLinks(parserLinks, urlFilter)
	require.Len(t, filtered, 2)
	assert.Equal(t, "https://allowed.example", filtered[0].URL)
	assert.Equal(t, 1, urlFilter.IgnoredCount())
	assert.Equal(t, 1, CountUniqueURLs(filtered))
}

func TestFilterResultHelpers(t *testing.T) {
	t.Parallel()

	primary := checker.Result{
		Link:   checker.Link{URL: "https://primary.example", FilePath: "README.md", Line: 1},
		Status: checker.StatusRedirect,
	}

	results := []checker.Result{
		{Link: checker.Link{URL: "https://alive.example"}, Status: checker.StatusAlive},
		primary,
		{Link: checker.Link{URL: "https://blocked.example"}, Status: checker.StatusBlocked},
		{Link: checker.Link{URL: "https://dead.example"}, Status: checker.StatusDead},
		{Link: checker.Link{URL: "https://error.example"}, Status: checker.StatusError},
		{Link: checker.Link{URL: "https://duplicate.example"}, Status: checker.StatusDuplicate, DuplicateOf: &primary},
	}

	assert.Len(t, FilterResultsWarnings(results), 2)
	assert.Len(t, FilterResultsDead(results), 2)
	assert.Len(t, FilterResultsDuplicates(results), 1)
	assert.Len(t, FilterResultsAlive(results), 1)
	assert.Equal(t, 5, len(filterResults(results, checkRenderOptions{ShowWarnings: false})))
}

func TestCheckHelpersAndReportBuilding(t *testing.T) {
	restore := saveCheckGlobalsForTest()
	defer restore()

	outputFormat = "json"
	outputFile = ""
	showIgnored = true
	showWarnings = true
	runner := newCheckRunner(currentCheckOptions(), defaultCommandEnv(), IOStreams{})
	assert.NoError(t, runner.validateFlags())
	assert.Equal(t, ".", getPathArg(nil))
	assert.Equal(t, "docs", getPathArg([]string{"docs"}))
	assert.NoError(t, validateFileTypes([]string{"md", "json", "yaml", "toml", "xml"}))
	assert.Error(t, validateFileTypes([]string{"pdf"}))
	assert.Equal(t, 0, getIgnoredCount(nil))

	results := []checker.Result{
		{
			Link: checker.Link{
				URL:      "https://redirect.example",
				FilePath: "README.md",
				Line:     2,
			},
			Status:        checker.StatusRedirect,
			StatusCode:    301,
			FinalStatus:   200,
			FinalURL:      "https://final.example",
			RedirectChain: []checker.Redirect{{URL: "https://redirect.example", StatusCode: 301}},
		},
		{
			Link:   checker.Link{URL: "https://dead.example", FilePath: "README.md", Line: 4},
			Status: checker.StatusDead,
		},
	}
	summary := checker.Summarize(results)

	urlFilter, err := filter.New(filter.Config{Domains: []string{"ignored.example"}})
	require.NoError(t, err)
	require.True(t, urlFilter.ShouldIgnore("https://ignored.example/path", "README.md", 7))
	assert.Equal(t, 1, getIgnoredCount(urlFilter))

	report := buildReport(time.Unix(123, 0), []string{"README.md"}, results, summary, urlFilter, runner.renderOptions())
	require.NotNil(t, report)
	assert.Equal(t, summary.Total+1, report.TotalLinks)
	require.Len(t, report.Ignored, 1)
	assert.Equal(t, "https://ignored.example/path", report.Ignored[0].URL)

	assert.Equal(t, "301 → 200", formatRedirectChain(results[0]))
	assert.Equal(t, "No warnings found.", getEmptyResultsMessage(checker.Summary{Alive: 2}, runner.renderOptions()))

	showWarnings = false
	showDead = true
	runner = newCheckRunner(currentCheckOptions(), defaultCommandEnv(), IOStreams{})
	assert.Equal(t, "No dead links found.", getEmptyResultsMessage(checker.Summary{Alive: 2}, runner.renderOptions()))
}

func TestValidateCheckFlags_InvalidCombinations(t *testing.T) {
	restore := saveCheckGlobalsForTest()
	defer restore()

	outputFormat = "json"
	outputFile = "report.json"
	err := newCheckRunner(currentCheckOptions(), defaultCommandEnv(), IOStreams{}).validateFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")

	outputFile = ""
	outputFormat = "invalid"
	err = newCheckRunner(currentCheckOptions(), defaultCommandEnv(), IOStreams{}).validateFlags()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid format"))
}

func saveCheckGlobalsForTest() func() {
	savedOutputFormat := outputFormat
	savedOutputFile := outputFile
	savedShowAlive := showAlive
	savedShowWarnings := showWarnings
	savedShowDead := showDead
	savedShowAll := showAll
	savedShowIgnored := showIgnored

	return func() {
		outputFormat = savedOutputFormat
		outputFile = savedOutputFile
		showAlive = savedShowAlive
		showWarnings = savedShowWarnings
		showDead = savedShowDead
		showAll = savedShowAll
		showIgnored = savedShowIgnored
	}
}
