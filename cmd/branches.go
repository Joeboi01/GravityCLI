package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var branchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Switch and create local branches",
	Long:  `Detects the active Git repository and presents an interactive list to switch branches or spawn new ones.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewBranchModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting branch manager: %v\n", err)
			os.Exit(1)
		}
	},
}
