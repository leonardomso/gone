package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/helpers"
)

// printProgressMessage displays the scanning progress with ignore info.
// Shows the total links found, unique URLs being checked, and counts for duplicates/ignored.
func printProgressMessage(w io.Writer, total, afterFilter, unique, duplicates, ignored int) {
	var parts []string

	if duplicates > 0 {
		parts = append(parts, fmt.Sprintf("%d duplicates", duplicates))
	}
	if ignored > 0 {
		parts = append(parts, fmt.Sprintf("%d ignored", ignored))
	}

	if len(parts) > 0 {
		fmt.Fprintf(w, "Found %d link(s), checking %d unique URLs (%s)...\n",
			total, unique, strings.Join(parts, ", "))
	} else {
		fmt.Fprintf(w, "Found %d link(s), checking...\n", afterFilter)
	}
}

// printSection prints a titled section of results if any exist.
// Uses the provided printer function to format each individual result.
func printSection(w io.Writer, title string, results []checker.Result, printer func(io.Writer, checker.Result)) {
	if len(results) == 0 {
		return
	}
	fmt.Fprintf(w, "=== %s (%d) ===\n\n", title, len(results))
	for _, r := range results {
		printer(w, r)
	}
	fmt.Fprintln(w)
}

// printResult dispatches to the appropriate printer based on result status.
func printResult(w io.Writer, r checker.Result) {
	switch r.Status {
	case checker.StatusAlive:
		printAliveResult(w, r)
	case checker.StatusRedirect, checker.StatusBlocked:
		printWarningResult(w, r)
	case checker.StatusDead, checker.StatusError:
		printDeadResult(w, r)
	case checker.StatusDuplicate:
		printDuplicateResult(w, r)
	}
}

// printAliveResult formats and prints a result with alive status.
func printAliveResult(w io.Writer, r checker.Result) {
	fmt.Fprintf(w, "  [%d] %s\n", r.StatusCode, r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
		fmt.Fprintf(w, "       Text: %q\n", text)
	}
	fmt.Fprintf(w, "       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Fprintf(w, ":%d", r.Link.Line)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

// printWarningResult formats and prints a result with warning status (redirect or blocked).
func printWarningResult(w io.Writer, r checker.Result) {
	fmt.Fprintf(w, "  %s %s\n", r.StatusDisplay(), r.Link.URL)

	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
		fmt.Fprintf(w, "       Text: %q\n", text)
	}

	if r.Status == checker.StatusRedirect && len(r.RedirectChain) > 0 {
		fmt.Fprintf(w, "       Chain: %s\n", formatRedirectChain(r))
		fmt.Fprintf(w, "       Final: %s\n", r.FinalURL)
	}

	fmt.Fprintf(w, "       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Fprintf(w, ":%d", r.Link.Line)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "       Note: %s\n\n", r.Status.Description())
}

// printDeadResult formats and prints a result with dead or error status.
func printDeadResult(w io.Writer, r checker.Result) {
	fmt.Fprintf(w, "  %s %s\n", r.StatusDisplay(), r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
		fmt.Fprintf(w, "       Text: %q\n", text)
	}
	fmt.Fprintf(w, "       File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Fprintf(w, ":%d", r.Link.Line)
	}
	fmt.Fprintln(w)

	if r.Error != "" {
		fmt.Fprintf(w, "       Error: %s\n", r.Error)
	}
	fmt.Fprintln(w)
}

// printDuplicateResult formats and prints a result that is a duplicate of another link.
func printDuplicateResult(w io.Writer, r checker.Result) {
	fmt.Fprintf(w, "  [DUPLICATE] %s\n", r.Link.URL)
	if text := helpers.TruncateText(r.Link.Text, 50); text != "" {
		fmt.Fprintf(w, "              Text: %q\n", text)
	}
	fmt.Fprintf(w, "              File: %s", r.Link.FilePath)
	if r.Link.Line > 0 {
		fmt.Fprintf(w, ":%d", r.Link.Line)
	}
	fmt.Fprintln(w)

	if r.DuplicateOf != nil {
		fmt.Fprintf(w, "              Same as: %s", r.DuplicateOf.Link.FilePath)
		if r.DuplicateOf.Link.Line > 0 {
			fmt.Fprintf(w, ":%d", r.DuplicateOf.Link.Line)
		}
		fmt.Fprintf(w, " → Status: %s\n", r.DuplicateOf.Status.Label())
	}
	fmt.Fprintln(w)
}

// printIgnoredURLs displays the list of URLs that were ignored by filter rules.
func printIgnoredURLs(w io.Writer, urlFilter *filter.Filter) {
	ignored := urlFilter.IgnoredURLs()
	if len(ignored) == 0 {
		return
	}

	fmt.Fprintf(w, "\n=== Ignored URLs (%d) ===\n\n", len(ignored))
	for _, ig := range ignored {
		fmt.Fprintf(w, "  [IGNORED] %s\n", ig.URL)
		fmt.Fprintf(w, "            File: %s", ig.File)
		if ig.Line > 0 {
			fmt.Fprintf(w, ":%d", ig.Line)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "            Reason: %s %q\n\n", ig.Type, ig.Rule)
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
