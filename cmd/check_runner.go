package cmd

import (
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/output"
	"github.com/leonardomso/gone/internal/stats"
)

type checkOptions struct {
	OutputFormat      string
	OutputFile        string
	Concurrency       int
	Timeout           int
	Retries           int
	ShowAlive         bool
	ShowWarnings      bool
	ShowDead          bool
	ShowAll           bool
	ShowStats         bool
	FileTypes         []string
	StrictMode        bool
	IgnoreDomains     []string
	IgnorePatterns    []string
	IgnoreRegex       []string
	ShowIgnored       bool
	NoConfig          bool
	AllowPrivateHosts bool
}

type checkRunner struct {
	opts checkOptions
	env  CommandEnv
	io   IOStreams
}

func newCheckRunner(opts checkOptions, env CommandEnv, streams IOStreams) *checkRunner {
	return &checkRunner{
		opts: opts,
		env:  env,
		io:   streams,
	}
}

func currentCheckOptions() checkOptions {
	return checkOptions{
		OutputFormat:      outputFormat,
		OutputFile:        outputFile,
		Concurrency:       concurrency,
		Timeout:           timeout,
		Retries:           retries,
		ShowAlive:         showAlive,
		ShowWarnings:      showWarnings,
		ShowDead:          showDead,
		ShowAll:           showAll,
		ShowStats:         showStats,
		FileTypes:         append([]string{}, fileTypes...),
		StrictMode:        strictMode,
		IgnoreDomains:     append([]string{}, ignoreDomains...),
		IgnorePatterns:    append([]string{}, ignorePatterns...),
		IgnoreRegex:       append([]string{}, ignoreRegex...),
		ShowIgnored:       showIgnored,
		NoConfig:          noConfig,
		AllowPrivateHosts: allowPrivateHosts,
	}
}

func (r *checkRunner) renderOptions() checkRenderOptions {
	return checkRenderOptions{
		ShowAlive:    r.opts.ShowAlive,
		ShowWarnings: r.opts.ShowWarnings,
		ShowDead:     r.opts.ShowDead,
		ShowAll:      r.opts.ShowAll,
		ShowIgnored:  r.opts.ShowIgnored,
	}
}

func (r *checkRunner) Run(args []string) int {
	perf := r.env.NewStats()
	if err := r.validateFlags(); err != nil {
		return r.fail("Invalid flags", err)
	}

	loadedCfg, err := r.env.LoadConfig(r.opts.NoConfig)
	if err != nil {
		return r.fail("Config error", err)
	}

	path := getPathArg(args)
	effectiveFormat := loadedCfg.GetOutputFormat(r.opts.OutputFormat)
	useStructuredOutput := effectiveFormat != ""

	files, err := r.scanFilesWithConfig(path, loadedCfg, perf, useStructuredOutput)
	if err != nil {
		return r.fail("Error scanning directory", err)
	}

	links, urlFilter, done, err := r.parseAndFilterLinksWithConfig(
		files,
		loadedCfg,
		perf,
		useStructuredOutput,
	)
	if err != nil {
		return r.fail(err.Error(), nil)
	}
	if done {
		return 0
	}

	results, summary := r.checkLinksWithConfig(links, loadedCfg, perf)

	if err := r.routeOutputWithConfig(
		files, results, summary, urlFilter, perf,
		useStructuredOutput, effectiveFormat, loadedCfg.GetShowStats(r.opts.ShowStats),
	); err != nil {
		return r.fail(err.Error(), nil)
	}

	if summary.HasDeadLinks() {
		return 1
	}
	return 0
}

func (r *checkRunner) fail(message string, err error) int {
	if err != nil {
		if message != "" {
			writef(r.io.ErrOut, "%s: %v\n", message, err)
		} else {
			writef(r.io.ErrOut, "%v\n", err)
		}
		return 1
	}

	if message != "" {
		writeln(r.io.ErrOut, message)
	}
	return 1
}

