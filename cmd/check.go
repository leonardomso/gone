package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gone/internal/checker"
	"gone/internal/config"
	"gone/internal/filter"
	"gone/internal/output"
	"gone/internal/parser"
	"gone/internal/scanner"

	"github.com/spf13/cobra"
)

// Flag variables.
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
  gone check --concurrency=20        # Use 20 concurrent workers

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
	checkCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers")
	checkCmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "Timeout per request in seconds")
	checkCmd.Flags().IntVarP(&retries, "retries", "r", 2, "Number of retries for failed requests")

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

func runCheck(_ *cobra.Command, args []string) {
	// Validate mutually exclusive flags
	if outputFormat != "" && outputFile != "" {
		fmt.Fprintf(os.Stderr, "Error: --format and --output are mutually exclusive\n")
		fmt.Fprintf(os.Stderr, "Use --format for stdout output, or --output for file output\n")
		os.Exit(1)
	}

	// Validate format if specified
	if outputFormat != "" && !output.IsValidFormat(outputFormat) {
		fmt.Fprintf(os.Stderr, "Error: invalid format %q\n", outputFormat)
		fmt.Fprintf(os.Stderr, "Valid formats: %s\n", strings.Join(output.ValidFormats(), ", "))
		os.Exit(1)
	}

	// Determine the path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Determine if we should suppress progress output
	useStructuredOutput := outputFormat != ""

	// Find all markdown files
	files, err := scanner.FindMarkdownFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if !useStructuredOutput {
		fmt.Printf("Found %d markdown file(s)\n", len(files))
	}

	// Extract all URLs from the files
	parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
		os.Exit(1)
	}

	if len(parserLinks) == 0 {
		switch {
		case useStructuredOutput:
			handleStructuredOutput(files, nil, checker.Summary{}, nil)
		case outputFile != "":
			handleFileOutput(files, nil, checker.Summary{}, nil)
		default:
			fmt.Println("No links found.")
		}
		return
	}

	// Load and create filter
	urlFilter, err := createFilter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filter: %v\n", err)
		os.Exit(1)
	}

	// Convert parser.Link to checker.Link, applying filter
	links := make([]checker.Link, 0, len(parserLinks))
	for _, pl := range parserLinks {
		// Check if URL should be ignored
		if urlFilter != nil && urlFilter.ShouldIgnore(pl.URL, pl.FilePath, pl.Line) {
			continue
		}
		links = append(links, checker.Link{
			URL:      pl.URL,
			FilePath: pl.FilePath,
			Line:     pl.Line,
			Text:     pl.Text,
		})
	}

	ignoredCount := 0
	if urlFilter != nil {
		ignoredCount = urlFilter.IgnoredCount()
	}

	// Count unique URLs for progress display
	uniqueURLs := countUniqueURLs(links)
	duplicates := len(links) - uniqueURLs

	if !useStructuredOutput {
		printProgressMessage(len(parserLinks), len(links), uniqueURLs, duplicates, ignoredCount)
	}

	// Handle case where all links were filtered out
	if len(links) == 0 {
		switch {
		case useStructuredOutput:
			handleStructuredOutput(files, nil, checker.Summary{}, urlFilter)
		case outputFile != "":
			handleFileOutput(files, nil, checker.Summary{}, urlFilter)
		default:
			fmt.Println("\nAll links were ignored by filter rules.")
			if showIgnored && urlFilter != nil {
				printIgnoredURLs(urlFilter)
			}
		}
		return
	}

	// Create checker with configured options
	opts := checker.DefaultOptions().
		WithConcurrency(concurrency).
		WithTimeout(time.Duration(timeout) * time.Second).
		WithMaxRetries(retries)

	c := checker.New(opts)

	// Check all links
	results := c.CheckAll(links)
	summary := checker.Summarize(results)

	// Handle output
	switch {
	case useStructuredOutput:
		handleStructuredOutput(files, results, summary, urlFilter)
	case outputFile != "":
		handleFileOutput(files, results, summary, urlFilter)
	default:
		outputText(results, summary, urlFilter)
	}

	// Exit with code 1 if dead links or errors found (not for warnings)
	if summary.HasDeadLinks() {
		os.Exit(1)
	}
}

// handleStructuredOutput outputs to stdout in the specified format.
func handleStructuredOutput(
	files []string, results []checker.Result, summary checker.Summary, urlFilter *filter.Filter,
) {
	report := buildReport(files, results, summary, urlFilter)

	data, err := output.FormatReport(report, output.Format(outputFormat))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(data))
}

// handleFileOutput writes the report to a file.
func handleFileOutput(files []string, results []checker.Result, summary checker.Summary, urlFilter *filter.Filter) {
	report := buildReport(files, results, summary, urlFilter)

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
}

