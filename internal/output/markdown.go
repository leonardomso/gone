package output

import (
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
)

// MarkdownFormatter formats reports as Markdown.
type MarkdownFormatter struct{}

// Format implements Formatter.
func (*MarkdownFormatter) Format(report *Report) ([]byte, error) {
	// Pre-grow builder: estimate ~200 bytes per result + ~500 bytes header
	var b strings.Builder
	b.Grow(len(report.Results)*200 + 500)

	// Header
	b.WriteString("# Gone Link Check Report\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s  \n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("**Files Scanned:** %d  \n", len(report.Files)))
	b.WriteString(fmt.Sprintf("**Total Links:** %d  \n", report.TotalLinks))
	b.WriteString(fmt.Sprintf("**Unique URLs:** %d\n\n", report.UniqueURLs))

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Status | Count |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Alive | %d |\n", report.Summary.Alive))
	b.WriteString(fmt.Sprintf("| Warnings | %d |\n", report.Summary.WarningsCount()))
	b.WriteString(fmt.Sprintf("| Dead | %d |\n", report.Summary.Dead+report.Summary.Errors))
	b.WriteString(fmt.Sprintf("| Duplicates | %d |\n", report.Summary.Duplicates))
	if len(report.Ignored) > 0 {
		b.WriteString(fmt.Sprintf("| Ignored | %d |\n", len(report.Ignored)))
	}
	b.WriteString("\n")

	// Dead Links section
	deadLinks := filterByStatus(report.Results, checker.StatusDead, checker.StatusError)
	if len(deadLinks) > 0 {
		b.WriteString(fmt.Sprintf("## Dead Links (%d)\n\n", len(deadLinks)))
		b.WriteString("| Status | URL | Text | File | Line |\n")
		b.WriteString("|--------|-----|------|------|------|\n")
		for _, r := range deadLinks {
			status := formatStatusForMarkdown(r)
			text := escapeMarkdown(truncateText(r.Link.Text, 40))
			url := escapeMarkdown(truncateText(r.Link.URL, 60))
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
				status, url, text, r.Link.FilePath, r.Link.Line))
		}
		b.WriteString("\n")

		// Detailed dead links with redirect chains
		b.WriteString("### Details\n\n")
		for _, r := range deadLinks {
			b.WriteString(fmt.Sprintf("#### %s\n\n", escapeMarkdown(r.Link.URL)))
			if r.Link.Text != "" {
				b.WriteString(fmt.Sprintf("- **Text:** %q\n", r.Link.Text))
			}
			b.WriteString(fmt.Sprintf("- **File:** `%s:%d`\n", r.Link.FilePath, r.Link.Line))
			b.WriteString(fmt.Sprintf("- **Status:** %s\n", formatStatusForMarkdown(r)))

			if len(r.RedirectChain) > 0 {
				b.WriteString("- **Redirect Chain:**\n")
				for i, red := range r.RedirectChain {
					b.WriteString(fmt.Sprintf("  %d. `%d` → %s\n", i+1, red.StatusCode, red.URL))
				}
				b.WriteString(fmt.Sprintf("  Final: `%d` → %s\n", r.FinalStatus, r.FinalURL))
			}

			if r.Error != "" {
				b.WriteString(fmt.Sprintf("- **Error:** %s\n", r.Error))
			}
			b.WriteString("\n")
		}
	}

	// Warnings section
	warnings := filterByStatus(report.Results, checker.StatusRedirect, checker.StatusBlocked)
	if len(warnings) > 0 {
		b.WriteString(fmt.Sprintf("## Warnings (%d)\n\n", len(warnings)))
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
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d |\n",
				issue, url, text, finalURL, r.Link.FilePath, r.Link.Line))
		}
		b.WriteString("\n")

		// Detailed warnings with full redirect chains
		b.WriteString("### Redirect Details\n\n")
		redirects := filterByStatus(report.Results, checker.StatusRedirect)
		for _, r := range redirects {
			b.WriteString(fmt.Sprintf("- **%s**\n", escapeMarkdown(r.Link.URL)))
			if r.Link.Text != "" {
				b.WriteString(fmt.Sprintf("  - Text: %q\n", truncateText(r.Link.Text, 60)))
			}
			b.WriteString(fmt.Sprintf("  - File: `%s:%d`\n", r.Link.FilePath, r.Link.Line))
			b.WriteString("  - Chain: ")
			chain := make([]string, 0, len(r.RedirectChain)+1)
			for _, red := range r.RedirectChain {
				chain = append(chain, fmt.Sprintf("`%d`", red.StatusCode))
			}
			chain = append(chain, fmt.Sprintf("`%d`", r.FinalStatus))
			b.WriteString(strings.Join(chain, " → "))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  - Final: %s\n", r.FinalURL))
		}
		b.WriteString("\n")
	}

	// Duplicates section
	duplicates := filterByStatus(report.Results, checker.StatusDuplicate)
	if len(duplicates) > 0 {
		b.WriteString(fmt.Sprintf("## Duplicates (%d)\n\n", len(duplicates)))
		b.WriteString("| URL | File | Line | Original Location |\n")
		b.WriteString("|-----|------|------|-------------------|\n")
		for _, r := range duplicates {
			url := escapeMarkdown(truncateText(r.Link.URL, 60))
			origLoc := ""
			if r.DuplicateOf != nil {
				origLoc = fmt.Sprintf("%s:%d", r.DuplicateOf.Link.FilePath, r.DuplicateOf.Link.Line)
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n",
				url, r.Link.FilePath, r.Link.Line, origLoc))
		}
		b.WriteString("\n")
	}

	// Ignored section
	if len(report.Ignored) > 0 {
		b.WriteString(fmt.Sprintf("## Ignored URLs (%d)\n\n", len(report.Ignored)))
		b.WriteString("| URL | File | Line | Reason | Rule |\n")
		b.WriteString("|-----|------|------|--------|------|\n")
		for _, ig := range report.Ignored {
			url := escapeMarkdown(truncateText(ig.URL, 60))
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %s | `%s` |\n",
				url, ig.File, ig.Line, ig.Reason, ig.Rule))
		}
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
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