func (r *checkRunner) validateFlags() error {
	if r.opts.OutputFormat != "" && r.opts.OutputFile != "" {
		return fmt.Errorf("--format and --output are mutually exclusive; " +
			"use --format for stdout output, or --output for file output")
	}

	if r.opts.OutputFormat != "" && !output.IsValidFormat(r.opts.OutputFormat) {
		return fmt.Errorf("invalid format %q; valid formats: %s",
			r.opts.OutputFormat, strings.Join(output.ValidFormats(), ", "))
	}

	return nil
}

func (r *checkRunner) scanFilesWithConfig(
	path string, cfg *LoadedConfig, perf *stats.Stats, useStructuredOutput bool,
) ([]string, error) {
	perf.StartScan()

	effectiveTypes := cfg.GetTypes(r.opts.FileTypes, []string{"md"})
	if err := validateFileTypes(effectiveTypes); err != nil {
		return nil, fmt.Errorf("invalid file types: %w", err)
	}

	scanOpts := cfg.BuildScanOptions(path, r.opts.FileTypes, []string{"md"})
	files, err := r.env.FindFiles(scanOpts)
	if err != nil {
		return nil, err
	}

	perf.EndScan(len(files))

	if !useStructuredOutput {
		writef(r.io.Out, "Found %d file(s) of type(s): %s\n", len(files), strings.Join(effectiveTypes, ", "))
	}

	return files, nil
}

func (r *checkRunner) parseAndFilterLinksWithConfig(
	files []string, cfg *LoadedConfig, perf *stats.Stats, useStructuredOutput bool,
) ([]checker.Link, *filter.Filter, bool, error) {
	perf.StartParse()

	parserLinks, err := r.env.ExtractLinks(files, cfg.GetStrict(r.opts.StrictMode))
	if err != nil {
		return nil, nil, false, fmt.Errorf("error parsing files: %w", err)
	}

	if len(parserLinks) == 0 {
		perf.EndParse(0, 0, 0, 0)
		if err := r.handleEmptyLinksWithStats(
			files,
			useStructuredOutput,
			perf,
			cfg.GetOutputFormat(r.opts.OutputFormat),
			cfg.GetShowStats(r.opts.ShowStats),
		); err != nil {
			return nil, nil, false, fmt.Errorf("error formatting output: %w", err)
		}
		return nil, nil, true, nil
	}

	urlFilter, err := r.env.CreateFilterWithConfig(
		cfg.Config(),
		r.opts.IgnoreDomains,
		r.opts.IgnorePatterns,
		r.opts.IgnoreRegex,
	)
	if err != nil {
		return nil, nil, false, fmt.Errorf("error creating filter: %w", err)
	}

	links := FilterParserLinks(parserLinks, urlFilter)
	ignoredCount := getIgnoredCount(urlFilter)
	uniqueURLs := CountUniqueURLs(links)
	duplicates := len(links) - uniqueURLs
	perf.EndParse(len(parserLinks), uniqueURLs, duplicates, ignoredCount)

	if !useStructuredOutput {
		printProgressMessage(r.io.Out, len(parserLinks), len(links), uniqueURLs, duplicates, ignoredCount)
	}

	if len(links) == 0 {
		if err := r.handleAllFilteredWithStats(
			files,
			useStructuredOutput,
			urlFilter,
			perf,
			cfg.GetOutputFormat(r.opts.OutputFormat),
			cfg.GetShowStats(r.opts.ShowStats),
		); err != nil {
			return nil, nil, false, fmt.Errorf("error formatting output: %w", err)
		}
		return nil, urlFilter, true, nil
	}

	return links, urlFilter, false, nil
}

func (r *checkRunner) checkLinksWithConfig(
	links []checker.Link, cfg *LoadedConfig, perf *stats.Stats,
) ([]checker.Result, checker.Summary) {
	perf.StartCheck()
	c := r.env.NewChecker(cfg.BuildCheckerOptions(
		r.opts.Concurrency, r.opts.Timeout, r.opts.Retries,
		r.opts.AllowPrivateHosts,
	))
	results := c.CheckAll(links)
	summary := checker.Summarize(results)
	perf.EndCheck()
	return results, summary
}

