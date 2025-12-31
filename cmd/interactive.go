package cmd

import (
	"fmt"
	"os"

	"gone/internal/config"
	"gone/internal/filter"
	"gone/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Interactive command flags (separate from check command).
var (
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

Navigate through results, see progress in real-time, and 
filter results by type.

Controls:
  ↑/↓ or j/k    Navigate through results
  f             Cycle through filters (All Issues → Warnings → Dead → Duplicates)
  ?             Toggle help
  q             Quit

Ignore patterns:
  gone interactive --ignore-domain=localhost,example.com
  gone interactive --ignore-pattern="*.local/*"
  gone interactive --ignore-regex=".*\\.test$"`,
	Args: cobra.MaximumNArgs(1),
	Run:  runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)

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

func runInteractive(_ *cobra.Command, args []string) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Create filter from config and flags
	urlFilter, err := createInteractiveFilter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filter: %v\n", err)
		os.Exit(1) //nolint:revive // deep-exit is acceptable for CLI entry points
	}

	p := tea.NewProgram(ui.New(path, urlFilter), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running interactive mode: %v\n", err)
		os.Exit(1) //nolint:revive // deep-exit is acceptable for CLI entry points
	}
}

// createInteractiveFilter builds a filter from config file and CLI flags.
func createInteractiveFilter() (*filter.Filter, error) {
	var cfg *config.Config

	// Load config file unless --no-config is set
	if !iNoConfig {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	} else {
		cfg = &config.Config{}
	}

	// Merge CLI flags (additive)
	cfg.Ignore.Domains = append(cfg.Ignore.Domains, iIgnoreDomains...)
	cfg.Ignore.Patterns = append(cfg.Ignore.Patterns, iIgnorePatterns...)
	cfg.Ignore.Regex = append(cfg.Ignore.Regex, iIgnoreRegex...)

	// If no ignore rules, return nil (no filtering)
	if cfg.IsEmpty() {
		return nil, nil
	}

	// Create filter
	return filter.New(filter.Config{
		Domains:       cfg.Ignore.Domains,
		GlobPatterns:  cfg.Ignore.Patterns,
		RegexPatterns: cfg.Ignore.Regex,
	})
}
