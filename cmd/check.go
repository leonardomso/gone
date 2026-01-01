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
	Short: "Scan markdown files for dead links",
	Long: `Scan a directory for markdown files and check all HTTP/HTTPS links.

If no path is provided, scans the current directory.

By default, shows warnings (redirects, blocked) and dead links.
Use flags to filter what's displayed.

Exit codes:
  0 - All links are alive or only have warnings
  1 - Dead links or errors found

Examples:
  gone check                         # Scan current directory
  gone check ./docs                  # Scan specific directory  
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
	// Initialize stats tracking
	perf := stats.New()

	// Validate mutually exclusive flags
	if err := validateCheckFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Determine the path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Determine if we should suppress progress output
	useStructuredOutput := outputFormat != ""

	// Phase 1: Scan for files
	perf.StartScan()
	files, err := scanner.FindMarkdownFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}
	perf.EndScan(len(files))

	if !useStructuredOutput {
		fmt.Printf("Found %d markdown file(s)\n", len(files))
	}

	// Phase 2: Parse links from files
	perf.StartParse()
	parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
		os.Exit(1)
	}

	if len(parserLinks) == 0 {
		perf.EndParse(0, 0, 0, 0)
		handleEmptyLinksWithStats(files, useStructuredOutput, perf)
		return
	}

	// Load and create filter
	urlFilter, err := CreateFilter(FilterOptions{
		Domains:  ignoreDomains,
		Patterns: ignorePatterns,
		Regex:    ignoreRegex,
		NoConfig: noConfig,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filter: %v\n", err)
		os.Exit(1)
	}

	// Convert parser.Link to checker.Link, applying filter
	links := FilterParserLinks(parserLinks, urlFilter)

	ignoredCount := 0
	if urlFilter != nil {
		ignoredCount = urlFilter.IgnoredCount()
	}

	// Count unique URLs for progress display
	uniqueURLs := CountUniqueURLs(links)
	duplicates := len(links) - uniqueURLs

	perf.EndParse(len(parserLinks), uniqueURLs, duplicates, ignoredCount)

	if !useStructuredOutput {
		printProgressMessage(len(parserLinks), len(links), uniqueURLs, duplicates, ignoredCount)
	}

	// Handle case where all links were filtered out
	if len(links) == 0 {
		handleAllFilteredWithStats(files, useStructuredOutput, urlFilter, perf)
		return
	}

	// Phase 3: Check URLs
	perf.StartCheck()

	// Create checker with configured options
	opts := checker.DefaultOptions().
		WithConcurrency(concurrency).
		WithTimeout(time.Duration(timeout) * time.Second).
		WithMaxRetries(retries)

	c := checker.New(opts)

	// Check all links
	results := c.CheckAll(links)
	summary := checker.Summarize(results)

	perf.EndCheck()

	// Handle output
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

	// Exit with code 1 if dead links or errors found (not for warnings)
	if summary.HasDeadLinks() {
		os.Exit(1)
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
