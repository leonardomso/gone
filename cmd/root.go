package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gone",
	Short: "A dead link detector for markdown files",
	Long: `Gone is a CLI tool that scans markdown files for dead links.

It extracts all HTTP/HTTPS URLs from your markdown files and checks
if they're still alive. Use 'check' for CI/scripts or 'interactive'
for a terminal UI experience.

Examples:
  gone check              # Scan current directory
  gone check ./docs       # Scan specific directory
  gone check --format=json
  gone interactive        # Launch interactive TUI`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
