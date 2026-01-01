package output

import (
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
)

// MarkdownFormatter formats reports as Markdown.
type MarkdownFormatter struct{}

// Format implements Formatter.
func (m *MarkdownFormatter) Format(report *Report) ([]byte, error) {
	var b strings.Builder
	b.Grow(len(report.Results)*200 + 500)

	m.writeHeader(&b, report)
	m.writeSummaryTable(&b, report)
	m.writeDeadLinksSection(&b, report.Results)
	m.writeWarningsSection(&b, report.Results)
	m.writeDuplicatesSection(&b, report.Results)
	m.writeIgnoredSection(&b, report.Ignored)

	return []byte(b.String()), nil
}

// writeHeader writes the report header section.
func (*MarkdownFormatter) writeHeader(b *strings.Builder, report *Report) {
	b.WriteString("# Gone Link Check Report\n\n")
	fmt.Fprintf(b, "**Generated:** %s  \n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(b, "**Files Scanned:** %d  \n", len(report.Files))
	fmt.Fprintf(b, "**Total Links:** %d  \n", report.TotalLinks)
	fmt.Fprintf(b, "**Unique URLs:** %d\n\n", report.UniqueURLs)
}

// writeSummaryTable writes the summary statistics table.
func (*MarkdownFormatter) writeSummaryTable(b *strings.Builder, report *Report) {
	b.WriteString("## Summary\n\n")
	b.WriteString("| Status | Count |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(b, "| Alive | %d |\n", report.Summary.Alive)
	fmt.Fprintf(b, "| Warnings | %d |\n", report.Summary.WarningsCount())
	fmt.Fprintf(b, "| Dead | %d |\n", report.Summary.Dead+report.Summary.Errors)
	fmt.Fprintf(b, "| Duplicates | %d |\n", report.Summary.Duplicates)
	if len(report.Ignored) > 0 {
		fmt.Fprintf(b, "| Ignored | %d |\n", len(report.Ignored))
	}
	b.WriteString("\n")
}

// writeDeadLinksSection writes the dead links section if any exist.
func (m *MarkdownFormatter) writeDeadLinksSection(b *strings.Builder, results []checker.Result) {
	deadLinks := filterByStatus(results, checker.StatusDead, checker.StatusError)
	if len(deadLinks) == 0 {
		return
	}

	fmt.Fprintf(b, "## Dead Links (%d)\n\n", len(deadLinks))
	m.writeDeadLinksTable(b, deadLinks)
	m.writeDeadLinksDetails(b, deadLinks)
}

// writeDeadLinksTable writes the dead links summary table.
func (*MarkdownFormatter) writeDeadLinksTable(b *strings.Builder, deadLinks []checker.Result) {
	b.WriteString("| Status | URL | Text | File | Line |\n")
	b.WriteString("|--------|-----|------|------|------|\n")
	for _, r := range deadLinks {
		status := formatStatusForMarkdown(r)
		text := escapeMarkdown(truncateText(r.Link.Text, 40))
		url := escapeMarkdown(truncateText(r.Link.URL, 60))
		fmt.Fprintf(b, "| %s | %s | %s | %s | %d |\n",
			status, url, text, r.Link.FilePath, r.Link.Line)
	}
	b.WriteString("\n")
}

// writeDeadLinksDetails writes detailed info for each dead link.
func (*MarkdownFormatter) writeDeadLinksDetails(b *strings.Builder, deadLinks []checker.Result) {
	b.WriteString("### Details\n\n")
	for _, r := range deadLinks {
		fmt.Fprintf(b, "#### %s\n\n", escapeMarkdown(r.Link.URL))
		if r.Link.Text != "" {
			fmt.Fprintf(b, "- **Text:** %q\n", r.Link.Text)
		}
		fmt.Fprintf(b, "- **File:** `%s:%d`\n", r.Link.FilePath, r.Link.Line)
		fmt.Fprintf(b, "- **Status:** %s\n", formatStatusForMarkdown(r))
		writeRedirectChain(b, r)
		if r.Error != "" {
			fmt.Fprintf(b, "- **Error:** %s\n", r.Error)
		}
		b.WriteString("\n")
	}
}

// writeRedirectChain writes the redirect chain for a result if present.
func writeRedirectChain(b *strings.Builder, r checker.Result) {
	if len(r.RedirectChain) == 0 {
		return
	}
	b.WriteString("- **Redirect Chain:**\n")
	for i, red := range r.RedirectChain {
		fmt.Fprintf(b, "  %d. `%d` → %s\n", i+1, red.StatusCode, red.URL)
	}
	fmt.Fprintf(b, "  Final: `%d` → %s\n", r.FinalStatus, r.FinalURL)
}

// writeWarningsSection writes the warnings section if any exist.
func (m *MarkdownFormatter) writeWarningsSection(b *strings.Builder, results []checker.Result) {
	warnings := filterByStatus(results, checker.StatusRedirect, checker.StatusBlocked)
	if len(warnings) == 0 {
		return
	}

	fmt.Fprintf(b, "## Warnings (%d)\n\n", len(warnings))
	m.writeWarningsTable(b, warnings)
	m.writeRedirectDetails(b, results)
}

// writeWarningsTable writes the warnings summary table.
func (*MarkdownFormatter) writeWarningsTable(b *strings.Builder, warnings []checker.Result) {
	b.WriteString("| Issue | URL | Text | Final URL | File | Line |\n")
	b.WriteString("|-------|-----|------|-----------|------|------|\n")
	for _, r := range warnings {
		issue := r.Status.Label()
		text := escapeMarkdown(truncateText(r.Link.Text, 30))
		url := escapeMarkdown(truncateText(r.Link.URL, 50))
		finalURL := ""
		if r.Status == checker.StatusRedirect {
			finalURL = escapeMarkdown(truncateText(r.FinalURL, 50))
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %d |\n",
			issue, url, text, finalURL, r.Link.FilePath, r.Link.Line)
	}
	b.WriteString("\n")
}

// writeRedirectDetails writes detailed redirect chain info.
func (*MarkdownFormatter) writeRedirectDetails(b *strings.Builder, results []checker.Result) {
	redirects := filterByStatus(results, checker.StatusRedirect)
	if len(redirects) == 0 {
		return
	}

	b.WriteString("### Redirect Details\n\n")
	for _, r := range redirects {
		fmt.Fprintf(b, "- **%s**\n", escapeMarkdown(r.Link.URL))
		if r.Link.Text != "" {
			fmt.Fprintf(b, "  - Text: %q\n", truncateText(r.Link.Text, 60))
		}
		fmt.Fprintf(b, "  - File: `%s:%d`\n", r.Link.FilePath, r.Link.Line)
		b.WriteString("  - Chain: ")
		b.WriteString(formatChainCodes(r))
		b.WriteString("\n")
		fmt.Fprintf(b, "  - Final: %s\n", r.FinalURL)
	}
	b.WriteString("\n")
}

// formatChainCodes formats redirect chain status codes as a string.
func formatChainCodes(r checker.Result) string {
	chain := make([]string, 0, len(r.RedirectChain)+1)
	for _, red := range r.RedirectChain {
		chain = append(chain, fmt.Sprintf("`%d`", red.StatusCode))
	}
	chain = append(chain, fmt.Sprintf("`%d`", r.FinalStatus))
	return strings.Join(chain, " → ")
}

// writeDuplicatesSection writes the duplicates section if any exist.
func (*MarkdownFormatter) writeDuplicatesSection(b *strings.Builder, results []checker.Result) {
	duplicates := filterByStatus(results, checker.StatusDuplicate)
	if len(duplicates) == 0 {
		return
	}

	fmt.Fprintf(b, "## Duplicates (%d)\n\n", len(duplicates))
	b.WriteString("| URL | File | Line | Original Location |\n")
	b.WriteString("|-----|------|------|-------------------|\n")
	for _, r := range duplicates {
		url := escapeMarkdown(truncateText(r.Link.URL, 60))
		origLoc := ""
		if r.DuplicateOf != nil {
			origLoc = fmt.Sprintf("%s:%d", r.DuplicateOf.Link.FilePath, r.DuplicateOf.Link.Line)
		}
		fmt.Fprintf(b, "| %s | %s | %d | %s |\n",
			url, r.Link.FilePath, r.Link.Line, origLoc)
	}
	b.WriteString("\n")
}

// writeIgnoredSection writes the ignored URLs section if any exist.
func (*MarkdownFormatter) writeIgnoredSection(b *strings.Builder, ignored []IgnoredURL) {
	if len(ignored) == 0 {
		return
	}

	fmt.Fprintf(b, "## Ignored URLs (%d)\n\n", len(ignored))
	b.WriteString("| URL | File | Line | Reason | Rule |\n")
	b.WriteString("|-----|------|------|--------|------|\n")
	for _, ig := range ignored {
		url := escapeMarkdown(truncateText(ig.URL, 60))
		fmt.Fprintf(b, "| %s | %s | %d | %s | `%s` |\n",
			url, ig.File, ig.Line, ig.Reason, ig.Rule)
	}
	b.WriteString("\n")
}

// formatStatusForMarkdown formats a result status for markdown display.
func formatStatusForMarkdown(r checker.Result) string {
	switch r.Status {
	case checker.StatusDead:
		if r.StatusCode > 0 {
			return fmt.Sprintf("`%d`", r.StatusCode)
		}
		return "DEAD"
	case checker.StatusError:
		return "ERROR"
	default:
		return r.Status.Label()
	}
}

// escapeMarkdown escapes special markdown characters in a string.
func escapeMarkdown(s string) string {
	// Escape pipe characters which break tables
	s = strings.ReplaceAll(s, "|", "\\|")
	// Escape backticks
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// truncateText shortens text to maxLen characters, adding "..." if truncated.
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
