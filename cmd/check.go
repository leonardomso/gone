package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gone/internal/checker"
	"gone/internal/parser"
	"gone/internal/scanner"
)

// jsonOutput represents the JSON structure for output
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

// Flag variables - these store the values passed via command line
var (
	format string // --format flag (text or json)
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Scan markdown files for dead links",
	Long: `Scan a directory for markdown files and check all HTTP/HTTPS links.

If no path is provided, scans the current directory.
Returns a list of all dead links found (non-200 status codes).

Examples:
  gone check                    # Scan current directory
  gone check ./docs             # Scan specific directory  
  gone check --format=json      # Output as JSON
  gone check ./docs -f json     # Both options combined`,
	Args: cobra.MaximumNArgs(1), // Allow 0 or 1 argument (the path)
	Run: func(cmd *cobra.Command, args []string) {
		// Determine the path to scan
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		// Find all markdown files
		files, err := scanner.FindMarkdownFiles(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
			os.Exit(1)
		}

		// Only show progress for text output
		isJSON := format == "json"
		if !isJSON {
			fmt.Printf("Found %d markdown file(s)\n", len(files))
		}

		// Extract all URLs from the files
		links, err := parser.ExtractLinksFromMultipleFiles(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing files: %v\n", err)
			os.Exit(1)
		}

		if len(links) == 0 {
			if !isJSON {
				fmt.Println("No links found.")
			}
			return
		}

		if !isJSON {
			fmt.Printf("Found %d link(s), checking...\n", len(links))
		}

		// Check all links concurrently
		results := checker.CheckLinks(links)
		deadLinks := checker.FilterDeadLinks(results)

		// Output based on format flag
		if format == "json" {
			outputJSON(files, links, deadLinks)
		} else {
			outputText(deadLinks)
		}
	},
}

// outputJSON prints results as JSON
func outputJSON(files []string, links []parser.Link, deadLinks []checker.Result) {
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

	// json.MarshalIndent creates pretty-printed JSON
	// The second arg is the prefix (empty), third is the indent string
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}

// outputText prints results as human-readable text
func outputText(deadLinks []checker.Result) {
	if len(deadLinks) == 0 {
		fmt.Println("\nAll links are alive!")
	} else {
		fmt.Printf("\nHEY, %d DEAD LINK(S) FOUND! THE LINKS ARE:\n\n", len(deadLinks))
		for _, r := range deadLinks {
			if r.Error != "" {
				fmt.Printf("  [ERROR] %s\n          File: %s\n          Error: %s\n\n", r.Link.URL, r.Link.FilePath, r.Error)
			} else {
				fmt.Printf("  [%d] %s\n       File: %s\n\n", r.StatusCode, r.Link.URL, r.Link.FilePath)
			}
		}
	}
}

// init is a special Go function that runs automatically when the package loads
func init() {
	// Register checkCmd as a subcommand of rootCmd
	rootCmd.AddCommand(checkCmd)

	// Define flags for the check command
	// StringVarP binds the flag to a variable:
	// - &format: pointer to the variable to store the value
	// - "format": long flag name (--format)
	// - "f": short flag name (-f)
	// - "text": default value
	// - "...": help description
	checkCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
}
