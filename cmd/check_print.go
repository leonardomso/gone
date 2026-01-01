package cmd

import (
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/helpers"
)

// printProgressMessage displays the scanning progress with ignore info.
// Shows the total links found, unique URLs being checked, and counts for duplicates/ignored.
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

// printSection prints a titled section of results if any exist.
// Uses the provided printer function to format each individual result.
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

// printResult dispatches to the appropriate printer based on result status.
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

// printAliveResult formats and prints a result with alive status.
func printAliveResult(r checker.Result) {
	fmt.Printf("  [%d] %s\n", r.StatusCode, r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
		fmt.Printf("       Text: %q\n", text)
	}
	fmt.Printf("       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Printf(":%d", r.Link.Line)
	}
	fmt.Println()
	fmt.Println()
}

// printWarningResult formats and prints a result with warning status (redirect or blocked).
func printWarningResult(r checker.Result) {
	fmt.Printf("  %s %s\n", r.StatusDisplay(), r.Link.URL)

	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
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

// printDeadResult formats and prints a result with dead or error status.
func printDeadResult(r checker.Result) {
	fmt.Printf("  %s %s\n", r.StatusDisplay(), r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
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

// printDuplicateResult formats and prints a result that is a duplicate of another link.
func printDuplicateResult(r checker.Result) {
	fmt.Printf("  [DUPLICATE] %s\n", r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
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

// printIgnoredURLs displays the list of URLs that were ignored by filter rules.
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

// formatRedirectChain formats a redirect chain as a string showing status codes.
// Example output: "301 → 302 → 200".
func formatRedirectChain(r checker.Result) string {
	parts := make([]string, 0, len(r.RedirectChain)+1)
	for _, red := range r.RedirectChain {
		parts = append(parts, fmt.Sprintf("%d", red.StatusCode))
	}
	parts = append(parts, fmt.Sprintf("%d", r.FinalStatus))
	return strings.Join(parts, " → ")
}
