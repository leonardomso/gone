package output

import (
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats reports as YAML.
type YAMLFormatter struct{}

// yamlOutput is the YAML structure for output.
type yamlOutput struct {
	GeneratedAt string        `yaml:"generated_at"`
	Results     []yamlResult  `yaml:"results"`
	Ignored     []yamlIgnored `yaml:"ignored,omitempty"`
	Summary     yamlSummary   `yaml:"summary"`
	TotalFiles  int           `yaml:"total_files"`
	TotalLinks  int           `yaml:"total_links"`
	UniqueURLs  int           `yaml:"unique_urls"`
}

type yamlSummary struct {
	Alive      int `yaml:"alive"`
	Redirects  int `yaml:"redirects"`
	Blocked    int `yaml:"blocked"`
	Dead       int `yaml:"dead"`
	Errors     int `yaml:"errors"`
	Duplicates int `yaml:"duplicates"`
	Ignored    int `yaml:"ignored,omitempty"`
}

type yamlResult struct {
	URL           string         `yaml:"url"`
	FilePath      string         `yaml:"file_path"`
	Text          string         `yaml:"text,omitempty"`
	Status        string         `yaml:"status"`
	Error         string         `yaml:"error,omitempty"`
	FinalURL      string         `yaml:"final_url,omitempty"`
	DuplicateOf   string         `yaml:"duplicate_of,omitempty"`
	RedirectChain []yamlRedirect `yaml:"redirect_chain,omitempty"`
	Line          int            `yaml:"line,omitempty"`
	StatusCode    int            `yaml:"status_code"`
	FinalStatus   int            `yaml:"final_status,omitempty"`
}

type yamlRedirect struct {
	URL        string `yaml:"url"`
	StatusCode int    `yaml:"status_code"`
}

type yamlIgnored struct {
	URL    string `yaml:"url"`
	File   string `yaml:"file"`
	Reason string `yaml:"reason"`
	Rule   string `yaml:"rule"`
	Line   int    `yaml:"line,omitempty"`
}

// Format implements Formatter.
func (*YAMLFormatter) Format(report *Report) ([]byte, error) {
	output := yamlOutput{
		GeneratedAt: report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalFiles:  len(report.Files),
		TotalLinks:  report.TotalLinks,
		UniqueURLs:  report.UniqueURLs,
		Summary: yamlSummary{
			Alive:      report.Summary.Alive,
			Redirects:  report.Summary.Redirects,
			Blocked:    report.Summary.Blocked,
			Dead:       report.Summary.Dead,
			Errors:     report.Summary.Errors,
			Duplicates: report.Summary.Duplicates,
			Ignored:    len(report.Ignored),
		},
		Results: make([]yamlResult, 0, len(report.Results)),
	}

	for _, r := range report.Results {
		yr := yamlResult{
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
			yr.RedirectChain = make([]yamlRedirect, len(r.RedirectChain))
			for i, red := range r.RedirectChain {
				yr.RedirectChain[i] = yamlRedirect{
					URL:        red.URL,
					StatusCode: red.StatusCode,
				}
			}
			yr.FinalURL = r.FinalURL
			yr.FinalStatus = r.FinalStatus
		}

		// Add duplicate reference if present
		if r.DuplicateOf != nil {
			yr.DuplicateOf = r.DuplicateOf.Link.URL
		}

		output.Results = append(output.Results, yr)
	}

	// Add ignored URLs if present
	for _, ig := range report.Ignored {
		output.Ignored = append(output.Ignored, yamlIgnored(ig))
	}

	return yaml.Marshal(output)
}
