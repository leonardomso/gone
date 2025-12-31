package cmd

import (
	"fmt"
	"os"

	"gone/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
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
  q             Quit`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		p := tea.NewProgram(ui.New(path), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running interactive mode: %v\n", err)
			os.Exit(1) //nolint:revive // deep-exit is acceptable for CLI entry points
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
