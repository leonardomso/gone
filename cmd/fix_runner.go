package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/fixer"
)

type FixOptions struct {
	Yes            bool
	DryRun         bool
	Concurrency    int
	Timeout        int
	Retries        int
	ShowStats      bool
	FileTypes      []string
	StrictMode     bool
	IgnoreDomains  []string
	IgnorePatterns []string
	IgnoreRegex    []string
	NoConfig       bool
}

type fixRunner struct {
	opts FixOptions
	env  CommandEnv
	io   IOStreams
}

func newFixRunner(opts FixOptions, env CommandEnv, streams IOStreams) *fixRunner {
	return &fixRunner{
		opts: opts,
		env:  env,
		io:   streams,
	}
}

func currentFixOptions() FixOptions {
	return FixOptions{
		Yes:            fixYes,
		DryRun:         fixDryRun,
		Concurrency:    fixConcurrency,
		Timeout:        fixTimeout,
		Retries:        fixRetries,
		ShowStats:      fixShowStats,
		FileTypes:      append([]string{}, fixFileTypes...),
		StrictMode:     fixStrictMode,
		IgnoreDomains:  append([]string{}, fixIgnoreDomains...),
		IgnorePatterns: append([]string{}, fixIgnorePatterns...),
		IgnoreRegex:    append([]string{}, fixIgnoreRegex...),
		NoConfig:       fixNoConfig,
	}
}

func (r *fixRunner) Run(args []string) int {
	perf := r.env.NewStats()

	loadedCfg, err := r.env.LoadConfig(r.opts.NoConfig)
	if err != nil {
		fmt.Fprintf(r.io.ErrOut, "Config error: %v\n", err)
		return 1
	}

	path := getPathArg(args)
	effectiveTypes := loadedCfg.GetTypes(r.opts.FileTypes, []string{"md"})
	if err := validateFileTypes(effectiveTypes); err != nil {
		fmt.Fprintf(r.io.ErrOut, "Error: %v\n", err)
		return 1
	}

	perf.StartScan()
	files, err := r.env.FindFiles(loadedCfg.BuildScanOptions(path, r.opts.FileTypes, []string{"md"}))
	if err != nil {
		fmt.Fprintf(r.io.ErrOut, "Error scanning directory: %v\n", err)
		return 1
	}
	perf.EndScan(len(files))

	fmt.Fprintf(r.io.Out, "Found %d file(s) of type(s): %s\n", len(files), strings.Join(effectiveTypes, ", "))

	perf.StartParse()
	parserLinks, err := r.env.ExtractLinks(files, loadedCfg.GetStrict(r.opts.StrictMode))
	if err != nil {
		fmt.Fprintf(r.io.ErrOut, "Error parsing files: %v\n", err)
		return 1
	}

	if len(parserLinks) == 0 {
		perf.EndParse(0, 0, 0, 0)
		fmt.Fprintln(r.io.Out, "No links found.")
		if loadedCfg.GetShowStats(r.opts.ShowStats) {
			fmt.Fprint(r.io.Out, perf.String())
		}
		return 0
	}

	urlFilter, err := r.env.CreateFilterWithConfig(loadedCfg.Config(), r.opts.IgnoreDomains, r.opts.IgnorePatterns, r.opts.IgnoreRegex)
	if err != nil {
		fmt.Fprintf(r.io.ErrOut, "Error creating filter: %v\n", err)
		return 1
	}

	links := FilterParserLinks(parserLinks, urlFilter)
	ignoredCount := getIgnoredCount(urlFilter)
	uniqueURLs := CountUniqueURLs(links)
	duplicates := len(links) - uniqueURLs
	perf.EndParse(len(parserLinks), uniqueURLs, duplicates, ignoredCount)

	if len(links) == 0 {
		fmt.Fprintln(r.io.Out, "All links were ignored by filter rules.")
		if loadedCfg.GetShowStats(r.opts.ShowStats) {
			fmt.Fprint(r.io.Out, perf.String())
		}
		return 0
	}

	fmt.Fprintf(r.io.Out, "Checking %d unique URL(s) for redirects...\n", uniqueURLs)

	perf.StartCheck()
	results := r.env.NewChecker(loadedCfg.BuildCheckerOptions(r.opts.Concurrency, r.opts.Timeout, r.opts.Retries)).CheckAll(links)
	perf.EndCheck()

	f := r.env.NewFixer()
	f.SetParserLinks(parserLinks)
	changes := f.FindFixes(results)

	if len(changes) == 0 {
		fmt.Fprintln(r.io.Out, "\nNo fixable redirects found.")
		printFixSummary(r.io.Out, results)
		if loadedCfg.GetShowStats(r.opts.ShowStats) {
			fmt.Fprint(r.io.Out, perf.String())
		}
		return 0
	}

	fmt.Fprintln(r.io.Out)
	fmt.Fprint(r.io.Out, f.Preview(changes))

	if r.opts.DryRun {
		fmt.Fprintln(r.io.Out, "Dry-run mode: no files were modified.")
		if loadedCfg.GetShowStats(r.opts.ShowStats) {
			fmt.Fprint(r.io.Out, perf.String())
		}
		return 0
	}

	if r.opts.Yes {
		applyAllFixes(r.io.Out, f, changes)
		if loadedCfg.GetShowStats(r.opts.ShowStats) {
			fmt.Fprint(r.io.Out, perf.String())
		}
		return 0
	}

	code := runInteractiveFix(r.io, f, changes)
	if loadedCfg.GetShowStats(r.opts.ShowStats) {
		fmt.Fprint(r.io.Out, perf.String())
	}
	return code
}

