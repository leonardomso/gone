package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/fixer"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"

	"github.com/spf13/cobra"
)

// Fix command flag variables.
var (
	fixYes         bool
	fixDryRun      bool
	fixConcurrency int
	fixTimeout     int
	fixRetries     int

	// Ignore flags (shared with check).
	fixIgnoreDomains  []string
	fixIgnorePatterns []string
	fixIgnoreRegex    []string
	fixNoConfig       bool
)

// fixCmd represents the fix command.
var fixCmd = &cobra.Command{
	Use:   "fix [path]",
	Short: "Automatically fix redirect URLs in markdown files",
	Long: `Scan markdown files for redirect URLs and update them to their final destinations.

Only redirects where the final destination returns 200 OK are fixed.
Dead links, errors, and blocked URLs are not modified.

By default, the command runs interactively, prompting for each file.
Use --yes to apply all fixes automatically (useful for CI/scripts).
Use --dry-run to preview changes without modifying files.

Examples:
  gone fix                      # Interactive mode, scan current directory
  gone fix ./docs               # Interactive mode, scan specific directory
  gone fix --dry-run            # Preview what would be fixed
  gone fix --yes                # Apply all fixes without prompting
  gone fix --yes --dry-run      # Preview all fixes (no prompts, no changes)

Ignore patterns (same as check command):
  gone fix --ignore-domain=localhost
  gone fix --ignore-pattern="*.local/*"
  gone fix --no-config          # Skip .gonerc.yaml`,
	Args: cobra.MaximumNArgs(1),
	Run:  runFix,
}

func init() {
	rootCmd.AddCommand(fixCmd)

	// Mode flags
	fixCmd.Flags().BoolVarP(&fixYes, "yes", "y", false,
		"Apply all fixes without prompting")
	fixCmd.Flags().BoolVarP(&fixDryRun, "dry-run", "n", false,
		"Preview changes without modifying files")

	// Performance options
	fixCmd.Flags().IntVarP(&fixConcurrency, "concurrency", "c", 10,
		"Number of concurrent workers")
	fixCmd.Flags().IntVarP(&fixTimeout, "timeout", "t", 10,
		"Timeout per request in seconds")
	fixCmd.Flags().IntVarP(&fixRetries, "retries", "r", 2,
		"Number of retries for failed requests")

	// Ignore options
	fixCmd.Flags().StringSliceVar(&fixIgnoreDomains, "ignore-domain", nil,
		"Domains to ignore (can be repeated or comma-separated)")
	fixCmd.Flags().StringSliceVar(&fixIgnorePatterns, "ignore-pattern", nil,
		"Glob patterns to ignore (can be repeated)")
	fixCmd.Flags().StringSliceVar(&fixIgnoreRegex, "ignore-regex", nil,
		"Regex patterns to ignore (can be repeated)")
	fixCmd.Flags().BoolVar(&fixNoConfig, "no-config", false,
		"Skip loading .gonerc.yaml config file")
}

// runFix is the main entry point for the fix command.
// It scans for redirects and applies fixes interactively or automatically.
func runFix(_ *cobra.Command, args []string) {
	// Determine the path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Find all markdown files
	files, err := scanner.FindMarkdownFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d markdown file(s)\n", len(files))

	// Extract all URLs from the files
	parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
		os.Exit(1)
	}

	if len(parserLinks) == 0 {
		fmt.Println("No links found.")
		return
	}

	// Load and create filter using shared helper
	urlFilter, err := CreateFilter(FilterOptions{
		Domains:  fixIgnoreDomains,
		Patterns: fixIgnorePatterns,
		Regex:    fixIgnoreRegex,
		NoConfig: fixNoConfig,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filter: %v\n", err)
		os.Exit(1)
	}

	// Convert parser.Link to checker.Link, applying filter
	links := FilterParserLinks(parserLinks, urlFilter)

	if len(links) == 0 {
		fmt.Println("All links were ignored by filter rules.")
		return
	}

	// Count unique URLs using shared helper
	uniqueURLs := CountUniqueURLs(links)
	fmt.Printf("Checking %d unique URL(s) for redirects...\n", uniqueURLs)

	// Create checker and check all links
	opts := checker.DefaultOptions().
		WithConcurrency(fixConcurrency).
		WithTimeout(time.Duration(fixTimeout) * time.Second).
		WithMaxRetries(fixRetries)

	c := checker.New(opts)
	results := c.CheckAll(links)

	// Create fixer and find fixable items
	f := fixer.New()
	f.SetParserLinks(parserLinks)
	changes := f.FindFixes(results)

	if len(changes) == 0 {
		fmt.Println("\nNo fixable redirects found.")
		printFixSummary(results)
		return
	}

	// Show preview
	fmt.Println()
	fmt.Print(f.Preview(changes))

	// Handle dry-run mode
	if fixDryRun {
		fmt.Println("Dry-run mode: no files were modified.")
		return
	}

	// Handle automatic mode
	if fixYes {
		applyAllFixes(f, changes)
		return
	}

	// Interactive mode
	runInteractiveFix(f, changes)
}

