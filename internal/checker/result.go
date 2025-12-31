package checker

import "fmt"

// LinkStatus represents the category of a checked link.
type LinkStatus int

const (
	// StatusAlive indicates the link returned a 2xx response.
	StatusAlive LinkStatus = iota
	// StatusRedirect indicates the link redirected but the final destination is alive.
	StatusRedirect
	// StatusBlocked indicates the server returned 403 (possible bot detection).
	StatusBlocked
	// StatusDead indicates the link is broken (4xx except 403, 5xx, or redirect to dead).
	StatusDead
	// StatusError indicates a network error (timeout, DNS failure, connection refused).
	StatusError
	// StatusDuplicate indicates this link was already checked (references primary result).
	StatusDuplicate
)

// String returns the string representation of the status.
func (s LinkStatus) String() string {
	switch s {
	case StatusAlive:
		return "alive"
	case StatusRedirect:
		return "redirect"
	case StatusBlocked:
		return "blocked"
	case StatusDead:
		return "dead"
	case StatusError:
		return "error"
	case StatusDuplicate:
		return "duplicate"
	default:
		return "unknown"
	}
}

// Label returns a short label for display (e.g., in badges).
func (s LinkStatus) Label() string {
	switch s {
	case StatusAlive:
		return "OK"
	case StatusRedirect:
		return "REDIRECT"
	case StatusBlocked:
		return "BLOCKED"
	case StatusDead:
		return "DEAD"
	case StatusError:
		return "ERROR"
	case StatusDuplicate:
		return "DUPLICATE"
	default:
		return "???"
	}
}

// Description returns a human-readable explanation of the status.
func (s LinkStatus) Description() string {
	switch s {
	case StatusAlive:
		return "Link is working (2xx response)"
	case StatusRedirect:
		return "URL redirected but final destination works. Consider updating the URL."
	case StatusBlocked:
		return "Server returned 403 Forbidden. May be blocking automated requests."
	case StatusDead:
		return "Link is broken (4xx/5xx response or redirect leads to dead page)"
	case StatusError:
		return "Network error (DNS failure, timeout, connection refused)"
	case StatusDuplicate:
		return "This URL appears multiple times. See original occurrence for status."
	default:
		return "Unknown status"
	}
}

// Redirect represents a single hop in a redirect chain.
type Redirect struct {
	URL        string // The URL that redirected
	StatusCode int    // The redirect status code (301, 302, 307, 308)
}

// Link represents a URL to be checked.
// This is decoupled from parser.Link to keep the checker package independent.
type Link struct {
	URL      string // The URL to check
	FilePath string // Source file where the link was found
	Line     int    // Line number in the source file (0 if unknown)
}

// Result represents the outcome of checking a single link.
type Result struct {
	Link       Link       // The original link that was checked
	StatusCode int        // HTTP status code (0 if request failed)
	Status     LinkStatus // Computed status category
	Error      string     // Error message if applicable

	// Redirect info (populated when redirects occurred)
	RedirectChain []Redirect // Full chain of redirects
	FinalURL      string     // Final destination URL after following redirects
	FinalStatus   int        // Status code of final destination

	// Duplicate info (populated when Status == StatusDuplicate)
	DuplicateOf *Result // Points to primary result if this is a duplicate
}

// IsAlive returns true if the link is considered alive (2xx response).
// Kept for backward compatibility.
func (r Result) IsAlive() bool {
	return r.Status == StatusAlive
}

// IsWarning returns true if the link has a warning status (redirect or blocked).
func (r Result) IsWarning() bool {
	return r.Status == StatusRedirect || r.Status == StatusBlocked
}

// IsDead returns true if the link is dead or errored.
func (r Result) IsDead() bool {
	return r.Status == StatusDead || r.Status == StatusError
}

// IsDuplicate returns true if this is a duplicate of another checked link.
func (r Result) IsDuplicate() bool {
	return r.Status == StatusDuplicate
}

// StatusDisplay returns a formatted string for CLI display.
func (r Result) StatusDisplay() string {
	switch r.Status {
	case StatusAlive:
		return fmt.Sprintf("[%d]", r.StatusCode)
	case StatusRedirect:
		return "[REDIRECT]"
	case StatusBlocked:
		return "[BLOCKED]"
	case StatusDead:
		if r.StatusCode > 0 {
			return fmt.Sprintf("[%d]", r.StatusCode)
		}
		return "[DEAD]"
	case StatusError:
		return "[ERROR]"
	case StatusDuplicate:
		return "[DUPLICATE]"
	default:
		return "[???]"
	}
}

// FilterByStatus returns results matching the given status.
func FilterByStatus(results []Result, status LinkStatus) []Result {
	var filtered []Result
	for _, r := range results {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterWarnings returns results with warning status (redirect or blocked).
func FilterWarnings(results []Result) []Result {
	var warnings []Result
	for _, r := range results {
		if r.IsWarning() {
			warnings = append(warnings, r)
		}
	}
	return warnings
}

// FilterDead returns results that are dead or errored.
func FilterDead(results []Result) []Result {
	var dead []Result
	for _, r := range results {
		if r.IsDead() {
			dead = append(dead, r)
		}
	}
	return dead
}

// FilterAlive returns only the results where the link is alive.
func FilterAlive(results []Result) []Result {
	var alive []Result
	for _, r := range results {
		if r.IsAlive() {
			alive = append(alive, r)
		}
	}
	return alive
}

// FilterDuplicates returns only duplicate results.
func FilterDuplicates(results []Result) []Result {
	var duplicates []Result
	for _, r := range results {
		if r.IsDuplicate() {
			duplicates = append(duplicates, r)
		}
	}
	return duplicates
}

// Summary provides statistics about check results.
type Summary struct {
	Total      int // Total links checked (including duplicates)
	UniqueURLs int // Number of unique URLs actually checked
	Alive      int // Links that are alive (2xx)
	Redirects  int // Links that redirect to working pages
	Blocked    int // Links blocked by 403
	Dead       int // Links that are dead (4xx/5xx)
	Errors     int // Links that failed with network errors
	Duplicates int // Duplicate occurrences
}

// Summarize creates a summary from a slice of results.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}

	// Count unique URLs
	seen := map[string]bool{}
	for _, r := range results {
		if !seen[r.Link.URL] {
			seen[r.Link.URL] = true
			s.UniqueURLs++
		}
	}

	for _, r := range results {
		switch r.Status {
		case StatusAlive:
			s.Alive++
		case StatusRedirect:
			s.Redirects++
		case StatusBlocked:
			s.Blocked++
		case StatusDead:
			s.Dead++
		case StatusError:
			s.Errors++
		case StatusDuplicate:
			s.Duplicates++
		}
	}
	return s
}

// HasIssues returns true if there are any warnings or dead links.
func (s Summary) HasIssues() bool {
	return s.Redirects > 0 || s.Blocked > 0 || s.Dead > 0 || s.Errors > 0
}

// HasDeadLinks returns true if there are dead links or errors (exit code 1 condition).
func (s Summary) HasDeadLinks() bool {
	return s.Dead > 0 || s.Errors > 0
}

// WarningsCount returns total warnings (redirects + blocked).
func (s Summary) WarningsCount() int {
	return s.Redirects + s.Blocked
}