func applyAllFixes(out io.Writer, f RedirectFixer, changes []fixer.FileChanges) {
	results := f.ApplyAll(changes)
	fmt.Fprintln(out, fixer.DetailedSummary(results))
}

func runInteractiveFix(streams IOStreams, f RedirectFixer, changes []fixer.FileChanges) int {
	reader := bufio.NewReader(streams.In)
	var allResults []fixer.FixResult
	applyAll := false

	for i := 0; i < len(changes); i++ {
		fc := changes[i]

		if applyAll {
			result, _ := f.ApplyToFile(fc)
			allResults = append(allResults, *result)
			continue
		}

		fmt.Fprintf(streams.Out, "\nFix %s? (%d change(s)) [y/n/a/q/?] ", fc.FilePath, fc.TotalFixes)

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(streams.ErrOut, "\nError reading input: %v\n", err)
			return 1
		}

		switch strings.TrimSpace(strings.ToLower(input)) {
		case "y", "yes":
			result, applyErr := f.ApplyToFile(fc)
			if applyErr != nil {
				fmt.Fprintf(streams.ErrOut, "Error: %v\n", applyErr)
			} else {
				fmt.Fprintf(streams.Out, "Fixed %d redirect(s) in %s\n", result.Applied, fc.FilePath)
			}
			allResults = append(allResults, *result)
		case "n", "no":
			fmt.Fprintf(streams.Out, "Skipped %s\n", fc.FilePath)
			allResults = append(allResults, fixer.FixResult{
				FilePath: fc.FilePath,
				Skipped:  fc.TotalFixes,
			})
		case "a", "all":
			result, applyErr := f.ApplyToFile(fc)
			if applyErr != nil {
				fmt.Fprintf(streams.ErrOut, "Error: %v\n", applyErr)
			} else {
				fmt.Fprintf(streams.Out, "Fixed %d redirect(s) in %s\n", result.Applied, fc.FilePath)
			}
			allResults = append(allResults, *result)
			applyAll = true
		case "q", "quit":
			fmt.Fprintln(streams.Out, "\nQuitting. Remaining files were not modified.")
			for j := i; j < len(changes); j++ {
				allResults = append(allResults, fixer.FixResult{
					FilePath: changes[j].FilePath,
					Skipped:  changes[j].TotalFixes,
				})
			}
			printInteractiveResults(streams.Out, allResults)
			return 2
		case "?", "help", "h":
			printInteractiveHelp(streams.Out)
			i--
		default:
			fmt.Fprintln(streams.Out, "Invalid input. Use y/n/a/q/? (or type 'help')")
			i--
		}
	}

	fmt.Fprintln(streams.Out)
	printInteractiveResults(streams.Out, allResults)
	return 0
}

func printInteractiveHelp(out io.Writer) {
	fmt.Fprintln(out, `
Interactive mode options:
  y, yes  - Fix this file
  n, no   - Skip this file
  a, all  - Fix this file and all remaining files
  q, quit - Quit without fixing remaining files
  ?, help - Show this help`)
}

func printInteractiveResults(out io.Writer, results []fixer.FixResult) {
	applied := 0
	filesModified := 0
	filesSkipped := 0

	for _, r := range results {
		applied += r.Applied
		if r.Applied > 0 {
			filesModified++
		}
		if r.Skipped > 0 && r.Applied == 0 {
			filesSkipped++
		}
	}

	if applied > 0 {
		fmt.Fprintf(out, "Fixed %d redirect(s) across %d file(s).\n", applied, filesModified)
	}
	if filesSkipped > 0 {
		fmt.Fprintf(out, "Skipped %d file(s).\n", filesSkipped)
	}
}

func printFixSummary(out io.Writer, results []checker.Result) {
	summary := checker.Summarize(results)
	fmt.Fprintf(out, "\nLink status: %d alive | %d redirects | %d dead | %d errors\n",
		summary.Alive, summary.Redirects, summary.Dead, summary.Errors)

	if summary.Redirects == 0 {
		return
	}

	notFixable := 0
	for _, r := range results {
		if r.Status == checker.StatusRedirect && r.FinalStatus != 200 {
			notFixable++
		}
	}
	if notFixable > 0 {
		fmt.Fprintf(out, "Note: %d redirect(s) lead to non-200 responses and cannot be auto-fixed.\n", notFixable)
	}
}
