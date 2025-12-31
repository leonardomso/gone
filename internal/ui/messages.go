package ui

import "gone/internal/checker"

// FilesFoundMsg is sent when markdown files have been discovered.
type FilesFoundMsg struct {
	Files []string
	Err   error
}

// LinksExtractedMsg is sent when links have been extracted from files.
type LinksExtractedMsg struct {
	Links []checker.Link
	Err   error
}

// LinkCheckedMsg is sent when a single link has been checked.
type LinkCheckedMsg struct {
	Result checker.Result
}

// AllChecksCompleteMsg is sent when all link checks have finished.
type AllChecksCompleteMsg struct{}

// WindowSizeMsg is sent when the terminal window size changes.
type WindowSizeMsg struct {
	Width  int
	Height int
}
