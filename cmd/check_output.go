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
		printSection("Warnings", FilterResultsWarnings(filtered), printWarningResult)
		printSection("Dead Links", FilterResultsDead(filtered), printDeadResult)
		printSection("Duplicates", FilterResultsDuplicates(filtered), printDuplicateResult)

		if showAll {
			printSection("Alive", FilterResultsAlive(filtered), printAliveResult)
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
