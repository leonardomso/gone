package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/output"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"
	"github.com/leonardomso/gone/internal/stats"

	"github.com/spf13/cobra"
)

// Flag variables for the check command.
var (
	outputFormat string
	outputFile   string
	concurrency  int
	timeout      int
	retries      int
	showAlive    bool
	showWarnings bool
	showDead     bool
	showAll      bool
	showStats    bool

	// File type flags.
	fileTypes  []string
	strictMode bool

	// Ignore flags.
	ignoreDomains  []string
	ignorePatterns []string
	ignoreRegex    []string
	showIgnored    bool
	noConfig       bool
)

// checkCmd represents the check command.
var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Scan files for dead links",
	Long: `Scan a directory for files and check all HTTP/HTTPS links.

If no path is provided, scans the current directory.
By default, scans only markdown files (.md).
Use --types to scan additional file types.

By default, shows warnings (redirects, blocked) and dead links.
Use flags to filter what's displayed.

Exit codes:
  0 - All links are alive or only have warnings
  1 - Dead links or errors found

Examples:
  gone check                         # Scan current directory (markdown only)
  gone check ./docs                  # Scan specific directory  
  gone check --types=md,json,yaml    # Scan markdown, JSON, and YAML files
  gone check --types=toml,xml        # Scan TOML and XML files
  gone check --types=json --strict   # Fail on malformed JSON files
  gone check --format=json           # Output JSON to stdout
  gone check --format=yaml           # Output YAML to stdout
  gone check --output=report.json    # Write JSON report to file
  gone check --output=report.md      # Write Markdown report to file
  gone check --output=report.junit.xml  # Write JUnit XML for CI/CD
  gone check --all                   # Show all results including alive
  gone check --dead                  # Show only dead links and errors
  gone check --concurrency=100       # Use 100 concurrent workers
  gone check --stats                 # Show performance statistics

Note: --format and --output are mutually exclusive.

Supported file types: md, json, yaml, toml, xml

Ignore patterns:
  gone check --ignore-domain=localhost,example.com
  gone check --ignore-pattern="*.local/*"
  gone check --ignore-regex=".*\\.test$"
  gone check --show-ignored          # Show which URLs were ignored

Config file (.gonerc.yaml):
  ignore:
    domains: [localhost, example.com]
    patterns: ["*.local/*"]
    regex: [".*\\.test$"]`,
	Args: cobra.MaximumNArgs(1),
	Run:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Output options
	checkCmd.Flags().StringVarP(&outputFormat, "format", "f", "",
		"Output format for stdout: json, yaml, xml, junit, markdown")
	checkCmd.Flags().StringVarP(&outputFile, "output", "o", "",
		"Write report to file (format inferred from extension: .json, .yaml, .xml, .junit.xml, .md)")

	// File type options
	checkCmd.Flags().StringSliceVarP(&fileTypes, "types", "T", []string{"md"},
		"File types to scan (comma-separated): md, json, yaml, toml, xml")
	checkCmd.Flags().BoolVar(&strictMode, "strict", false,
		"Fail on malformed files instead of skipping them")

	// Filter flags
	checkCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all results (alive, warnings, dead)")
	checkCmd.Flags().BoolVar(&showAlive, "alive", false, "Show only alive links")
	checkCmd.Flags().BoolVarP(&showWarnings, "warnings", "w", false, "Show only warnings (redirects, blocked)")
	checkCmd.Flags().BoolVarP(&showDead, "dead", "d", false, "Show only dead links and errors")

	// Performance options
	checkCmd.Flags().IntVarP(&concurrency, "concurrency", "c", checker.DefaultConcurrency,
		"Number of concurrent workers")
	checkCmd.Flags().IntVarP(&timeout, "timeout", "t", int(checker.DefaultTimeout.Seconds()),
		"Timeout per request in seconds")
	checkCmd.Flags().IntVarP(&retries, "retries", "r", checker.DefaultMaxRetries,
		"Number of retries for failed requests")

	// Stats flag
	checkCmd.Flags().BoolVar(&showStats, "stats", false,
		"Show detailed performance statistics")

	// Ignore options
	checkCmd.Flags().StringSliceVar(&ignoreDomains, "ignore-domain", nil,
		"Domains to ignore, includes subdomains (can be repeated or comma-separated)")
	checkCmd.Flags().StringSliceVar(&ignorePatterns, "ignore-pattern", nil,
		"Glob patterns to ignore (can be repeated)")
	checkCmd.Flags().StringSliceVar(&ignoreRegex, "ignore-regex", nil,
		"Regex patterns to ignore (can be repeated)")
	checkCmd.Flags().BoolVar(&showIgnored, "show-ignored", false,
		"Show which URLs were ignored and why")
	checkCmd.Flags().BoolVar(&noConfig, "no-config", false,
		"Skip loading .gonerc.yaml config file")
}

