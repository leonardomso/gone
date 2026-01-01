package cmd

import (
	"fmt"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/output"
)

// buildReport creates an output.Report from check results.
// This consolidates all data needed for formatted output.
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

// filterResults returns results based on the filter flags.
// This determines which results are included in the output based on CLI flags.
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
	// Pre-allocate with estimated capacity
	filtered := make([]checker.Result, 0, len(results)/4)
	for _, r := range results {
		if !r.IsAlive() {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// outputText prints results as human-readable text to stdout.
// This is the default output mode when no format flag is specified.
func outputText(results []checker.Result, summary checker.Summary, urlFilter *filter.Filter) {
	ignoredCount := getFilterIgnoredCount(urlFilter)
	printSummaryLine(summary, ignoredCount)

	filtered := filterResults(results)

	if len(filtered) == 0 {
		fmt.Println(getEmptyResultsMessage(summary))
		maybeShowIgnored(urlFilter)
		return
	}

	if shouldGroupResults() {
		outputGroupedResults(filtered)
	} else {
		outputFlatResults(filtered)
	}

	maybeShowIgnored(urlFilter)
}

// getFilterIgnoredCount returns the ignored count from filter, or 0 if nil.
func getFilterIgnoredCount(urlFilter *filter.Filter) int {
	if urlFilter != nil {
		return urlFilter.IgnoredCount()
	}
	return 0
}

// printSummaryLine prints the summary statistics line.
func printSummaryLine(summary checker.Summary, ignoredCount int) {
	fmt.Println()
	if ignoredCount > 0 {
		fmt.Printf("Summary: %d alive | %d warnings | %d dead | %d duplicates | %d ignored\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors,
			summary.Duplicates, ignoredCount)
	} else {
		fmt.Printf("Summary: %d alive | %d warnings | %d dead | %d duplicates\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors,
			summary.Duplicates)
	}
}

// getEmptyResultsMessage returns the appropriate message when no results match filters.
func getEmptyResultsMessage(summary checker.Summary) string {
	switch {
	case showAlive && summary.Alive == 0:
		return "No alive links found."
	case showWarnings && summary.WarningsCount() == 0:
		return "No warnings found."
	case showDead && !summary.HasDeadLinks():
		return "No dead links found."
	default:
		return "All links are alive!"
	}
}

// shouldGroupResults returns true if results should be displayed in grouped sections.
func shouldGroupResults() bool {
	return showAll || (!showAlive && !showWarnings && !showDead)
}

// outputGroupedResults prints results grouped by status sections.
func outputGroupedResults(filtered []checker.Result) {
	printSection("Warnings", FilterResultsWarnings(filtered), printWarningResult)
	printSection("Dead Links", FilterResultsDead(filtered), printDeadResult)
	printSection("Duplicates", FilterResultsDuplicates(filtered), printDuplicateResult)

	if showAll {
		printSection("Alive", FilterResultsAlive(filtered), printAliveResult)
	}
}

// outputFlatResults prints results as a flat list.
func outputFlatResults(filtered []checker.Result) {
	for _, r := range filtered {
		printResult(r)
	}
}

// maybeShowIgnored shows ignored URLs if the flag is set and filter exists.
func maybeShowIgnored(urlFilter *filter.Filter) {
	if showIgnored && urlFilter != nil {
		printIgnoredURLs(urlFilter)
	}
}