// buildReport creates an output.Report from check results.
func buildReport(
	files []string, results []checker.Result, summary checker.Summary, urlFilter *filter.Filter,
) *output.Report {
	report := &output.Report{
		GeneratedAt: time.Now(),
		Files:       files,
		TotalLinks:  summary.Total,
		UniqueURLs:  summary.UniqueURLs,
		Summary:     summary,
		Results:     filterResults(results),
	}

	// Add ignored URLs if filter is present and --show-ignored is set
	if showIgnored && urlFilter != nil {
		for _, ig := range urlFilter.IgnoredURLs() {
			report.Ignored = append(report.Ignored, output.IgnoredURL{
				URL:    ig.URL,
				File:   ig.File,
				Line:   ig.Line,
				Reason: ig.Type,
				Rule:   ig.Rule,
			})
		}
	}

	// Adjust total links to include ignored
	if urlFilter != nil {
		report.TotalLinks += urlFilter.IgnoredCount()
	}

	return report
}

// createFilter builds a filter from config file and CLI flags.
func createFilter() (*filter.Filter, error) {
	var cfg *config.Config

	// Load config file unless --no-config is set
	if !noConfig {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	} else {
		cfg = &config.Config{}
	}

	// Merge CLI flags (additive)
	cfg.Ignore.Domains = append(cfg.Ignore.Domains, ignoreDomains...)
	cfg.Ignore.Patterns = append(cfg.Ignore.Patterns, ignorePatterns...)
	cfg.Ignore.Regex = append(cfg.Ignore.Regex, ignoreRegex...)

	// If no ignore rules, return nil (no filtering)
	if cfg.IsEmpty() {
		return nil, nil
	}

	// Create filter
	return filter.New(filter.Config{
		Domains:       cfg.Ignore.Domains,
		GlobPatterns:  cfg.Ignore.Patterns,
		RegexPatterns: cfg.Ignore.Regex,
	})
}

// printProgressMessage shows the scanning progress with ignore info.
func printProgressMessage(total, afterFilter, unique, duplicates, ignored int) {
	var parts []string

	if duplicates > 0 {
		parts = append(parts, fmt.Sprintf("%d duplicates", duplicates))
	}
	if ignored > 0 {
		parts = append(parts, fmt.Sprintf("%d ignored", ignored))
	}

	if len(parts) > 0 {
		fmt.Printf("Found %d link(s), checking %d unique URLs (%s)...\n",
			total, unique, strings.Join(parts, ", "))
	} else {
		fmt.Printf("Found %d link(s), checking...\n", afterFilter)
	}
}

// countUniqueURLs returns the number of unique URLs in the slice.
func countUniqueURLs(links []checker.Link) int {
	seen := map[string]bool{}
	for _, l := range links {
		seen[l.URL] = true
	}
	return len(seen)
}

