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
	filterAll        filterType = iota // All non-alive (warnings + dead + duplicates)
	filterWarnings                     // Redirects + Blocked
	filterDead                         // Dead + Errors
	filterDuplicates                   // Duplicates only
)

const filterCount = 4

func (f filterType) String() string {
	switch f {
	case filterAll:
		return "All Issues"
	case filterWarnings:
		return "Warnings"
	case filterDead:
		return "Dead"
	case filterDuplicates:
		return "Duplicates"
	default:
		return "Unknown"
	}
}

func (f filterType) Next() filterType {
	return (f + 1) % filterCount
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
	files   []string
	links   []checker.Link
	results []checker.Result

	// Categorized results
	aliveLinks     []checker.Result
	warningLinks   []checker.Result
	deadLinks      []checker.Result
	duplicateLinks []checker.Result

	// Progress tracking
	checked    int
	uniqueURLs int
	duplicates int

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
	l.Title = "Link Check Results"
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
		// Reserve space for header, summary, and detail panel
		listHeight := max(msg.Height-12, 5)
		m.list.SetSize(msg.Width, listHeight)
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
	m.uniqueURLs = msg.UniqueURLs
	m.duplicates = msg.Duplicates

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

	// Categorize the result
	switch msg.Result.Status {
	case checker.StatusAlive:
		m.aliveLinks = append(m.aliveLinks, msg.Result)
	case checker.StatusRedirect, checker.StatusBlocked:
		m.warningLinks = append(m.warningLinks, msg.Result)
	case checker.StatusDead, checker.StatusError:
		m.deadLinks = append(m.deadLinks, msg.Result)
	case checker.StatusDuplicate:
		m.duplicateLinks = append(m.duplicateLinks, msg.Result)
	}

	return m, WaitForNextResultCmd(&m.checkerState)
}

func (m *Model) handleAllChecksComplete() (tea.Model, tea.Cmd) {
	m.state = stateResults
	m.checkerState.ResultsChan = nil
	m.updateListItems()
	return m, nil
}

// updateListItems updates the list with filtered results.
func (m *Model) updateListItems() {
	filtered := m.getFilteredResults()
	items := make([]list.Item, len(filtered))
	for i, r := range filtered {
		items[i] = ResultItem{Result: r}
	}
	m.list.SetItems(items)
}

// getFilteredResults returns results based on current filter.
func (m *Model) getFilteredResults() []checker.Result {
	switch m.filter {
	case filterAll:
		// All non-alive: warnings + dead + duplicates
		var all []checker.Result
		all = append(all, m.warningLinks...)
		all = append(all, m.deadLinks...)
		all = append(all, m.duplicateLinks...)
		return all
	case filterWarnings:
		return m.warningLinks
	case filterDead:
		return m.deadLinks
	case filterDuplicates:
		return m.duplicateLinks
	default:
		return nil
	}
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
		s += m.renderCheckingProgress()

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

func (m Model) renderCheckingProgress() string {
	var s string

	if m.duplicates > 0 {
		s += m.spinner.View() + fmt.Sprintf(" Checking %d unique URLs... %d/%d",
			m.uniqueURLs, m.checked, len(m.links))
	} else {
		s += m.spinner.View() + fmt.Sprintf(" Checking links... %d/%d", m.checked, len(m.links))
	}
	s += "\n\n"

	// Live category counts
	s += fmt.Sprintf("  %s  %s  %s",
		SuccessStyle.Render(fmt.Sprintf("✓ %d alive", len(m.aliveLinks))),
		WarningStyle.Render(fmt.Sprintf("⚠ %d warnings", len(m.warningLinks))),
		ErrorStyle.Render(fmt.Sprintf("✗ %d dead", len(m.deadLinks))))

	return s
}

func (m Model) renderResults() string {
	var s string

	// Summary line
	s += fmt.Sprintf("Scanned %d file(s), checked %d link(s)", len(m.files), len(m.links))
	if m.duplicates > 0 {
		s += fmt.Sprintf(" (%d unique)", m.uniqueURLs)
	}
	s += "\n\n"

	// Category summary
	s += fmt.Sprintf("%s | %s | %s | %s\n\n",
		SuccessStyle.Render(fmt.Sprintf("✓ %d alive", len(m.aliveLinks))),
		WarningStyle.Render(fmt.Sprintf("⚠ %d warnings", len(m.warningLinks))),
		ErrorStyle.Render(fmt.Sprintf("✗ %d dead", len(m.deadLinks))),
		DuplicateStyle.Render(fmt.Sprintf("◈ %d duplicates", len(m.duplicateLinks))))

	// Check if everything is alive
	if len(m.warningLinks) == 0 && len(m.deadLinks) == 0 && len(m.duplicateLinks) == 0 {
		s += SuccessStyle.Render("All links are alive!")
		return s
	}

	// Filter indicator
	filteredCount := len(m.getFilteredResults())
	totalIssues := len(m.warningLinks) + len(m.deadLinks) + len(m.duplicateLinks)
	s += fmt.Sprintf("Filter: %s (%d/%d)\n\n",
		SelectedStyle.Render(m.filter.String()),
		filteredCount,
		totalIssues)

	// List view
	s += m.list.View()

	// Detail panel for selected item
	if selected := m.list.SelectedItem(); selected != nil {
		if item, ok := selected.(ResultItem); ok {
			s += "\n" + item.DetailView()
		}
	}

	return s
}

func (Model) renderShortHelp() string {
	return HelpStyle.Render("↑/↓ navigate • f filter • ? help • q quit")
}
