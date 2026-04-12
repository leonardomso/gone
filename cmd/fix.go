package cmd

import (
	"os"

	"github.com/leonardomso/gone/internal/checker"

	"github.com/spf13/cobra"
)

// Fix command flag variables.
var (
	fixYes         bool
	fixDryRun      bool
	fixConcurrency int
	fixTimeout     int
	fixRetries     int
	fixShowStats   bool

	// File type flags.
	fixFileTypes  []string
	fixStrictMode bool

	// Ignore flags (shared with check).
	fixIgnoreDomains  []string
	fixIgnorePatterns []string
	fixIgnoreRegex    []string
	fixNoConfig       bool
)

// fixCmd represents the fix command.
var fixCmd = &cobra.Command{
	Use:   "fix [path]",
	Short: "Automatically fix redirect URLs in files",
	Long: `Scan files for redirect URLs and update them to their final destinations.

Only redirects where the final destination returns 200 OK are fixed.
Dead links, errors, and blocked URLs are not modified.

By default, scans only markdown files (.md).
Use --types to scan additional file types.

By default, the command runs interactively, prompting for each file.
Use --yes to apply all fixes automatically (useful for CI/scripts).
Use --dry-run to preview changes without modifying files.

Examples:
  gone fix                      # Interactive mode, scan current directory
  gone fix ./docs               # Interactive mode, scan specific directory
  gone fix --types=md,json      # Scan markdown and JSON files
  gone fix --types=toml,xml     # Scan TOML and XML files
  gone fix --dry-run            # Preview what would be fixed
  gone fix --yes                # Apply all fixes without prompting
  gone fix --yes --dry-run      # Preview all fixes (no prompts, no changes)
  gone fix --stats              # Show performance statistics

Supported file types: md, json, yaml, toml, xml

Ignore patterns (same as check command):
  gone fix --ignore-domain=localhost
  gone fix --ignore-pattern="*.local/*"
  gone fix --no-config          # Skip .gonerc.yaml`,
	Args: cobra.MaximumNArgs(1),
	Run:  runFix,
}

func init() {
	rootCmd.AddCommand(fixCmd)

	// Mode flags
	fixCmd.Flags().BoolVarP(&fixYes, "yes", "y", false,
		"Apply all fixes without prompting")
	fixCmd.Flags().BoolVarP(&fixDryRun, "dry-run", "n", false,
		"Preview changes without modifying files")

	// File type options
	fixCmd.Flags().StringSliceVarP(&fixFileTypes, "types", "T", []string{"md"},
		"File types to scan (comma-separated): md, json, yaml, toml, xml")
	fixCmd.Flags().BoolVar(&fixStrictMode, "strict", false,
		"Fail on malformed files instead of skipping them")

	// Performance options
	fixCmd.Flags().IntVarP(&fixConcurrency, "concurrency", "c", checker.DefaultConcurrency,
		"Number of concurrent workers")
	fixCmd.Flags().IntVarP(&fixTimeout, "timeout", "t", int(checker.DefaultTimeout.Seconds()),
		"Timeout per request in seconds")
	fixCmd.Flags().IntVarP(&fixRetries, "retries", "r", checker.DefaultMaxRetries,
		"Number of retries for failed requests")

	// Stats flag
	fixCmd.Flags().BoolVar(&fixShowStats, "stats", false,
		"Show detailed performance statistics")

	// Ignore options
	fixCmd.Flags().StringSliceVar(&fixIgnoreDomains, "ignore-domain", nil,
		"Domains to ignore (can be repeated or comma-separated)")
	fixCmd.Flags().StringSliceVar(&fixIgnorePatterns, "ignore-pattern", nil,
		"Glob patterns to ignore (can be repeated)")
	fixCmd.Flags().StringSliceVar(&fixIgnoreRegex, "ignore-regex", nil,
		"Regex patterns to ignore (can be repeated)")
	fixCmd.Flags().BoolVar(&fixNoConfig, "no-config", false,
		"Skip loading .gonerc.yaml config file")
}

// runFix is the main entry point for the fix command.
// It scans for redirects and applies fixes interactively or automatically.
func runFix(_ *cobra.Command, args []string) {
	code := newFixRunner(currentFixOptions(), defaultCommandEnv(), defaultIOStreams()).Run(args)
	if code != 0 {
		os.Exit(code)
	}
}