func (r *checkRunner) routeOutputWithConfig(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats, useStructuredOutput bool,
	effectiveFormat string, effectiveShowStats bool,
) error {
	switch {
	case useStructuredOutput:
		return r.handleStructuredOutputWithStats(
			files,
			results,
			summary,
			urlFilter,
			perf,
			effectiveFormat,
			effectiveShowStats,
		)
	case r.opts.OutputFile != "":
		return r.handleFileOutputWithStats(files, results, summary, urlFilter, perf, effectiveShowStats)
	default:
		outputText(r.io.Out, results, summary, urlFilter, r.renderOptions())
		if effectiveShowStats {
			writeString(r.io.Out, perf.String())
		}
		return nil
	}
}

func (r *checkRunner) handleEmptyLinksWithStats(
	files []string,
	useStructuredOutput bool,
	perf *stats.Stats,
	effectiveFormat string,
	effectiveShowStats bool,
) error {
	switch {
	case useStructuredOutput:
		return r.handleStructuredOutputWithStats(
			files,
			nil,
			checker.Summary{},
			nil,
			perf,
			effectiveFormat,
			effectiveShowStats,
		)
	case r.opts.OutputFile != "":
		return r.handleFileOutputWithStats(files, nil, checker.Summary{}, nil, perf, effectiveShowStats)
	default:
		writeln(r.io.Out, "No links found.")
		if effectiveShowStats {
			writeString(r.io.Out, perf.String())
		}
		return nil
	}
}

func (r *checkRunner) handleAllFilteredWithStats(
	files []string, useStructuredOutput bool, urlFilter *filter.Filter,
	perf *stats.Stats, effectiveFormat string, effectiveShowStats bool,
) error {
	switch {
	case useStructuredOutput:
		return r.handleStructuredOutputWithStats(
			files,
			nil,
			checker.Summary{},
			urlFilter,
			perf,
			effectiveFormat,
			effectiveShowStats,
		)
	case r.opts.OutputFile != "":
		return r.handleFileOutputWithStats(files, nil, checker.Summary{}, urlFilter, perf, effectiveShowStats)
	default:
		writeln(r.io.Out, "\nAll links were ignored by filter rules.")
		if r.opts.ShowIgnored && urlFilter != nil {
			printIgnoredURLs(r.io.Out, urlFilter)
		}
		if effectiveShowStats {
			writeString(r.io.Out, perf.String())
		}
		return nil
	}
}

func (r *checkRunner) handleStructuredOutputWithStats(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats, effectiveFormat string, effectiveShowStats bool,
) error {
	report := buildReport(r.env.Now(), files, results, summary, urlFilter, r.renderOptions())
	if effectiveShowStats && perf != nil {
		report.Stats = perf.ToJSON()
	}

	data, err := r.env.FormatReport(report, output.Format(effectiveFormat))
	if err != nil {
		return fmt.Errorf("error formatting output: %w", err)
	}

	_, err = r.io.Out.Write(data)
	return err
}

func (r *checkRunner) handleFileOutputWithStats(
	files []string, results []checker.Result, summary checker.Summary,
	urlFilter *filter.Filter, perf *stats.Stats, effectiveShowStats bool,
) error {
	report := buildReport(r.env.Now(), files, results, summary, urlFilter, r.renderOptions())
	if effectiveShowStats && perf != nil {
		report.Stats = perf.ToJSON()
	}

	if err := r.env.WriteToFile(report, r.opts.OutputFile); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	writef(r.io.Out, "Wrote report to %s\n", r.opts.OutputFile)
	writef(r.io.Out, "\nSummary: %d alive | %d warnings | %d dead | %d duplicates",
		summary.Alive, summary.WarningsCount(), summary.Dead+summary.Errors, summary.Duplicates)
	if urlFilter != nil && urlFilter.IgnoredCount() > 0 {
		writef(r.io.Out, " | %d ignored", urlFilter.IgnoredCount())
	}
	writeln(r.io.Out)

	if effectiveShowStats {
		writeString(r.io.Out, perf.String())
	}

	return nil
}