// applyAllFixes applies all fixes without prompting.
func applyAllFixes(f *fixer.Fixer, changes []fixer.FileChanges) {
	results := f.ApplyAll(changes)
	fmt.Println(fixer.DetailedSummary(results))
}

// runInteractiveFix prompts the user for each file before applying fixes.
func runInteractiveFix(f *fixer.Fixer, changes []fixer.FileChanges) {
	reader := bufio.NewReader(os.Stdin)
	var allResults []fixer.FixResult
	applyAll := false

	for i := 0; i < len(changes); i++ {
		fc := changes[i]

		if applyAll {
			result, _ := f.ApplyToFile(fc)
			allResults = append(allResults, *result)
			continue
		}

		// Prompt for this file
		fmt.Printf("\nFix %s? (%d change(s)) [y/n/a/q/?] ",
			fc.FilePath, fc.TotalFixes)

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes":
			result, applyErr := f.ApplyToFile(fc)
			if applyErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", applyErr)
			} else {
				fmt.Printf("Fixed %d redirect(s) in %s\n", result.Applied, fc.FilePath)
			}
			allResults = append(allResults, *result)

		case "n", "no":
			fmt.Printf("Skipped %s\n", fc.FilePath)
			allResults = append(allResults, fixer.FixResult{
				FilePath: fc.FilePath,
				Skipped:  fc.TotalFixes,
			})

		case "a", "all":
			// Apply this file and all remaining
			result, applyErr := f.ApplyToFile(fc)
			if applyErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", applyErr)
			} else {
				fmt.Printf("Fixed %d redirect(s) in %s\n", result.Applied, fc.FilePath)
			}
			allResults = append(allResults, *result)
			applyAll = true

		case "q", "quit":
			fmt.Println("\nQuitting. Remaining files were not modified.")
			// Add remaining as skipped
			for j := i; j < len(changes); j++ {
				allResults = append(allResults, fixer.FixResult{
					FilePath: changes[j].FilePath,
					Skipped:  changes[j].TotalFixes,
				})
			}
			printInteractiveResults(allResults)
			os.Exit(2)

		case "?", "help":
			printInteractiveHelp()
			i-- // Re-prompt for this file

		default:
			fmt.Println("Invalid input. Use y/n/a/q/? (or type 'help')")
			i-- // Retry this file
		}
	}

	fmt.Println()
	printInteractiveResults(allResults)
}

// printInteractiveHelp displays help for interactive mode options.
func printInteractiveHelp() {
	fmt.Println(`
Interactive mode options:
  y, yes  - Fix this file
  n, no   - Skip this file
  a, all  - Fix this file and all remaining files
  q, quit - Quit without fixing remaining files
  ?, help - Show this help`)
}

// printInteractiveResults displays a summary of the interactive session.
func printInteractiveResults(results []fixer.FixResult) {
	applied := 0
	skipped := 0
	filesModified := 0
	filesSkipped := 0

	for _, r := range results {
		applied += r.Applied
		skipped += r.Skipped
		if r.Applied > 0 {
			filesModified++
		}
		if r.Skipped > 0 && r.Applied == 0 {
			filesSkipped++
		}
	}

	if applied > 0 {
		fmt.Printf("Fixed %d redirect(s) across %d file(s).\n", applied, filesModified)
	}
	if filesSkipped > 0 {
		fmt.Printf("Skipped %d file(s).\n", filesSkipped)
	}
}

// printFixSummary displays a summary of check results for context.
func printFixSummary(results []checker.Result) {
	summary := checker.Summarize(results)

	fmt.Printf("\nLink status: %d alive | %d redirects | %d dead | %d errors\n",
		summary.Alive, summary.Redirects, summary.Dead, summary.Errors)

	if summary.Redirects > 0 {
		// Count how many redirects are not fixable (final status != 200)
		notFixable := 0
		for _, r := range results {
			if r.Status == checker.StatusRedirect && r.FinalStatus != 200 {
				notFixable++
			}
		}
		if notFixable > 0 {
			fmt.Printf("Note: %d redirect(s) lead to non-200 responses and cannot be auto-fixed.\n",
				notFixable)
		}
	}
}
