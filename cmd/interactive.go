package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"gone/internal/checker"
	"gone/internal/parser"
	"gone/internal/scanner"
)

// interactiveCmd represents the interactive command
var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive TUI for dead link detection",
	Long: `Launch an interactive terminal UI to scan for dead links.

Navigate through results, see progress in real-time, and 
select which dead links to review.

Controls:
  ↑/↓ or j/k    Navigate through results
  q             Quit`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create and run the Bubble Tea program
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running interactive mode: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

// =============================================================================
// STYLES - Define all our styles in one place (like CSS)
// =============================================================================

var (
	// Colors
	primaryColor   = lipgloss.Color("205") // Pink
	secondaryColor = lipgloss.Color("241") // Gray
	successColor   = lipgloss.Color("82")  // Green
	errorColor     = lipgloss.Color("196") // Red
	warningColor   = lipgloss.Color("214") // Orange

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	selectedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginTop(1)
)

// =============================================================================
// STATE MACHINE - Different phases of our app
// =============================================================================

type appState int

const (
	stateScanning   appState = iota // Finding markdown files
	stateExtracting                 // Extracting links from files
	stateChecking                   // Making HTTP requests
	stateDone                       // Showing results
)

// =============================================================================
// MESSAGES - Events that can happen in our app
// =============================================================================

// filesFoundMsg is sent when we finish scanning for files
type filesFoundMsg struct {
	files []string
	err   error
}

// linksExtractedMsg is sent when we finish extracting links
type linksExtractedMsg struct {
	links []parser.Link
	err   error
}

// linkCheckedMsg is sent each time a link check completes
type linkCheckedMsg struct {
	result checker.Result
}

// allChecksCompleteMsg is sent when all checks are done
type allChecksCompleteMsg struct{}

// =============================================================================
// MODEL - Our application state
// =============================================================================

type model struct {
	state     appState         // Current phase
	spinner   spinner.Model    // Loading spinner
	files     []string         // Markdown files found
	links     []parser.Link    // Links extracted
	results   []checker.Result // Check results (accumulates)
	deadLinks []checker.Result // Only the dead ones
	checked   int              // How many links checked so far
	cursor    int              // Currently selected result
	quitting  bool
	err       error
}

// initialModel creates the starting state
func initialModel() model {
	// Create a spinner with a nice style
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return model{
		state:   stateScanning,
		spinner: s,
	}
}

// =============================================================================
// COMMANDS - Functions that perform side effects and return messages
// =============================================================================

// scanFiles is a command that scans for markdown files
func scanFiles() tea.Msg {
	files, err := scanner.FindMarkdownFiles(".")
	return filesFoundMsg{files: files, err: err}
}

// extractLinks is a command that extracts links from files
func extractLinks(files []string) tea.Cmd {
	return func() tea.Msg {
		links, err := parser.ExtractLinksFromMultipleFiles(files)
		return linksExtractedMsg{links: links, err: err}
	}
}

// checkLinksCmd starts the async link checking
func checkLinksCmd(links []parser.Link) tea.Cmd {
	return func() tea.Msg {
		// Get the channel of results
		resultsChan := checker.CheckLinksAsync(links)

		// Return the first result (subsequent results will be fetched by waitForResult)
		result, ok := <-resultsChan
		if !ok {
			return allChecksCompleteMsg{}
		}
		return linkCheckedMsg{result: result}
	}
}

// waitForResult creates a command that waits for the next result
// We store the channel in a package-level variable (not ideal but simple)
var resultsChan <-chan checker.Result

func startCheckingLinks(links []parser.Link) tea.Cmd {
	return func() tea.Msg {
		resultsChan = checker.CheckLinksAsync(links)
		return waitForNextResult()
	}
}

func waitForNextResult() tea.Msg {
	result, ok := <-resultsChan
	if !ok {
		return allChecksCompleteMsg{}
	}
	return linkCheckedMsg{result: result}
}

func waitForResultCmd() tea.Cmd {
	return func() tea.Msg {
		return waitForNextResult()
	}
}

// =============================================================================
// INIT - Called once at startup
// =============================================================================

func (m model) Init() tea.Cmd {
	// Start the spinner AND scan for files
	return tea.Batch(m.spinner.Tick, scanFiles)
}

// =============================================================================
// UPDATE - Handle all events
// =============================================================================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Key presses
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.deadLinks)-1 {
				m.cursor++
			}
		}

	// Spinner tick
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// Files found
	case filesFoundMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateDone
			return m, nil
		}
		m.files = msg.files
		m.state = stateExtracting
		return m, extractLinks(msg.files)

	// Links extracted
	case linksExtractedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateDone
			return m, nil
		}
		m.links = msg.links
		if len(m.links) == 0 {
			m.state = stateDone
			return m, nil
		}
		m.state = stateChecking
		return m, startCheckingLinks(m.links)

	// Single link checked
	case linkCheckedMsg:
		m.results = append(m.results, msg.result)
		m.checked++
		if !msg.result.IsAlive {
			m.deadLinks = append(m.deadLinks, msg.result)
		}
		// Wait for next result
		return m, waitForResultCmd()

	// All checks complete
	case allChecksCompleteMsg:
		m.state = stateDone
		return m, nil
	}

	return m, nil
}

// =============================================================================
// VIEW - Render the UI
// =============================================================================

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var s string

	// Title
	s += titleStyle.Render("Gone - Dead Link Detector")
	s += "\n\n"

	// Show error if any
	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		s += "\n"
		s += helpStyle.Render("Press q to quit")
		return s
	}

	// State-specific content
	switch m.state {
	case stateScanning:
		s += m.spinner.View() + " Scanning for markdown files..."

	case stateExtracting:
		s += m.spinner.View() + fmt.Sprintf(" Found %d file(s), extracting links...", len(m.files))

	case stateChecking:
		s += m.spinner.View() + fmt.Sprintf(" Checking links... %d/%d", m.checked, len(m.links))
		s += "\n"
		if len(m.deadLinks) > 0 {
			s += errorStyle.Render(fmt.Sprintf("  Dead links found: %d", len(m.deadLinks)))
		}

	case stateDone:
		s += m.renderResults()
	}

	// Help text
	s += helpStyle.Render("\nPress q to quit")
	if m.state == stateDone && len(m.deadLinks) > 0 {
		s += helpStyle.Render(" | ↑/↓ to navigate")
	}

	return s
}

// renderResults shows the final results
func (m model) renderResults() string {
	var s string

	s += fmt.Sprintf("Scanned %d file(s), checked %d link(s)\n\n", len(m.files), len(m.links))

	if len(m.deadLinks) == 0 {
		s += successStyle.Render("All links are alive!")
		return s
	}

	s += errorStyle.Render(fmt.Sprintf("Found %d dead link(s):\n\n", len(m.deadLinks)))

	// Show dead links with cursor
	for i, r := range m.deadLinks {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		// Format the status
		status := fmt.Sprintf("[%d]", r.StatusCode)
		if r.Error != "" {
			status = "[ERR]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, status, r.Link.URL)
		s += style.Render(line) + "\n"

		// Show file path for selected item
		if i == m.cursor {
			s += statusStyle.Render(fmt.Sprintf("      File: %s", r.Link.FilePath)) + "\n"
			if r.Error != "" {
				s += statusStyle.Render(fmt.Sprintf("      Error: %s", r.Error)) + "\n"
			}
		}
	}

	return s
}
