package output

import (
	"encoding/xml"
	"fmt"

	"github.com/leonardomso/gone/internal/checker"
)

// JUnitFormatter formats reports as JUnit XML for CI/CD integration.
// Only failed/error links are included as test cases.
type JUnitFormatter struct{}

// junitTestSuites is the root element for JUnit XML.
type junitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	Name      string           `xml:"name,attr"`
	Tests     int              `xml:"tests,attr"`
	Failures  int              `xml:"failures,attr"`
	Errors    int              `xml:"errors,attr"`
	TestSuite []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Format implements Formatter.
func (*JUnitFormatter) Format(report *Report) ([]byte, error) {
	// Group results by file
	fileResults := map[string][]checker.Result{}
	for _, r := range report.Results {
		// Only include dead and error results
		if r.Status == checker.StatusDead || r.Status == checker.StatusError {
			fileResults[r.Link.FilePath] = append(fileResults[r.Link.FilePath], r)
		}
	}

	// Count totals
	totalTests := 0
	totalFailures := 0
	totalErrors := 0

	for _, results := range fileResults {
		for _, r := range results {
			totalTests++
			if r.Status == checker.StatusError {
				totalErrors++
			} else {
				totalFailures++
			}
		}
	}

	// Build test suites
	suites := junitTestSuites{
		Name:     "gone-link-check",
		Tests:    totalTests,
		Failures: totalFailures,
		Errors:   totalErrors,
	}

	for file, results := range fileResults {
		suite := junitTestSuite{
			Name: file,
		}

		for _, r := range results {
			suite.Tests++

			tc := junitTestCase{
				Name:      r.Link.URL,
				ClassName: fmt.Sprintf("%s:%d", r.Link.FilePath, r.Link.Line),
			}

			if r.Status == checker.StatusError {
				suite.Errors++
				tc.Error = &junitError{
					Message: truncateForXML(r.Error, 200),
					Type:    "error",
					Content: buildErrorContent(r),
				}
			} else {
				suite.Failures++
				tc.Failure = &junitFailure{
					Message: buildFailureMessage(r),
					Type:    "dead",
					Content: buildFailureContent(r),
				}
			}

			suite.TestCases = append(suite.TestCases, tc)
		}

		suites.TestSuite = append(suites.TestSuite, suite)
	}

	// If no failures/errors, create an empty test suite to indicate success
	if len(suites.TestSuite) == 0 {
		suites.TestSuite = append(suites.TestSuite, junitTestSuite{
			Name:  "all-links",
			Tests: 0,
		})
	}

	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}

// buildFailureMessage creates a short failure message.
func buildFailureMessage(r checker.Result) string {
	if r.StatusCode > 0 {
		return fmt.Sprintf("HTTP %d", r.StatusCode)
	}
	return "Link is dead"
}

// buildFailureContent creates detailed failure content.
func buildFailureContent(r checker.Result) string {
	content := ""
	if r.Link.Text != "" {
		content += fmt.Sprintf("Link text: %q\n", truncateForXML(r.Link.Text, 100))
	}
	if r.StatusCode > 0 {
		content += fmt.Sprintf("Status: %d\n", r.StatusCode)
	}
	if len(r.RedirectChain) > 0 {
		content += fmt.Sprintf("Final URL: %s\n", r.FinalURL)
		content += fmt.Sprintf("Final Status: %d\n", r.FinalStatus)
	}
	return content
}

// buildErrorContent creates detailed error content.
func buildErrorContent(r checker.Result) string {
	content := ""
	if r.Link.Text != "" {
		content += fmt.Sprintf("Link text: %q\n", truncateForXML(r.Link.Text, 100))
	}
	content += fmt.Sprintf("Error: %s\n", r.Error)
	return content
}

// truncateForXML truncates a string and ensures it's safe for XML.
func truncateForXML(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