// filterResults returns results based on the filter flags.
func filterResults(results []checker.Result) []checker.Result {
	// If specific filter is set, use it
	if showAlive {
		return checker.FilterAlive(results)
	}
	if showWarnings {
		return checker.FilterWarnings(results)
	}
	if showDead {
		return checker.FilterDead(results)
	}
	if showAll {
		return results
	}

	// Default: show warnings + dead + duplicates (non-alive)
	var filtered []checker.Result
	for _, r := range results {
		if !r.IsAlive() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// outputText prints results as human-readable text.
func outputText(results []checker.Result, summary checker.Summary, urlFilter *filter.Filter) {
	ignoredCount := 0
	if urlFilter != nil {
		ignoredCount = urlFilter.IgnoredCount()
	}

	fmt.Println()
	if ignoredCount > 0 {
		fmt.Printf("Summary: %d alive | %d warnings | %d dead | %d duplicates | %d ignored\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors, summary.Duplicates, ignoredCount)
	} else {
		fmt.Printf("Summary: %d alive | %d warnings | %d dead | %d duplicates\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors, summary.Duplicates)
	}

	filtered := filterResults(results)

	if len(filtered) == 0 {
		switch {
		case showAlive && summary.Alive == 0:
			fmt.Println("No alive links found.")
		case showWarnings && summary.WarningsCount() == 0:
			fmt.Println("No warnings found.")
		case showDead && !summary.HasDeadLinks():
			fmt.Println("No dead links found.")
		default:
			fmt.Println("All links are alive!")
		}

		// Show ignored URLs if requested
		if showIgnored && urlFilter != nil {
			printIgnoredURLs(urlFilter)
		}
		return
	}

	// Group results by status for nicer output
	if showAll || (!showAlive && !showWarnings && !showDead) {
		// Show in sections
		printSection("Warnings", filterWarningsFromSlice(filtered), printWarningResult)
		printSection("Dead Links", filterDeadFromSlice(filtered), printDeadResult)
		printSection("Duplicates", filterDuplicatesFromSlice(filtered), printDuplicateResult)

		if showAll {
			printSection("Alive", filterAliveFromSlice(filtered), printAliveResult)
		}
	} else {
		// Show flat list for specific filter
		for _, r := range filtered {
			printResult(r)
		}
	}

	// Show ignored URLs if requested
	if showIgnored && urlFilter != nil {
		printIgnoredURLs(urlFilter)
	}
}

// printIgnoredURLs displays the list of ignored URLs.
func printIgnoredURLs(urlFilter *filter.Filter) {
	ignored := urlFilter.IgnoredURLs()
	if len(ignored) == 0 {
		return
	}

	fmt.Printf("\n=== Ignored URLs (%d) ===\n\n", len(ignored))
	for _, ig := range ignored {
		fmt.Printf("  [IGNORED] %s\n", ig.URL)
		fmt.Printf("            File: %s", ig.File)
		if ig.Line > 0 {
			fmt.Printf(":%d", ig.Line)
		}
		fmt.Println()
		fmt.Printf("            Reason: %s %q\n\n", ig.Type, ig.Rule)
	}
}

func printSection(title string, results []checker.Result, printer func(checker.Result)) {
	if len(results) == 0 {
		return
	}
	fmt.Printf("=== %s (%d) ===\n\n", title, len(results))
	for _, r := range results {
		printer(r)
	}
	fmt.Println()
}

func printResult(r checker.Result) {
	switch r.Status {
	case checker.StatusAlive:
		printAliveResult(r)
	case checker.StatusRedirect, checker.StatusBlocked:
		printWarningResult(r)
	case checker.StatusDead, checker.StatusError:
		printDeadResult(r)
	case checker.StatusDuplicate:
		printDuplicateResult(r)
	}
}

func printAliveResult(r checker.Result) {
	fmt.Printf("  [%d] %s\n", r.StatusCode, r.Link.URL)
	if text := truncateText(r.Link.Text); text != "" {
		fmt.Printf("       Text: %q\n", text)
	}
	fmt.Printf("       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Printf(":%d", r.Link.Line)
	}
	fmt.Println()
	fmt.Println()
}

func printWarningResult(r checker.Result) {
	fmt.Printf("  %s %s\n", r.StatusDisplay(), r.Link.URL)

	if text := truncateText(r.Link.Text); text != "" {
		fmt.Printf("       Text: %q\n", text)
	}

	if r.Status == checker.StatusRedirect && len(r.RedirectChain) > 0 {
		fmt.Printf("       Chain: %s\n", formatRedirectChain(r))
		fmt.Printf("       Final: %s\n", r.FinalURL)
	}

	fmt.Printf("       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Printf(":%d", r.Link.Line)
	}
	fmt.Println()
	fmt.Printf("       Note: %s\n\n", r.Status.Description())
}

func printDeadResult(r checker.Result) {
	fmt.Printf("  %s %s\n", r.StatusDisplay(), r.Link.URL)
	if text := truncateText(r.Link.Text); text != "" {
		fmt.Printf("       Text: %q\n", text)
	}
	fmt.Printf("       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Printf(":%d", r.Link.Line)
	}
	fmt.Println()

	if r.Error != "" {
		fmt.Printf("       Error: %s\n", r.Error)
	}
	fmt.Println()
}

func printDuplicateResult(r checker.Result) {
	fmt.Printf("  [DUPLICATE] %s\n", r.Link.URL)
	if text := truncateText(r.Link.Text); text != "" {
		fmt.Printf("              Text: %q\n", text)
	}
	fmt.Printf("              File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Printf(":%d", r.Link.Line)
	}
	fmt.Println()

	if r.DuplicateOf != nil {
		fmt.Printf("              Same as: %s", r.DuplicateOf.Link.FilePath)
		if r.DuplicateOf.Link.Line > 0 {
			fmt.Printf(":%d", r.DuplicateOf.Link.Line)
		}
		fmt.Printf(" → Status: %s\n", r.DuplicateOf.Status.Label())
	}
	fmt.Println()
}

func formatRedirectChain(r checker.Result) string {
	parts := make([]string, 0, len(r.RedirectChain)+1)
	for _, red := range r.RedirectChain {
		parts = append(parts, fmt.Sprintf("%d", red.StatusCode))
	}
	parts = append(parts, fmt.Sprintf("%d", r.FinalStatus))
	return strings.Join(parts, " → ")
}

// truncateText shortens text to 50 characters, adding "..." if truncated.
// Returns empty string if input is empty or only whitespace.
func truncateText(text string) string {
	const maxLen = 50
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// Helper filter functions for slices.
func filterWarningsFromSlice(results []checker.Result) []checker.Result {
	var filtered []checker.Result
	for _, r := range results {
		if r.IsWarning() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterDeadFromSlice(results []checker.Result) []checker.Result {
	var filtered []checker.Result
	for _, r := range results {
		if r.IsDead() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterDuplicatesFromSlice(results []checker.Result) []checker.Result {
	var filtered []checker.Result
	for _, r := range results {
		if r.IsDuplicate() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterAliveFromSlice(results []checker.Result) []checker.Result {
	var filtered []checker.Result
	for _, r := range results {
		if r.IsAlive() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
