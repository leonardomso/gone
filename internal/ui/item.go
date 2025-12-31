package ui

import (
	"fmt"
	"strings"

	"gone/internal/checker"
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
	return i.Result.Link.URL
}

// Description returns secondary text for the item.
// Implements list.DefaultItem interface.
func (i ResultItem) Description() string {
	r := i.Result
	switch r.Status {
	case checker.StatusAlive:
		return fmt.Sprintf("[%d] %s", r.StatusCode, r.Link.FilePath)

	case checker.StatusRedirect:
		finalURL := r.FinalURL
		if len(finalURL) > 50 {
			finalURL = finalURL[:47] + "..."
		}
		return fmt.Sprintf("→ %s | %s", finalURL, r.Link.FilePath)

	case checker.StatusBlocked:
		return fmt.Sprintf("403 Forbidden | %s", r.Link.FilePath)

	case checker.StatusDead:
		if r.StatusCode > 0 {
			return fmt.Sprintf("[%d] %s", r.StatusCode, r.Link.FilePath)
		}
		return fmt.Sprintf("Dead | %s", r.Link.FilePath)

	case checker.StatusError:
		errMsg := r.Error
		if len(errMsg) > 40 {
			errMsg = errMsg[:37] + "..."
		}
		return fmt.Sprintf("Error: %s | %s", errMsg, r.Link.FilePath)

	case checker.StatusDuplicate:
		if r.DuplicateOf != nil {
			return fmt.Sprintf("Same as %s | %s", r.DuplicateOf.Link.FilePath, r.Link.FilePath)
		}
		return fmt.Sprintf("Duplicate | %s", r.Link.FilePath)

	default:
		return r.Link.FilePath
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
