package ui

import (
	"gone/internal/checker"

	"github.com/charmbracelet/lipgloss"
)

// Color palette.
var (
	PrimaryColor   = lipgloss.Color("205") // Pink
	SecondaryColor = lipgloss.Color("241") // Gray
	SuccessColor   = lipgloss.Color("82")  // Green
	ErrorColor     = lipgloss.Color("196") // Red
	WarningColor   = lipgloss.Color("214") // Orange
	BlockedColor   = lipgloss.Color("208") // Dark orange
	DuplicateColor = lipgloss.Color("139") // Purple/gray
	MutedColor     = lipgloss.Color("245") // Dimmed text
)

// Text styles.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginBottom(1)

	StatusStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	BlockedStyle = lipgloss.NewStyle().
			Foreground(BlockedColor)

	DuplicateStyle = lipgloss.NewStyle().
			Foreground(DuplicateColor)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	HelpStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			MarginTop(1)

	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Detail panel styles.
	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(0, 1).
			MarginTop(1)

	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor).
				Bold(true)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	DetailNoteStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)
)

// SpinnerStyle returns the style for the spinner.
func SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(PrimaryColor)
}

// Badge styles for status codes.
var (
	BadgeAlive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(SuccessColor).
			Padding(0, 1).
			Bold(true)

	BadgeRedirect = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(WarningColor).
			Padding(0, 1).
			Bold(true)

	BadgeBlocked = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(BlockedColor).
			Padding(0, 1).
			Bold(true)

	BadgeDead = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(ErrorColor).
			Padding(0, 1).
			Bold(true)

	BadgeError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("161")). // Darker red
			Padding(0, 1).
			Bold(true)

	BadgeDuplicate = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(DuplicateColor).
			Padding(0, 1).
			Bold(true)
)

// StatusBadge returns a styled badge for the given link status.
func StatusBadge(status checker.LinkStatus) string {
	switch status {
	case checker.StatusAlive:
		return BadgeAlive.Render("OK")
	case checker.StatusRedirect:
		return BadgeRedirect.Render("REDIRECT")
	case checker.StatusBlocked:
		return BadgeBlocked.Render("BLOCKED")
	case checker.StatusDead:
		return BadgeDead.Render("DEAD")
	case checker.StatusError:
		return BadgeError.Render("ERROR")
	case checker.StatusDuplicate:
		return BadgeDuplicate.Render("DUPLICATE")
	default:
		return BadgeError.Render("???")
	}
}

// StatusTextStyle returns the appropriate style for a link status.
func StatusTextStyle(status checker.LinkStatus) lipgloss.Style {
	switch status {
	case checker.StatusAlive:
		return SuccessStyle
	case checker.StatusRedirect:
		return WarningStyle
	case checker.StatusBlocked:
		return BlockedStyle
	case checker.StatusDead, checker.StatusError:
		return ErrorStyle
	case checker.StatusDuplicate:
		return DuplicateStyle
	default:
		return NormalStyle
	}
}

// SummaryAlive returns a styled count for alive links.
func SummaryAlive(count int) string {
	return SuccessStyle.Render("✓ " + formatCount(count, "alive"))
}

// SummaryWarnings returns a styled count for warning links.
func SummaryWarnings(count int) string {
	return WarningStyle.Render("⚠ " + formatCount(count, "warning", "warnings"))
}

// SummaryDead returns a styled count for dead links.
func SummaryDead(count int) string {
	return ErrorStyle.Render("✗ " + formatCount(count, "dead"))
}

// SummaryDuplicates returns a styled count for duplicate links.
func SummaryDuplicates(count int) string {
	return DuplicateStyle.Render("◈ " + formatCount(count, "duplicate", "duplicates"))
}

func formatCount(count int, singular string, pluralOpt ...string) string {
	plural := singular
	if len(pluralOpt) > 0 {
		plural = pluralOpt[0]
	}
	if count == 1 {
		return "1 " + singular
	}
	return lipgloss.NewStyle().Render(string(rune('0'+count%10))) + " " + plural
}
