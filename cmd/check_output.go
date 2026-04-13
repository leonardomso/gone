package cmd

import (
	"io"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/output"
	"github.com/samber/lo"
)

type checkRenderOptions struct {
	ShowAlive    bool
	ShowWarnings bool
	ShowDead     bool
	ShowAll      bool
	ShowIgnored  bool
}

// buildReport creates an output.Report from check results.
// This consolidates all data needed for formatted output.
func buildReport(
	now time.Time,
	files []string,
	results []checker.Result,
	summary checker.Summary,
	urlFilter *filter.Filter,
	opts checkRenderOptions,
) *output.Report {
	report := &output.Report{
		GeneratedAt: now,
		Files:       files,
		TotalLinks:  summary.Total,
		UniqueURLs:  summary.UniqueURLs,
		Summary:     summary,
		Results:     filterResults(results, opts),
	}

	// Add ignored URLs if filter is present and --show-ignored is set
	if opts.ShowIgnored && urlFilter != nil {
		report.Ignored = lo.Map(urlFilter.IgnoredURLs(), func(ig filter.IgnoreReason, _ int) output.IgnoredURL {
			return output.IgnoredURL{
				URL:    ig.URL,
				File:   ig.File,
				Line:   ig.Line,
				Reason: ig.Type,
				Rule:   ig.Rule,
			}
		})
	}

	// Adjust total links to include ignored
	if urlFilter != nil {
		report.TotalLinks += urlFilter.IgnoredCount()
	}

	return report
}

// filterResults returns results based on the filter flags.
// This determines which results are included in the output based on CLI flags.
func filterResults(results []checker.Result, opts checkRenderOptions) []checker.Result {
	// If specific filter is set, use it
	if opts.ShowAlive {
		return checker.FilterAlive(results)
	}
	if opts.ShowWarnings {
		return checker.FilterWarnings(results)
	}
	if opts.ShowDead {
		return checker.FilterDead(results)
	}
	if opts.ShowAll {
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
func outputText(
	w io.Writer,
	results []checker.Result,
	summary checker.Summary,
	urlFilter *filter.Filter,
	opts checkRenderOptions,
) {
	ignoredCount := getFilterIgnoredCount(urlFilter)
	printSummaryLine(w, summary, ignoredCount)

	filtered := filterResults(results, opts)

	if len(filtered) == 0 {
		writeln(w, getEmptyResultsMessage(summary, opts))
		maybeShowIgnored(w, urlFilter, opts)
		return
	}

	if shouldGroupResults(opts) {
		outputGroupedResults(w, filtered, opts)
	} else {
		outputFlatResults(w, filtered)
	}

	maybeShowIgnored(w, urlFilter, opts)
}

// getFilterIgnoredCount returns the ignored count from filter, or 0 if nil.
func getFilterIgnoredCount(urlFilter *filter.Filter) int {
	if urlFilter != nil {
		return urlFilter.IgnoredCount()
	}
	return 0
}

// printSummaryLine prints the summary statistics line.
func printSummaryLine(w io.Writer, summary checker.Summary, ignoredCount int) {
	writeln(w)
	if ignoredCount > 0 {
		writef(w, "Summary: %d alive | %d warnings | %d dead | %d duplicates | %d ignored\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors,
			summary.Duplicates, ignoredCount)
	} else {
		writef(w, "Summary: %d alive | %d warnings | %d dead | %d duplicates\n\n",
			summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors,
			summary.Duplicates)
	}
}

// getEmptyResultsMessage returns the appropriate message when no results match filters.
func getEmptyResultsMessage(summary checker.Summary, opts checkRenderOptions) string {
	switch {
	case opts.ShowAlive && summary.Alive == 0:
		return "No alive links found."
	case opts.ShowWarnings && summary.WarningsCount() == 0:
		return "No warnings found."
	case opts.ShowDead && !summary.HasDeadLinks():
		return "No dead links found."
	default:
		return "All links are alive!"
	}
}

// shouldGroupResults returns true if results should be displayed in grouped sections.
func shouldGroupResults(opts checkRenderOptions) bool {
	return opts.ShowAll || (!opts.ShowAlive && !opts.ShowWarnings && !opts.ShowDead)
}

// outputGroupedResults prints results grouped by status sections.
func outputGroupedResults(w io.Writer, filtered []checker.Result, opts checkRenderOptions) {
	printSection(w, "Warnings", FilterResultsWarnings(filtered), printWarningResult)
	printSection(w, "Dead Links", FilterResultsDead(filtered), printDeadResult)
	printSection(w, "Duplicates", FilterResultsDuplicates(filtered), printDuplicateResult)

	if opts.ShowAll {
		printSection(w, "Alive", FilterResultsAlive(filtered), printAliveResult)
	}
}

// outputFlatResults prints results as a flat list.
func outputFlatResults(w io.Writer, filtered []checker.Result) {
	for _, r := range filtered {
		printResult(w, r)
	}
}

// maybeShowIgnored shows ignored URLs if the flag is set and filter exists.
func maybeShowIgnored(w io.Writer, urlFilter *filter.Filter, opts checkRenderOptions) {
	if opts.ShowIgnored && urlFilter != nil {
		printIgnoredURLs(w, urlFilter)
	}
}
