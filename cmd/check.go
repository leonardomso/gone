package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gone/internal/checker"
	"gone/internal/parser"
	"gone/internal/scanner"

	"github.com/spf13/cobra"
)

// jsonOutput represents the JSON structure for output.
type jsonOutput struct {
	TotalFiles int          `json:"total_files"`
	TotalLinks int          `json:"total_links"`
	DeadLinks  int          `json:"dead_links"`
	Results    []jsonResult `json:"results"`
}

type jsonResult struct {
	URL        string `json:"url"`
	FilePath   string `json:"file_path"`
	StatusCode int    `json:"status_code"`
	IsAlive    bool   `json:"is_alive"`
	Error      string `json:"error,omitempty"`
}

// Flag variables.
var (
	format      string
	concurrency int
	timeout     int
	retries     int
)

// checkCmd represents the check command.
var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Scan markdown files for dead links",
	Long: `Scan a directory for markdown files and check all HTTP/HTTPS links.

If no path is provided, scans the current directory.
Returns a list of all dead links found (non-200 status codes).

Examples:
  gone check                         # Scan current directory
  gone check ./docs                  # Scan specific directory  
  gone check --format=json           # Output as JSON
  gone check --concurrency=20        # Use 20 concurrent workers
  gone check --timeout=30            # 30 second timeout per request
  gone check --retries=3             # Retry failed requests 3 times`,
	Args: cobra.MaximumNArgs(1),
	Run:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Output format
	checkCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")

	// Performance options
	checkCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers")
	checkCmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "Timeout per request in seconds")
	checkCmd.Flags().IntVarP(&retries, "retries", "r", 2, "Number of retries for failed requests")
}

func runCheck(_ *cobra.Command, args []string) {
	// Determine the path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	isJSON := format == "json"

	// Find all markdown files
	files, err := scanner.FindMarkdownFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if !isJSON {
		fmt.Printf("Found %d markdown file(s)\n", len(files))
	}

	// Extract all URLs from the files
	parserLinks, err := parser.ExtractLinksFromMultipleFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
		os.Exit(1)
	}

	if len(parserLinks) == 0 {
		if isJSON {
			outputJSON(files, nil, nil)
		} else {
			fmt.Println("No links found.")
		}
		return
	}

	if !isJSON {
		fmt.Printf("Found %d link(s), checking...\n", len(parserLinks))
	}

	// Convert parser.Link to checker.Link
	links := make([]checker.Link, len(parserLinks))
	for i, pl := range parserLinks {
		links[i] = checker.Link{
			URL:      pl.URL,
			FilePath: pl.FilePath,
			Line:     pl.Line,
		}
	}

	// Create checker with configured options
	opts := checker.DefaultOptions().
		WithConcurrency(concurrency).
		WithTimeout(time.Duration(timeout) * time.Second).
		WithMaxRetries(retries)

	c := checker.New(opts)

	// Check all links
	results := c.CheckAll(links)
	deadLinks := checker.FilterDead(results)

	// Output based on format flag
	if isJSON {
		outputJSON(files, links, deadLinks)
	} else {
		outputText(deadLinks)
	}

	// Exit with code 1 if dead links found (useful for CI)
	if len(deadLinks) > 0 {
		os.Exit(1)
	}
}

// outputJSON prints results as JSON.
func outputJSON(files []string, links []checker.Link, deadLinks []checker.Result) {
	output := jsonOutput{
		TotalFiles: len(files),
		TotalLinks: len(links),
		DeadLinks:  len(deadLinks),
		Results:    make([]jsonResult, 0, len(deadLinks)),
	}

	for _, r := range deadLinks {
		output.Results = append(output.Results, jsonResult{
			URL:        r.Link.URL,
			FilePath:   r.Link.FilePath,
			StatusCode: r.StatusCode,
			IsAlive:    r.IsAlive,
			Error:      r.Error,
		})
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}

// outputText prints results as human-readable text.
func outputText(deadLinks []checker.Result) {
	if len(deadLinks) == 0 {
		fmt.Println("\nAll links are alive!")
	} else {
		fmt.Printf("\nFound %d dead link(s):\n\n", len(deadLinks))
		for _, r := range deadLinks {
			if r.Error != "" {
				fmt.Printf("  [ERROR] %s\n", r.Link.URL)
				fmt.Printf("          File: %s\n", r.Link.FilePath)
				fmt.Printf("          Error: %s\n\n", r.Error)
			} else {
				fmt.Printf("  [%d] %s\n       File: %s\n\n", r.StatusCode, r.Link.URL, r.Link.FilePath)
			}
		}
	}
}
