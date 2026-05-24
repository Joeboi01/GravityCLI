package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create pull requests on GitHub",
	Long:  `Wizard to choose target branch, input title and markdown descriptions, and post pull requests to GitHub.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewPRModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting Pull Request creator: %v\n", err)
			os.Exit(1)
		}
	},
}
