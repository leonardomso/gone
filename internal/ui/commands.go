package ui

import (
	"context"

	"gone/internal/checker"
	"gone/internal/parser"
	"gone/internal/scanner"

	tea "github.com/charmbracelet/bubbletea"
)

// ScanFilesCmdWithPath returns a command that scans for markdown files in the given path.
func ScanFilesCmdWithPath(path string) tea.Cmd {
	return func() tea.Msg {
		files, err := scanner.FindMarkdownFiles(path)
		return FilesFoundMsg{Files: files, Err: err}
	}
}

// ExtractLinksCmd extracts links from the given files.
func ExtractLinksCmd(files []string) tea.Cmd {
	return func() tea.Msg {
		parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
		if err != nil {
			return LinksExtractedMsg{Err: err}
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
		return LinksExtractedMsg{Links: links}
	}
}

// CheckerState holds the state needed for checking links.
// This allows the commands to be stateless functions.
type CheckerState struct {
	ResultsChan <-chan checker.Result
	CancelFunc  context.CancelFunc
}

// StartCheckingCmd initializes the checker and returns the first result.
func StartCheckingCmd(links []checker.Link, state *CheckerState) tea.Cmd {
	return func() tea.Msg {
		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		state.CancelFunc = cancel

		// Create checker with default options
		opts := checker.DefaultOptions()
		c := checker.New(opts)

		// Start checking and store the channel
		state.ResultsChan = c.Check(ctx, links)

		// Get the first result
		result, ok := <-state.ResultsChan
		if !ok {
			return AllChecksCompleteMsg{}
		}
		return LinkCheckedMsg{Result: result}
	}
}

// WaitForNextResultCmd waits for the next result from the channel.
func WaitForNextResultCmd(state *CheckerState) tea.Cmd {
	return func() tea.Msg {
		if state.ResultsChan == nil {
			return AllChecksCompleteMsg{}
		}

		result, ok := <-state.ResultsChan
		if !ok {
			return AllChecksCompleteMsg{}
		}
		return LinkCheckedMsg{Result: result}
	}
}
