package checker

// Link represents a URL to be checked.
// This is decoupled from parser.Link to keep the checker package independent.
type Link struct {
	URL      string // The URL to check
	FilePath string // Source file where the link was found
	Line     int    // Line number in the source file (0 if unknown)
}

// Result represents the outcome of checking a single link.
type Result struct {
	Link       Link   // The original link that was checked
	StatusCode int    // HTTP status code (0 if request failed)
	IsAlive    bool   // true if the link is considered alive
	Error      string // Error message if the request failed
}

// FilterDead returns only the results where the link is not alive.
func FilterDead(results []Result) []Result {
	var dead []Result
	for _, r := range results {
		if !r.IsAlive {
			dead = append(dead, r)
		}
	}
	return dead
}

// FilterAlive returns only the results where the link is alive.
func FilterAlive(results []Result) []Result {
	var alive []Result
	for _, r := range results {
		if r.IsAlive {
			alive = append(alive, r)
		}
	}
	return alive
}

// Summary provides statistics about check results.
type Summary struct {
	Total  int // Total links checked
	Alive  int // Links that are alive
	Dead   int // Links that are dead
	Errors int // Links that failed with errors
}

// Summarize creates a summary from a slice of results.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch {
		case r.IsAlive:
			s.Alive++
		case r.Error != "":
			s.Errors++
			s.Dead++ // Errors count as dead
		default:
			s.Dead++
		}
	}
	return s
}
