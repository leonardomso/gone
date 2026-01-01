package output

import (
	"encoding/xml"
)

// XMLFormatter formats reports as generic XML.
type XMLFormatter struct{}

// xmlOutput is the XML structure for output.
type xmlOutput struct {
	Ignored     *xmlIgnored `xml:"ignored,omitempty"`
	XMLName     xml.Name    `xml:"report"`
	GeneratedAt string      `xml:"generated_at,attr"`
	Results     xmlResults  `xml:"results"`
	Summary     xmlSummary  `xml:"summary"`
	TotalFiles  int         `xml:"total_files,attr"`
	TotalLinks  int         `xml:"total_links,attr"`
	UniqueURLs  int         `xml:"unique_urls,attr"`
}

type xmlSummary struct {
	Alive      int `xml:"alive"`
	Redirects  int `xml:"redirects"`
	Blocked    int `xml:"blocked"`
	Dead       int `xml:"dead"`
	Errors     int `xml:"errors"`
	Duplicates int `xml:"duplicates"`
	Ignored    int `xml:"ignored,omitempty"`
}

type xmlResults struct {
	Results []xmlResult `xml:"result"`
}

type xmlResult struct {
	RedirectChain *xmlRedirectChain `xml:"redirect_chain,omitempty"`
	Status        string            `xml:"status,attr"`
	URL           string            `xml:"url"`
	FilePath      string            `xml:"file"`
	Text          string            `xml:"text,omitempty"`
	Error         string            `xml:"error,omitempty"`
	FinalURL      string            `xml:"final_url,omitempty"`
	DuplicateOf   string            `xml:"duplicate_of,omitempty"`
	StatusCode    int               `xml:"status_code,attr"`
	Line          int               `xml:"line,omitempty"`
	FinalStatus   int               `xml:"final_status,omitempty"`
}

type xmlRedirectChain struct {
	Redirects []xmlRedirect `xml:"redirect"`
}

type xmlRedirect struct {
	URL        string `xml:"url,attr"`
	StatusCode int    `xml:"status_code,attr"`
}

type xmlIgnored struct {
	Items []xmlIgnoredItem `xml:"item"`
}

type xmlIgnoredItem struct {
	URL    string `xml:"url"`
	File   string `xml:"file"`
	Reason string `xml:"reason"`
	Rule   string `xml:"rule"`
	Line   int    `xml:"line,omitempty"`
}

// Format implements Formatter.
func (*XMLFormatter) Format(report *Report) ([]byte, error) {
	output := xmlOutput{
		GeneratedAt: report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalFiles:  len(report.Files),
		TotalLinks:  report.TotalLinks,
		UniqueURLs:  report.UniqueURLs,
		Summary: xmlSummary{
			Alive:      report.Summary.Alive,
			Redirects:  report.Summary.Redirects,
			Blocked:    report.Summary.Blocked,
			Dead:       report.Summary.Dead,
			Errors:     report.Summary.Errors,
			Duplicates: report.Summary.Duplicates,
			Ignored:    len(report.Ignored),
		},
		Results: xmlResults{
			Results: make([]xmlResult, 0, len(report.Results)),
		},
	}

	for _, r := range report.Results {
		xr := xmlResult{
			Status:     r.Status.String(),
			StatusCode: r.StatusCode,
			URL:        r.Link.URL,
			FilePath:   r.Link.FilePath,
			Line:       r.Link.Line,
			Text:       r.Link.Text,
			Error:      r.Error,
		}

		// Add redirect chain if present
		if len(r.RedirectChain) > 0 {
			xr.RedirectChain = &xmlRedirectChain{
				Redirects: make([]xmlRedirect, len(r.RedirectChain)),
			}
			for i, red := range r.RedirectChain {
				xr.RedirectChain.Redirects[i] = xmlRedirect{
					URL:        red.URL,
					StatusCode: red.StatusCode,
				}
			}
			xr.FinalURL = r.FinalURL
			xr.FinalStatus = r.FinalStatus
		}

		// Add duplicate reference if present
		if r.DuplicateOf != nil {
			xr.DuplicateOf = r.DuplicateOf.Link.URL
		}

		output.Results.Results = append(output.Results.Results, xr)
	}

	// Add ignored URLs if present
	if len(report.Ignored) > 0 {
		output.Ignored = &xmlIgnored{
			Items: make([]xmlIgnoredItem, len(report.Ignored)),
		}
		for i, ig := range report.Ignored {
			output.Ignored.Items[i] = xmlIgnoredItem(ig)
		}
	}

	// Add XML header and marshal with indentation
	data, err := xml.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}
