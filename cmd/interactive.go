package cmd

import (
	"context"
	"fmt"
	"os"

	"gone/internal/checker"
	"gone/internal/parser"
	"gone/internal/scanner"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// interactiveCmd represents the interactive command.
var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive TUI for dead link detection",
	Long: `Launch an interactive terminal UI to scan for dead links.

Navigate through results, see progress in real-time, and 
select which dead links to review.

Controls:
  ↑/↓ or j/k    Navigate through results
  q             Quit`,
	Run: func(_ *cobra.Command, _ []string) {
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running interactive mode: %v\n", err)
			os.Exit(1) //nolint:revive // deep-exit is acceptable for CLI entry points
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

// =============================================================================
// STYLES
// =============================================================================

var (
	primaryColor   = lipgloss.Color("205") // Pink
	secondaryColor = lipgloss.Color("241") // Gray
	successColor   = lipgloss.Color("82")  // Green
	errorColor     = lipgloss.Color("196") // Red

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
// STATE MACHINE
// =============================================================================

type appState int

const (
	stateScanning   appState = iota // Finding markdown files
	stateExtracting                 // Extracting links from files
	stateChecking                   // Making HTTP requests
	stateDone                       // Showing results
)

// =============================================================================
// MESSAGES
// =============================================================================

type filesFoundMsg struct {
	files []string
	err   error
}

type linksExtractedMsg struct {
	links []checker.Link
	err   error
}

type linkCheckedMsg struct {
	result checker.Result
}

type allChecksCompleteMsg struct{}

// =============================================================================
// MODEL
// =============================================================================

type model struct {
	state       appState
	spinner     spinner.Model
	files       []string
	links       []checker.Link
	results     []checker.Result
	deadLinks   []checker.Result
	checked     int
	cursor      int
	quitting    bool
	err         error
	cancelFunc  context.CancelFunc    // For canceling ongoing checks
	resultsChan <-chan checker.Result // Channel for streaming results
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return model{
		state:   stateScanning,
		spinner: s,
	}
}

// =============================================================================
// COMMANDS
// =============================================================================

func scanFilesCmd() tea.Msg {
	files, err := scanner.FindMarkdownFiles(".")
	return filesFoundMsg{files: files, err: err}
}

func extractLinksCmd(files []string) tea.Cmd {
	return func() tea.Msg {
		parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
		if err != nil {
			return linksExtractedMsg{err: err}
		}

		// Convert parser.Link to checker.Link
		links := make([]checker.Link, len(parserLinks))
		for i, pl := range parserLinks {
			links[i] = checker.Link{
				URL:      pl.URL,
				FilePath: pl.FilePath,
				Line:     pl.Line,
			}
		}
		return linksExtractedMsg{links: links}
	}
}

// startCheckingCmd initializes the checker and returns the first result.
func (m *model) startCheckingCmd(links []checker.Link) tea.Cmd {
	return func() tea.Msg {
		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		// Create checker with default options
		opts := checker.DefaultOptions()
		c := checker.New(opts)

		// Start checking and store the channel
		m.resultsChan = c.Check(ctx, links)

		// Get the first result
		result, ok := <-m.resultsChan
		if !ok {
			return allChecksCompleteMsg{}
		}
		return linkCheckedMsg{result: result}
	}
}

// waitForNextResultCmd waits for the next result from the channel.
func (m *model) waitForNextResultCmd() tea.Cmd {
	return func() tea.Msg {
		if m.resultsChan == nil {
			return allChecksCompleteMsg{}
		}

		result, ok := <-m.resultsChan
		if !ok {
			return allChecksCompleteMsg{}
		}
		return linkCheckedMsg{result: result}
	}
}

// =============================================================================
// INIT
// =============================================================================

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, scanFilesCmd)
}

// =============================================================================
// UPDATE
// =============================================================================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Cancel ongoing checks if any
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case filesFoundMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateDone
			return m, nil
		}
		m.files = msg.files
		m.state = stateExtracting
		return m, extractLinksCmd(msg.files)

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
		cmd := m.startCheckingCmd(m.links)
		return m, cmd

	case linkCheckedMsg:
		m.results = append(m.results, msg.result)
		m.checked++
		if !msg.result.IsAlive {
			m.deadLinks = append(m.deadLinks, msg.result)
		}
		cmd := m.waitForNextResultCmd()
		return m, cmd

	case allChecksCompleteMsg:
		m.state = stateDone
		m.resultsChan = nil
		return m, nil
	}

	return m, nil
}

// =============================================================================
// VIEW
// =============================================================================

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var s string

	s += titleStyle.Render("Gone - Dead Link Detector")
	s += "\n\n"

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		s += "\n"
		s += helpStyle.Render("Press q to quit")
		return s
	}

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

	s += helpStyle.Render("\nPress q to quit")
	if m.state == stateDone && len(m.deadLinks) > 0 {
		s += helpStyle.Render(" | ↑/↓ to navigate")
	}

	return s
}

func (m model) renderResults() string {
	var s string

	s += fmt.Sprintf("Scanned %d file(s), checked %d link(s)\n\n", len(m.files), len(m.links))

	if len(m.deadLinks) == 0 {
		s += successStyle.Render("All links are alive!")
		return s
	}

	s += errorStyle.Render(fmt.Sprintf("Found %d dead link(s):\n\n", len(m.deadLinks)))

	for i, r := range m.deadLinks {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		status := fmt.Sprintf("[%d]", r.StatusCode)
		if r.Error != "" {
			status = "[ERR]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, status, r.Link.URL)
		s += style.Render(line) + "\n"

		if i == m.cursor {
			s += statusStyle.Render(fmt.Sprintf("      File: %s", r.Link.FilePath)) + "\n"
			if r.Error != "" {
				s += statusStyle.Render(fmt.Sprintf("      Error: %s", r.Error)) + "\n"
			}
		}
	}

	return s
}
