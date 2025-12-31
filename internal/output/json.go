package output

import (
	"encoding/json"

	"github.com/leonardomso/gone/internal/checker"
)

// JSONFormatter formats reports as JSON.
type JSONFormatter struct{}

// jsonOutput is the JSON structure for output.
type jsonOutput struct {
	GeneratedAt string        `json:"generated_at"`
	TotalFiles  int           `json:"total_files"`
	TotalLinks  int           `json:"total_links"`
	UniqueURLs  int           `json:"unique_urls"`
	Summary     jsonSummary   `json:"summary"`
	Results     []jsonResult  `json:"results"`
	Ignored     []jsonIgnored `json:"ignored,omitempty"`
}

type jsonSummary struct {
	Alive      int `json:"alive"`
	Redirects  int `json:"redirects"`
	Blocked    int `json:"blocked"`
	Dead       int `json:"dead"`
	Errors     int `json:"errors"`
	Duplicates int `json:"duplicates"`
	Ignored    int `json:"ignored,omitempty"`
}

type jsonResult struct {
	URL           string         `json:"url"`
	FilePath      string         `json:"file_path"`
	Line          int            `json:"line,omitempty"`
	Text          string         `json:"text,omitempty"`
	StatusCode    int            `json:"status_code"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
	RedirectChain []jsonRedirect `json:"redirect_chain,omitempty"`
	FinalURL      string         `json:"final_url,omitempty"`
	FinalStatus   int            `json:"final_status,omitempty"`
	DuplicateOf   string         `json:"duplicate_of,omitempty"`
}

type jsonRedirect struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

type jsonIgnored struct {
	URL    string `json:"url"`
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Reason string `json:"reason"`
	Rule   string `json:"rule"`
}

// Format implements Formatter.
func (*JSONFormatter) Format(report *Report) ([]byte, error) {
	output := jsonOutput{
		GeneratedAt: report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalFiles:  len(report.Files),
		TotalLinks:  report.TotalLinks,
		UniqueURLs:  report.UniqueURLs,
		Summary: jsonSummary{
			Alive:      report.Summary.Alive,
			Redirects:  report.Summary.Redirects,
			Blocked:    report.Summary.Blocked,
			Dead:       report.Summary.Dead,
			Errors:     report.Summary.Errors,
			Duplicates: report.Summary.Duplicates,
			Ignored:    len(report.Ignored),
		},
		Results: make([]jsonResult, 0, len(report.Results)),
	}

	for _, r := range report.Results {
		jr := jsonResult{
			URL:        r.Link.URL,
			FilePath:   r.Link.FilePath,
			Line:       r.Link.Line,
			Text:       r.Link.Text,
			StatusCode: r.StatusCode,
			Status:     r.Status.String(),
			Error:      r.Error,
		}

		// Add redirect chain if present
		if len(r.RedirectChain) > 0 {
			jr.RedirectChain = make([]jsonRedirect, len(r.RedirectChain))
			for i, red := range r.RedirectChain {
				jr.RedirectChain[i] = jsonRedirect{
					URL:        red.URL,
					StatusCode: red.StatusCode,
				}
			}
			jr.FinalURL = r.FinalURL
			jr.FinalStatus = r.FinalStatus
		}

		// Add duplicate reference if present
		if r.DuplicateOf != nil {
			jr.DuplicateOf = r.DuplicateOf.Link.URL
		}

		output.Results = append(output.Results, jr)
	}

	// Add ignored URLs if present
	for _, ig := range report.Ignored {
		output.Ignored = append(output.Ignored, jsonIgnored(ig))
	}

	return json.MarshalIndent(output, "", "  ")
}

// filterResults returns results based on status.
func filterByStatus(results []checker.Result, statuses ...checker.LinkStatus) []checker.Result {
	statusSet := map[checker.LinkStatus]bool{}
	for _, s := range statuses {
		statusSet[s] = true
	}

	var filtered []checker.Result
	for _, r := range results {
		if statusSet[r.Status] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
