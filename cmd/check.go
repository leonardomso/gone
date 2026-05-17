package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/parser"

	"github.com/spf13/cobra"
)

// Flag variables for the check command.
var (
	outputFormat string
	outputFile   string
	concurrency  int
	timeout      int
	retries      int
	showAlive    bool
	showWarnings bool
	showDead     bool
	showAll      bool
	showStats    bool

	// File type flags.
	fileTypes  []string
	strictMode bool

	// Ignore flags.
	ignoreDomains     []string
	ignorePatterns    []string
	ignoreRegex       []string
	showIgnored       bool
	noConfig          bool
	allowPrivateHosts bool
)

// checkCmd represents the check command.
var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Scan files for dead links",
	Long: `Scan a directory for files and check all HTTP/HTTPS links.

If no path is provided, scans the current directory.
By default, scans only markdown files (.md).
Use --types to scan additional file types.

By default, shows warnings (redirects, blocked) and dead links.
Use flags to filter what's displayed.

Exit codes:
  0 - All links are alive or only have warnings
  1 - Dead links or errors found

Examples:
  gone check                         # Scan current directory (markdown only)
  gone check ./docs                  # Scan specific directory  
  gone check --types=md,json,yaml    # Scan markdown, JSON, and YAML files
  gone check --types=toml,xml        # Scan TOML and XML files
  gone check --types=json --strict   # Fail on malformed JSON files
  gone check --format=json           # Output JSON to stdout
  gone check --format=yaml           # Output YAML to stdout
  gone check --output=report.json    # Write JSON report to file
  gone check --output=report.md      # Write Markdown report to file
  gone check --output=report.junit.xml  # Write JUnit XML for CI/CD
  gone check --all                   # Show all results including alive
  gone check --dead                  # Show only dead links and errors
  gone check --concurrency=100       # Use 100 concurrent workers
  gone check --stats                 # Show performance statistics

Note: --format and --output are mutually exclusive.

Supported file types: md (includes .mdx, .markdown), json, yaml (includes .yml), toml, xml

Ignore patterns:
  gone check --ignore-domain=localhost,example.com
  gone check --ignore-pattern="*.local/*"
  gone check --ignore-regex=".*\\.test$"
  gone check --show-ignored          # Show which URLs were ignored

Config file (.gonerc.yaml):
  types: [md, json, yaml]       # Default file types to scan
  scan:
    include: ["docs/**"]        # Only scan matching paths
    exclude: ["vendor/**"]      # Skip matching paths
  check:
    concurrency: 100            # Concurrent workers
    timeout: 30                 # Request timeout (seconds)
    retries: 2                  # Retry attempts
    strict: false               # Fail on malformed files
  output:
    showStats: true             # Show performance stats
  ignore:
    domains: [localhost, example.com]
    patterns: ["*.local/*"]
    regex: [".*\\.test$"]`,
	Args: cobra.MaximumNArgs(1),
	Run:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Output options
	checkCmd.Flags().StringVarP(&outputFormat, "format", "f", "",
		"Output format for stdout: json, yaml, xml, junit, markdown")
	checkCmd.Flags().StringVarP(&outputFile, "output", "o", "",
		"Write report to file (format inferred from extension: .json, .yaml, .xml, .junit.xml, .md)")

	// File type options
	checkCmd.Flags().StringSliceVarP(&fileTypes, "types", "T", []string{"md"},
		"File types to scan: md (includes .mdx, .markdown), json, yaml, toml, xml")
	checkCmd.Flags().BoolVar(&strictMode, "strict", false,
		"Fail on malformed files instead of skipping them")

	// Filter flags
	checkCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all results (alive, warnings, dead)")
	checkCmd.Flags().BoolVar(&showAlive, "alive", false, "Show only alive links")
	checkCmd.Flags().BoolVarP(&showWarnings, "warnings", "w", false, "Show only warnings (redirects, blocked)")
	checkCmd.Flags().BoolVarP(&showDead, "dead", "d", false, "Show only dead links and errors")

	// Performance options
	checkCmd.Flags().IntVarP(&concurrency, "concurrency", "c", checker.DefaultConcurrency,
		"Number of concurrent workers")
	checkCmd.Flags().IntVarP(&timeout, "timeout", "t", int(checker.DefaultTimeout.Seconds()),
		"Timeout per request in seconds")
	checkCmd.Flags().IntVarP(&retries, "retries", "r", checker.DefaultMaxRetries,
		"Number of retries for failed requests")

	// Stats flag
	checkCmd.Flags().BoolVar(&showStats, "stats", false,
		"Show detailed performance statistics")

	// Ignore options
	checkCmd.Flags().StringSliceVar(&ignoreDomains, "ignore-domain", nil,
		"Domains to ignore, includes subdomains (can be repeated or comma-separated)")
	checkCmd.Flags().StringSliceVar(&ignorePatterns, "ignore-pattern", nil,
		"Glob patterns to ignore (can be repeated)")
	checkCmd.Flags().StringSliceVar(&ignoreRegex, "ignore-regex", nil,
		"Regex patterns to ignore (can be repeated)")
	checkCmd.Flags().BoolVar(&showIgnored, "show-ignored", false,
		"Show which URLs were ignored and why")
	checkCmd.Flags().BoolVar(&noConfig, "no-config", false,
		"Skip loading .gonerc.yaml config file")

	// Security options
	checkCmd.Flags().BoolVar(&allowPrivateHosts, "allow-private-hosts", false,
		"Allow requests to loopback, private, link-local and reserved IP "+
			"ranges. Default is to block them to prevent SSRF when scanning "+
			"untrusted documents.")
}

// runCheck is the main entry point for the check command.
// It orchestrates the entire link checking workflow.
func runCheck(_ *cobra.Command, args []string) {
	code := newCheckRunner(currentCheckOptions(), defaultCommandEnv(), defaultIOStreams()).Run(args)
	if code != 0 {
		os.Exit(code)
	}
}

// getPathArg returns the path argument or "." as default.
func getPathArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

// validateFileTypes checks if all specified file types are supported.
func validateFileTypes(types []string) error {
	supportedTypes := parser.SupportedFileTypes()
	supported := make(map[string]bool, len(supportedTypes))
	for _, t := range supportedTypes {
		supported[t] = true
	}

	for _, t := range types {
		if !supported[strings.ToLower(t)] {
			return fmt.Errorf("unsupported file type: %s (supported: %s)",
				t, strings.Join(supportedTypes, ", "))
		}
	}
	return nil
}

// getIgnoredCount returns the ignored count from filter, or 0 if filter is nil.
func getIgnoredCount(urlFilter *filter.Filter) int {
	if urlFilter != nil {
		return urlFilter.IgnoredCount()
	}
	return 0
}
