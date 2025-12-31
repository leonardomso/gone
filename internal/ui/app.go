package ui

import (
	"fmt"

	"gone/internal/checker"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// STATE MACHINE
// =============================================================================

type appState int

const (
	stateScanning   appState = iota // Finding markdown files
	stateExtracting                 // Extracting links from files
	stateChecking                   // Making HTTP requests
	stateResults                    // Showing results (list view)
)

// =============================================================================
// FILTER TYPES
// =============================================================================

type filterType int

const (
	filterAll filterType = iota
	filterErrors
	filter4xx
	filter5xx
	filter3xx
)

func (f filterType) String() string {
	switch f {
	case filterAll:
		return "All"
	case filterErrors:
		return "Errors"
	case filter4xx:
		return "4xx"
	case filter5xx:
		return "5xx"
	case filter3xx:
		return "3xx"
	default:
		return "Unknown"
	}
}

func (f filterType) Next() filterType {
	return (f + 1) % 5
}

// =============================================================================
// MODEL
// =============================================================================

// Model is the main application model.
type Model struct {
	// State
	state    appState
	quitting bool
	err      error

	// Data
	files     []string
	links     []checker.Link
	results   []checker.Result
	deadLinks []checker.Result

	// Progress tracking
	checked int

	// Filter
	filter filterType

	// Components
	spinner spinner.Model
	list    list.Model
	help    help.Model
	keys    KeyMap

	// Checker state (for async operations)
	checkerState CheckerState

	// UI state
	width    int
	height   int
	showHelp bool

	// Config
	path string
}

// New creates and returns a new Model for the given path.
func New(path string) Model {
	if path == "" {
		path = "."
	}
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle()

	// Initialize help
	h := help.New()

	// Initialize keys
	k := DefaultKeyMap()

	// Initialize list with empty items (will be populated later)
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = SelectedStyle
	delegate.Styles.SelectedDesc = StatusStyle

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Dead Links"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We use our own help
	l.Styles.Title = TitleStyle

	return Model{
		state:   stateScanning,
		spinner: s,
		list:    l,
		help:    h,
		keys:    k,
		filter:  filterAll,
		path:    path,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, ScanFilesCmdWithPath(m.path))
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-6) // Reserve space for header/footer
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case FilesFoundMsg:
		return m.handleFilesFound(msg)

	case LinksExtractedMsg:
		return m.handleLinksExtracted(msg)

	case LinkCheckedMsg:
		return m.handleLinkChecked(msg)

	case AllChecksCompleteMsg:
		return m.handleAllChecksComplete()
	}

	// Pass other messages to list if in results state
	if m.state == stateResults {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys that work in any state
	if key.Matches(msg, m.keys.Quit) {
		if m.checkerState.CancelFunc != nil {
			m.checkerState.CancelFunc()
		}
		m.quitting = true
		return m, tea.Quit
	}

	if key.Matches(msg, m.keys.Help) {
		m.showHelp = !m.showHelp
		return m, nil
	}

	// State-specific keys
	if m.state == stateResults {
		if key.Matches(msg, m.keys.Filter) {
			m.filter = m.filter.Next()
			m.updateListItems()
			return m, nil
		}

		// Pass navigation keys to list
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleFilesFound(msg FilesFoundMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.err = msg.Err
		m.state = stateResults
		return m, nil
	}
	m.files = msg.Files
	m.state = stateExtracting
	return m, ExtractLinksCmd(msg.Files)
}

func (m *Model) handleLinksExtracted(msg LinksExtractedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.err = msg.Err
		m.state = stateResults
		return m, nil
	}
	m.links = msg.Links
	if len(m.links) == 0 {
		m.state = stateResults
		return m, nil
	}
	m.state = stateChecking
	return m, StartCheckingCmd(m.links, &m.checkerState)
}

func (m *Model) handleLinkChecked(msg LinkCheckedMsg) (tea.Model, tea.Cmd) {
	m.results = append(m.results, msg.Result)
	m.checked++
	if !msg.Result.IsAlive {
		m.deadLinks = append(m.deadLinks, msg.Result)
	}
	return m, WaitForNextResultCmd(&m.checkerState)
}

func (m *Model) handleAllChecksComplete() (tea.Model, tea.Cmd) {
	m.state = stateResults
	m.checkerState.ResultsChan = nil
	m.updateListItems()
	return m, nil
}

// updateListItems updates the list with filtered dead links.
func (m *Model) updateListItems() {
	filtered := m.filterDeadLinks()
	items := make([]list.Item, len(filtered))
	for i, r := range filtered {
		items[i] = ResultItem{Result: r}
	}
	m.list.SetItems(items)
}

// filterDeadLinks returns dead links based on current filter.
func (m *Model) filterDeadLinks() []checker.Result {
	if m.filter == filterAll {
		return m.deadLinks
	}

	var filtered []checker.Result
	for _, r := range m.deadLinks {
		switch m.filter {
		case filterErrors:
			if r.Error != "" {
				filtered = append(filtered, r)
			}
		case filter4xx:
			if r.StatusCode >= 400 && r.StatusCode < 500 {
				filtered = append(filtered, r)
			}
		case filter5xx:
			if r.StatusCode >= 500 {
				filtered = append(filtered, r)
			}
		case filter3xx:
			if r.StatusCode >= 300 && r.StatusCode < 400 {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// =============================================================================
// VIEW
// =============================================================================

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var s string

	// Header
	s += TitleStyle.Render("Gone - Dead Link Detector")
	s += "\n\n"

	// Error state
	if m.err != nil {
		s += ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		s += "\n"
		s += HelpStyle.Render("Press q to quit")
		return s
	}

	// State-specific view
	switch m.state {
	case stateScanning:
		s += m.spinner.View() + " Scanning for markdown files..."

	case stateExtracting:
		s += m.spinner.View() + fmt.Sprintf(" Found %d file(s), extracting links...", len(m.files))

	case stateChecking:
		s += m.spinner.View() + fmt.Sprintf(" Checking links... %d/%d", m.checked, len(m.links))
		s += "\n"
		if len(m.deadLinks) > 0 {
			s += ErrorStyle.Render(fmt.Sprintf("  Dead links found: %d", len(m.deadLinks)))
		}

	case stateResults:
		s += m.renderResults()
	}

	// Help
	if m.showHelp {
		s += "\n\n" + m.help.View(m.keys)
	} else {
		s += "\n\n" + m.renderShortHelp()
	}

	return s
}

func (m Model) renderResults() string {
	var s string

	// Summary
	s += fmt.Sprintf("Scanned %d file(s), checked %d link(s)\n", len(m.files), len(m.links))

	if len(m.deadLinks) == 0 {
		s += "\n" + SuccessStyle.Render("All links are alive!")
		return s
	}

	// Filter indicator
	filteredCount := len(m.filterDeadLinks())
	s += fmt.Sprintf("Filter: %s (%d/%d)\n\n",
		SelectedStyle.Render(m.filter.String()),
		filteredCount,
		len(m.deadLinks))

	// List view
	s += m.list.View()

	return s
}

func (Model) renderShortHelp() string {
	return HelpStyle.Render("↑/↓ navigate • f filter • ? help • q quit")
}
