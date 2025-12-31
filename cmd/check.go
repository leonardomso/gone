package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gone/internal/checker"
	"gone/internal/config"
	"gone/internal/filter"
	"gone/internal/parser"
	"gone/internal/scanner"

	"github.com/spf13/cobra"
)

// jsonOutput represents the JSON structure for output.
type jsonOutput struct {
	TotalFiles int           `json:"total_files"`
	TotalLinks int           `json:"total_links"`
	UniqueURLs int           `json:"unique_urls"`
	Summary    jsonSummary   `json:"summary"`
	Results    []jsonResult  `json:"results"`
	Ignored    []jsonIgnored `json:"ignored,omitempty"`
}

type jsonSummary struct {
	Alive      int `json:"alive"`
	Redirects  int `json:"redirects"`
	Blocked    int `json:"blocked"`
	Dead       int `json:"dead"`
	Errors     int `json:"errors"`
	Duplicates int `json:"duplicates"`
	Ignored    int `json:"ignored,omitempty"`
}

type jsonIgnored struct {
	URL    string `json:"url"`
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Reason string `json:"reason"`
	Rule   string `json:"rule"`
}

type jsonResult struct {
	URL           string         `json:"url"`
	FilePath      string         `json:"file_path"`
	Line          int            `json:"line,omitempty"`
	Text          string         `json:"text,omitempty"`
	StatusCode    int            `json:"status_code"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
	RedirectChain []jsonRedirect `json:"redirect_chain,omitempty"`
	FinalURL      string         `json:"final_url,omitempty"`
	FinalStatus   int            `json:"final_status,omitempty"`
	DuplicateOf   string         `json:"duplicate_of,omitempty"`
}

type jsonRedirect struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

// Flag variables.
var (
	format       string
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
  gone check --format=json           # Output as JSON
  gone check --all                   # Show all results including alive
  gone check --alive                 # Show only alive links
  gone check --warnings              # Show only warnings (redirects, blocked)
  gone check --dead                  # Show only dead links and errors
  gone check --concurrency=20        # Use 20 concurrent workers
  gone check --timeout=30            # 30 second timeout per request

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

	// Output format
	checkCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")

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
	// Determine the path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	isJSON := format == "json"

	// Find all markdown files
	files, err := scanner.FindMarkdownFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if !isJSON {
		fmt.Printf("Found %d markdown file(s)\n", len(files))
	}

	// Extract all URLs from the files
	parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
		os.Exit(1)
	}

	if len(parserLinks) == 0 {
		if isJSON {
			outputJSON(files, nil, checker.Summary{}, nil)
		} else {
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

	if !isJSON {
		printProgressMessage(len(parserLinks), len(links), uniqueURLs, duplicates, ignoredCount)
	}

	// Handle case where all links were filtered out
	if len(links) == 0 {
		if isJSON {
			outputJSON(files, nil, checker.Summary{}, urlFilter)
		} else {
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

	// Output based on format flag
	if isJSON {
		outputJSON(files, results, summary, urlFilter)
	} else {
		outputText(results, summary, urlFilter)
	}

	// Exit with code 1 if dead links or errors found (not for warnings)
	if summary.HasDeadLinks() {
		os.Exit(1)
	}
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

// outputJSON prints results as JSON.
func outputJSON(files []string, results []checker.Result, summary checker.Summary, urlFilter *filter.Filter) {
	ignoredCount := 0
	if urlFilter != nil {
		ignoredCount = urlFilter.IgnoredCount()
	}

	output := jsonOutput{
		TotalFiles: len(files),
		TotalLinks: summary.Total + ignoredCount,
		UniqueURLs: summary.UniqueURLs,
		Summary: jsonSummary{
			Alive:      summary.Alive,
			Redirects:  summary.Redirects,
			Blocked:    summary.Blocked,
			Dead:       summary.Dead,
			Errors:     summary.Errors,
			Duplicates: summary.Duplicates,
			Ignored:    ignoredCount,
		},
		Results: []jsonResult{},
	}

	// Filter results based on flags
	filtered := filterResults(results)

	for _, r := range filtered {
		jr := jsonResult{
			URL:        r.Link.URL,
			FilePath:   r.Link.FilePath,
			Line:       r.Link.Line,
			Text:       r.Link.Text,
			StatusCode: r.StatusCode,
			Status:     r.Status.String(),
			Error:      r.Error,
		}

		// Add redirect chain if present
		if len(r.RedirectChain) > 0 {
			jr.RedirectChain = make([]jsonRedirect, len(r.RedirectChain))
			for i, red := range r.RedirectChain {
				jr.RedirectChain[i] = jsonRedirect{
					URL:        red.URL,
					StatusCode: red.StatusCode,
				}
			}
			jr.FinalURL = r.FinalURL
			jr.FinalStatus = r.FinalStatus
		}

		// Add duplicate reference if present
		if r.DuplicateOf != nil {
			jr.DuplicateOf = r.DuplicateOf.Link.URL
		}

		output.Results = append(output.Results, jr)
	}

	// Add ignored URLs if --show-ignored is set
	if showIgnored && urlFilter != nil {
		for _, ig := range urlFilter.IgnoredURLs() {
			output.Ignored = append(output.Ignored, jsonIgnored{
				URL:    ig.URL,
				File:   ig.File,
				Line:   ig.Line,
				Reason: ig.Type,
				Rule:   ig.Rule,
			})
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
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
