package ui

import "github.com/charmbracelet/lipgloss"

// Color palette.
var (
	PrimaryColor   = lipgloss.Color("205") // Pink
	SecondaryColor = lipgloss.Color("241") // Gray
	SuccessColor   = lipgloss.Color("82")  // Green
	ErrorColor     = lipgloss.Color("196") // Red
	WarningColor   = lipgloss.Color("214") // Orange (for 3xx redirects)
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
)

// SpinnerStyle returns the style for the spinner.
func SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(PrimaryColor)
}

// Badge styles for status codes.
var (
	BadgeError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(ErrorColor).
			Padding(0, 1)

	Badge4xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(ErrorColor).
			Padding(0, 1)

	Badge5xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("161")). // Darker red
			Padding(0, 1)

	Badge3xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(WarningColor).
			Padding(0, 1)
)

// StatusBadge returns a styled badge for the given status code.
func StatusBadge(statusCode int, hasError bool) string {
	if hasError {
		return BadgeError.Render("ERR")
	}

	switch {
	case statusCode >= 500:
		return Badge5xx.Render("5xx")
	case statusCode >= 400:
		return Badge4xx.Render("4xx")
	case statusCode >= 300:
		return Badge3xx.Render("3xx")
	default:
		return BadgeError.Render("???")
	}
}
