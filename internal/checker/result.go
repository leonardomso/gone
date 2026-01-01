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

// Pre-defined strings to avoid allocations in String() methods.
var (
	statusStrings = [...]string{
		StatusAlive:     "alive",
		StatusRedirect:  "redirect",
		StatusBlocked:   "blocked",
		StatusDead:      "dead",
		StatusError:     "error",
		StatusDuplicate: "duplicate",
	}

	statusLabels = [...]string{
		StatusAlive:     "OK",
		StatusRedirect:  "REDIRECT",
		StatusBlocked:   "BLOCKED",
		StatusDead:      "DEAD",
		StatusError:     "ERROR",
		StatusDuplicate: "DUPLICATE",
	}

	statusDescriptions = [...]string{
		StatusAlive:     "Link is working (2xx response)",
		StatusRedirect:  "URL redirected but final destination works. Consider updating the URL.",
		StatusBlocked:   "Server returned 403 Forbidden. May be blocking automated requests.",
		StatusDead:      "Link is broken (4xx/5xx response or redirect leads to dead page)",
		StatusError:     "Network error (DNS failure, timeout, connection refused)",
		StatusDuplicate: "This URL appears multiple times. See original occurrence for status.",
	}
)

// String returns the string representation of the status.
// Uses pre-defined strings to avoid allocations.
func (s LinkStatus) String() string {
	if s >= 0 && int(s) < len(statusStrings) {
		return statusStrings[s]
	}
	return "unknown"
}

// Label returns a short label for display (e.g., in badges).
// Uses pre-defined strings to avoid allocations.
func (s LinkStatus) Label() string {
	if s >= 0 && int(s) < len(statusLabels) {
		return statusLabels[s]
	}
	return "???"
}

// Description returns a human-readable explanation of the status.
// Uses pre-defined strings to avoid allocations.
func (s LinkStatus) Description() string {
	if s >= 0 && int(s) < len(statusDescriptions) {
		return statusDescriptions[s]
	}
	return "Unknown status"
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
	Text     string // Link text (e.g., "Click here") for display purposes
	Line     int    // Line number in the source file (0 if unknown)
}

// Result represents the outcome of checking a single link.
type Result struct {

	// Duplicate info (populated when Status == StatusDuplicate)
	DuplicateOf *Result // Points to primary result if this is a duplicate
	Link        Link    // The original link that was checked
	Error       string  // Error message if applicable

	FinalURL string // Final destination URL after following redirects

	// Redirect info (populated when redirects occurred)
	RedirectChain []Redirect // Full chain of redirects
	StatusCode    int        // HTTP status code (0 if request failed)
	Status        LinkStatus // Computed status category
	FinalStatus   int        // Status code of final destination

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
// Pre-allocates slice capacity based on expected ratio.
func FilterByStatus(results []Result, status LinkStatus) []Result {
	// Estimate capacity - most filters return ~10-30% of results
	filtered := make([]Result, 0, len(results)/4)
	for _, r := range results {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterWarnings returns results with warning status (redirect or blocked).
// Pre-allocates slice capacity based on expected ratio.
func FilterWarnings(results []Result) []Result {
	warnings := make([]Result, 0, len(results)/4)
	for _, r := range results {
		if r.IsWarning() {
			warnings = append(warnings, r)
		}
	}
	return warnings
}

// FilterDead returns results that are dead or errored.
// Pre-allocates slice capacity based on expected ratio.
func FilterDead(results []Result) []Result {
	dead := make([]Result, 0, len(results)/8)
	for _, r := range results {
		if r.IsDead() {
			dead = append(dead, r)
		}
	}
	return dead
}

// FilterAlive returns only the results where the link is alive.
// Pre-allocates slice capacity - alive is typically the majority.
func FilterAlive(results []Result) []Result {
	alive := make([]Result, 0, len(results)*3/4)
	for _, r := range results {
		if r.IsAlive() {
			alive = append(alive, r)
		}
	}
	return alive
}

// FilterDuplicates returns only duplicate results.
// Pre-allocates slice capacity based on expected ratio.
func FilterDuplicates(results []Result) []Result {
	duplicates := make([]Result, 0, len(results)/10)
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