// runCheck is the main entry point for the check command.
// It orchestrates the entire link checking workflow.
func runCheck(_ *cobra.Command, args []string) {
	perf := stats.New()
	exitOnError(validateCheckFlags(), "Invalid flags")

	path := getPathArg(args)
	useStructuredOutput := outputFormat != ""

	// Phase 1: Scan for files
	files := scanFiles(path, perf, useStructuredOutput)

	// Phase 2: Parse links from files
	links, urlFilter, done := parseAndFilterLinks(files, perf, useStructuredOutput)
	if done {
		return
	}

	// Phase 3: Check URLs
	results, summary := checkLinks(links, perf)

	// Phase 4: Output results
	routeOutput(files, results, summary, urlFilter, perf, useStructuredOutput)

	if summary.HasDeadLinks() {
		os.Exit(1)
	}
}

// exitOnError prints an error message and exits if err is not nil.
func exitOnError(err error, message string) {
	if err != nil {
		if message != "" {
			fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
}

// getPathArg returns the path argument or "." as default.
func getPathArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

// scanFiles scans for files with the specified types and returns the list.
func scanFiles(path string, perf *stats.Stats, useStructuredOutput bool) []string {
	perf.StartScan()

	// Validate file types
	if err := validateFileTypes(fileTypes); err != nil {
		exitOnError(err, "Invalid file types")
	}

	files, err := scanner.FindFilesByTypes(path, fileTypes)
	exitOnError(err, "Error scanning directory")
	perf.EndScan(len(files))

	if !useStructuredOutput {
		typeStr := strings.Join(fileTypes, ", ")
		fmt.Printf("Found %d file(s) of type(s): %s\n", len(files), typeStr)
	}
	return files
}

// validateFileTypes checks if all specified file types are supported.
func validateFileTypes(types []string) error {
	supportedTypes := parser.SupportedFileTypes()
	supported := make(map[string]bool, len(supportedTypes))
	for _, t := range supportedTypes {
		supported[t] = true
	}

	for _, t := range types {
		if !supported[strings.ToLower(t)] {
			return fmt.Errorf("unsupported file type: %s (supported: %s)",
				t, strings.Join(supportedTypes, ", "))
		}
	}
	return nil
}

// parseAndFilterLinks extracts links from files and applies filters.
// Returns the links, filter, and whether processing should stop (done=true).
func parseAndFilterLinks(
	files []string, perf *stats.Stats, useStructuredOutput bool,
) ([]checker.Link, *filter.Filter, bool) {
	perf.StartParse()
	parserLinks, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, strictMode)
	exitOnError(err, "Error parsing files")

	if len(parserLinks) == 0 {
		perf.EndParse(0, 0, 0, 0)
		handleEmptyLinksWithStats(files, useStructuredOutput, perf)
		return nil, nil, true
	}

	urlFilter, err := CreateFilter(FilterOptions{
		Domains:  ignoreDomains,
		Patterns: ignorePatterns,
		Regex:    ignoreRegex,
		NoConfig: noConfig,
	})
	exitOnError(err, "Error creating filter")

	links := FilterParserLinks(parserLinks, urlFilter)
	ignoredCount := getIgnoredCount(urlFilter)
	uniqueURLs := CountUniqueURLs(links)
	duplicates := len(links) - uniqueURLs

	perf.EndParse(len(parserLinks), uniqueURLs, duplicates, ignoredCount)

	if !useStructuredOutput {
		printProgressMessage(len(parserLinks), len(links), uniqueURLs, duplicates, ignoredCount)
	}

	if len(links) == 0 {
		handleAllFilteredWithStats(files, useStructuredOutput, urlFilter, perf)
		return nil, urlFilter, true
	}

	return links, urlFilter, false
}

// getIgnoredCount returns the ignored count from filter, or 0 if filter is nil.
func getIgnoredCount(urlFilter *filter.Filter) int {
	if urlFilter != nil {
		return urlFilter.IgnoredCount()
	}
	return 0
}

// checkLinks checks all links and returns results with summary.
func checkLinks(links []checker.Link, perf *stats.Stats) ([]checker.Result, checker.Summary) {
	perf.StartCheck()

	opts := checker.DefaultOptions().
		WithConcurrency(concurrency).
		WithTimeout(time.Duration(timeout) * time.Second).
		WithMaxRetries(retries)

	c := checker.New(opts)
	results := c.CheckAll(links)
	summary := checker.Summarize(results)

	perf.EndCheck()
	return results, summary
}

// routeOutput handles output based on format flags.
func routeOutput(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats, useStructuredOutput bool,
) {
	switch {
	case useStructuredOutput:
		handleStructuredOutputWithStats(files, results, summary, urlFilter, perf)
	case outputFile != "":
		handleFileOutputWithStats(files, results, summary, urlFilter, perf)
	default:
		outputText(results, summary, urlFilter)
		if showStats {
			fmt.Print(perf.String())
		}
	}
}

// validateCheckFlags checks for invalid flag combinations.
func validateCheckFlags() error {
	// Validate mutually exclusive flags
	if outputFormat != "" && outputFile != "" {
		return fmt.Errorf("--format and --output are mutually exclusive; " +
			"use --format for stdout output, or --output for file output")
	}

	// Validate format if specified
	if outputFormat != "" && !output.IsValidFormat(outputFormat) {
		return fmt.Errorf("invalid format %q; valid formats: %s",
			outputFormat, strings.Join(output.ValidFormats(), ", "))
	}

	return nil
}

// handleEmptyLinksWithStats handles the case when no links are found in the files.
func handleEmptyLinksWithStats(files []string, useStructuredOutput bool, perf *stats.Stats) {
	switch {
	case useStructuredOutput:
		handleStructuredOutputWithStats(files, nil, checker.Summary{}, nil, perf)
	case outputFile != "":
		handleFileOutputWithStats(files, nil, checker.Summary{}, nil, perf)
	default:
		fmt.Println("No links found.")
		if showStats {
			fmt.Print(perf.String())
		}
	}
}

// handleAllFilteredWithStats handles the case when all links were filtered out.
func handleAllFilteredWithStats(files []string, useStructuredOutput bool, urlFilter *filter.Filter, perf *stats.Stats) {
	switch {
	case useStructuredOutput:
		handleStructuredOutputWithStats(files, nil, checker.Summary{}, urlFilter, perf)
	case outputFile != "":
		handleFileOutputWithStats(files, nil, checker.Summary{}, urlFilter, perf)
	default:
		fmt.Println("\nAll links were ignored by filter rules.")
		if showIgnored && urlFilter != nil {
			printIgnoredURLs(urlFilter)
		}
		if showStats {
			fmt.Print(perf.String())
		}
	}
}

// handleStructuredOutputWithStats outputs to stdout with optional stats.
func handleStructuredOutputWithStats(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats,
) {
	report := buildReportWithStats(files, results, summary, urlFilter, perf)

	data, err := output.FormatReport(report, output.Format(outputFormat))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(data))
}

// handleFileOutputWithStats writes to file with optional stats.
func handleFileOutputWithStats(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats,
) {
	report := buildReportWithStats(files, results, summary, urlFilter, perf)

	if err := output.WriteToFile(report, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote report to %s\n", outputFile)

	// Also print summary to stdout
	fmt.Printf("\nSummary: %d alive | %d warnings | %d dead | %d duplicates",
		summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors, summary.Duplicates)
	if urlFilter != nil && urlFilter.IgnoredCount() > 0 {
		fmt.Printf(" | %d ignored", urlFilter.IgnoredCount())
	}
	fmt.Println()

	if showStats {
		fmt.Print(perf.String())
	}
}

// buildReportWithStats creates an output.Report with optional stats.
func buildReportWithStats(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats,
) *output.Report {
	report := buildReport(files, results, summary, urlFilter)

	// Add stats if requested
	if showStats && perf != nil {
		report.Stats = perf.ToJSON()
	}

	return report
}
