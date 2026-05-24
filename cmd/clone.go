package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone repository from GitHub interactively",
	Long:  `Queries your GitHub repositories and presents a searchable TUI list to clone them locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewCloneModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting repository clone: %v\n", err)
			os.Exit(1)
		}
	},
}
