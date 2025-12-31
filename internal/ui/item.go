package ui

import (
	"fmt"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
)

// ResultItem wraps a checker.Result to implement list.Item interface.
type ResultItem struct {
	Result checker.Result
}

// FilterValue returns the string used for filtering.
// Implements list.Item interface.
func (i ResultItem) FilterValue() string {
	return i.Result.Link.URL
}

// Title returns the main display text for the item.
// Implements list.DefaultItem interface.
func (i ResultItem) Title() string {
	// Show link text if available, otherwise URL
	if text := truncateText(i.Result.Link.Text, 60); text != "" {
		return fmt.Sprintf("%q", text)
	}
	return i.Result.Link.URL
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

// Description returns secondary text for the item.
// Implements list.DefaultItem interface.
func (i ResultItem) Description() string {
	r := i.Result

	// If we're showing link text in Title, show URL in description
	url := ""
	if r.Link.Text != "" {
		url = truncateText(r.Link.URL, 50) + " | "
	}

	switch r.Status {
	case checker.StatusAlive:
		return fmt.Sprintf("%s[%d] %s", url, r.StatusCode, r.Link.FilePath)

	case checker.StatusRedirect:
		finalURL := r.FinalURL
		if len(finalURL) > 40 {
			finalURL = finalURL[:37] + "..."
		}
		return fmt.Sprintf("%s→ %s | %s", url, finalURL, r.Link.FilePath)

	case checker.StatusBlocked:
		return fmt.Sprintf("%s403 Forbidden | %s", url, r.Link.FilePath)

	case checker.StatusDead:
		if r.StatusCode > 0 {
			return fmt.Sprintf("%s[%d] %s", url, r.StatusCode, r.Link.FilePath)
		}
		return fmt.Sprintf("%sDead | %s", url, r.Link.FilePath)

	case checker.StatusError:
		errMsg := r.Error
		if len(errMsg) > 30 {
			errMsg = errMsg[:27] + "..."
		}
		return fmt.Sprintf("%sError: %s | %s", url, errMsg, r.Link.FilePath)

	case checker.StatusDuplicate:
		if r.DuplicateOf != nil {
			return fmt.Sprintf("%sSame as %s | %s", url, r.DuplicateOf.Link.FilePath, r.Link.FilePath)
		}
		return fmt.Sprintf("%sDuplicate | %s", url, r.Link.FilePath)

	default:
		return url + r.Link.FilePath
	}
}

// DetailView returns an expanded detail view for the selected item.
func (i ResultItem) DetailView() string {
	r := i.Result
	var b strings.Builder

	b.WriteString("┌─ Details ─────────────────────────────────────────────────────────────\n")

	// Status line
	b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Status:"), StatusBadge(r.Status)))

	// Status-specific details
	switch r.Status {
	case checker.StatusAlive:
		b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("HTTP Code:"), r.StatusCode))

	case checker.StatusRedirect:
		b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("Original:"), r.StatusCode))
		b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Chain:"), formatRedirectChain(r)))
		b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Final URL:"), r.FinalURL))
		b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("Final Status:"), r.FinalStatus))
		b.WriteString("│\n")
		b.WriteString(fmt.Sprintf("│ %s\n", DetailNoteStyle.Render("Note: "+r.Status.Description())))

	case checker.StatusBlocked:
		b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("HTTP Code:"), r.StatusCode))
		b.WriteString("│\n")
		b.WriteString(fmt.Sprintf("│ %s\n", DetailNoteStyle.Render("Note: "+r.Status.Description())))

	case checker.StatusDead:
		if r.StatusCode > 0 {
			b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("HTTP Code:"), r.StatusCode))
		}
		if len(r.RedirectChain) > 0 {
			b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Chain:"), formatRedirectChain(r)))
			b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Final URL:"), r.FinalURL))
			b.WriteString(fmt.Sprintf("│ %s  %d\n", DetailLabelStyle.Render("Final Status:"), r.FinalStatus))
		}
		if r.Error != "" {
			b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Error:"), r.Error))
		}

	case checker.StatusError:
		b.WriteString(fmt.Sprintf("│ %s  %s\n", DetailLabelStyle.Render("Error:"), r.Error))

	case checker.StatusDuplicate:
		if r.DuplicateOf != nil {
			b.WriteString(fmt.Sprintf("│ %s  %s", DetailLabelStyle.Render("First found:"), r.DuplicateOf.Link.FilePath))
			if r.DuplicateOf.Link.Line > 0 {
				b.WriteString(fmt.Sprintf(" (line %d)", r.DuplicateOf.Link.Line))
			}
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("│ %s  %s\n",
				DetailLabelStyle.Render("Original status:"), StatusBadge(r.DuplicateOf.Status)))
		}
		b.WriteString("│\n")
		b.WriteString(fmt.Sprintf("│ %s\n", DetailNoteStyle.Render("Note: "+r.Status.Description())))
	}

	// Link text if available
	if text := truncateText(r.Link.Text, 60); text != "" {
		b.WriteString("│\n")
		b.WriteString(fmt.Sprintf("│ %s  %q\n", DetailLabelStyle.Render("Text:"), text))
	}

	// File location
	b.WriteString("│\n")
	b.WriteString(fmt.Sprintf("│ %s  %s", DetailLabelStyle.Render("File:"), r.Link.FilePath))
	if r.Link.Line > 0 {
		b.WriteString(fmt.Sprintf(" (line %d)", r.Link.Line))
	}
	b.WriteString("\n")

	b.WriteString("└────────────────────────────────────────────────────────────────────────\n")

	return b.String()
}

// formatRedirectChain formats the redirect chain for display.
func formatRedirectChain(r checker.Result) string {
	if len(r.RedirectChain) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(r.RedirectChain)+1)
	for _, red := range r.RedirectChain {
		parts = append(parts, fmt.Sprintf("%d", red.StatusCode))
	}
	parts = append(parts, fmt.Sprintf("%d", r.FinalStatus))
	return strings.Join(parts, " → ")
}

// ResultsToItems converts a slice of checker.Result to ResultItems.
func ResultsToItems(results []checker.Result) []ResultItem {
	items := make([]ResultItem, len(results))
	for i, r := range results {
		items[i] = ResultItem{Result: r}
	}
	return items
}
