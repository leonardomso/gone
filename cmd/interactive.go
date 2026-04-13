package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Interactive command flags (separate from check command).
var (
	iFileTypes  []string
	iStrictMode bool

	iIgnoreDomains  []string
	iIgnorePatterns []string
	iIgnoreRegex    []string
	iNoConfig       bool
)

// interactiveCmd represents the interactive command.
var interactiveCmd = &cobra.Command{
	Use:   "interactive [path]",
	Short: "Launch interactive TUI for dead link detection",
	Long: `Launch an interactive terminal UI to scan for dead links.

If no path is provided, scans the current directory.
By default, scans only markdown files (.md).
Use --types to scan additional file types.

Navigate through results, see progress in real-time, and 
filter results by type.

Controls:
  ↑/↓ or j/k    Navigate through results
  f             Cycle through filters (All Issues → Warnings → Dead → Duplicates)
  ?             Toggle help
  q             Quit

Supported file types: md, json, yaml, toml, xml

Ignore patterns:
  gone interactive --ignore-domain=localhost,example.com
  gone interactive --ignore-pattern="*.local/*"
  gone interactive --ignore-regex=".*\\.test$"`,
	Args: cobra.MaximumNArgs(1),
	Run:  runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)

	// File type options
	interactiveCmd.Flags().StringSliceVarP(&iFileTypes, "types", "T", []string{"md"},
		"File types to scan (comma-separated): md, json, yaml, toml, xml")
	interactiveCmd.Flags().BoolVar(&iStrictMode, "strict", false,
		"Fail on malformed files instead of skipping them")

	// Ignore options (same as check command)
	interactiveCmd.Flags().StringSliceVar(&iIgnoreDomains, "ignore-domain", nil,
		"Domains to ignore, includes subdomains (can be repeated or comma-separated)")
	interactiveCmd.Flags().StringSliceVar(&iIgnorePatterns, "ignore-pattern", nil,
		"Glob patterns to ignore (can be repeated)")
	interactiveCmd.Flags().StringSliceVar(&iIgnoreRegex, "ignore-regex", nil,
		"Regex patterns to ignore (can be repeated)")
	interactiveCmd.Flags().BoolVar(&iNoConfig, "no-config", false,
		"Skip loading .gonerc.yaml config file")
}

// runInteractive launches the interactive TUI for link checking.
func runInteractive(_ *cobra.Command, args []string) {
	code := newInteractiveRunner(
		currentInteractiveOptions(),
		defaultCommandEnv(),
		defaultIOStreams(),
	).Run(args)
	if code != 0 {
		os.Exit(code)
	}
}
