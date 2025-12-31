package ui

import (
	"fmt"

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
	if i.Result.Error != "" {
		return fmt.Sprintf("%s | Error: %s", i.Result.Link.FilePath, i.Result.Error)
	}
	return fmt.Sprintf("%s | Status: %d", i.Result.Link.FilePath, i.Result.StatusCode)
}

// StatusCode returns the HTTP status code for display.
func (i ResultItem) StatusCode() int {
	return i.Result.StatusCode
}

// HasError returns true if the result has an error.
func (i ResultItem) HasError() bool {
	return i.Result.Error != ""
}

// Is4xx returns true if the status code is a 4xx error.
func (i ResultItem) Is4xx() bool {
	return i.Result.StatusCode >= 400 && i.Result.StatusCode < 500
}

// Is5xx returns true if the status code is a 5xx error.
func (i ResultItem) Is5xx() bool {
	return i.Result.StatusCode >= 500
}

// Is3xx returns true if the status code is a 3xx redirect.
func (i ResultItem) Is3xx() bool {
	return i.Result.StatusCode >= 300 && i.Result.StatusCode < 400
}

// ResultsToItems converts a slice of checker.Result to ResultItems.
func ResultsToItems(results []checker.Result) []ResultItem {
	items := make([]ResultItem, len(results))
	for i, r := range results {
		items[i] = ResultItem{Result: r}
	}
	return items
}
