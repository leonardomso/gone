package ui

import (
	"context"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"

	tea "github.com/charmbracelet/bubbletea"
)

// ScanFilesCmdWithPath returns a command that scans for markdown files in the given path.
//
// Deprecated: Use ScanFilesCmdWithTypes instead.
func ScanFilesCmdWithPath(path string) tea.Cmd {
	return ScanFilesCmdWithTypes(path, []string{"md"})
}

// ScanFilesCmdWithTypes returns a command that scans for files of the given types in the given path.
func ScanFilesCmdWithTypes(path string, fileTypes []string) tea.Cmd {
	return func() tea.Msg {
		files, err := scanner.FindFilesByTypes(path, fileTypes)
		return FilesFoundMsg{Files: files, Err: err}
	}
}

// ScanFilesCmdWithOptions returns a command that scans for files using ScanOptions.
// This supports include/exclude glob patterns from the config file.
func ScanFilesCmdWithOptions(path string, fileTypes, include, exclude []string) tea.Cmd {
	return func() tea.Msg {
		opts := scanner.ScanOptions{
			Root:    path,
			Types:   fileTypes,
			Include: include,
			Exclude: exclude,
		}
		files, err := scanner.FindFilesWithOptions(opts)
		return FilesFoundMsg{Files: files, Err: err}
	}
}

// ExtractLinksCmd extracts links from the given files.
//
// Deprecated: Use ExtractLinksCmdWithRegistry instead.
func ExtractLinksCmd(files []string) tea.Cmd {
	return ExtractLinksCmdWithRegistry(files, false)
}

// ExtractLinksCmdWithRegistry extracts links from the given files using the parser registry.
func ExtractLinksCmdWithRegistry(files []string, strictMode bool) tea.Cmd {
	return func() tea.Msg {
		parserLinks, err := parser.ExtractLinksFromMultipleFilesWithRegistry(files, strictMode)
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
				Text:     pl.Text,
			}
		}

		// Count unique URLs
		uniqueURLs := countUniqueURLs(links)
		duplicates := len(links) - uniqueURLs

		return LinksExtractedMsg{
			Links:      links,
			UniqueURLs: uniqueURLs,
			Duplicates: duplicates,
		}
	}
}

// countUniqueURLs returns the number of unique URLs in the slice.
func countUniqueURLs(links []checker.Link) int {
	seen := map[string]bool{}
	for _, l := range links {
		seen[l.URL] = true
	}
	return len(seen)
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
