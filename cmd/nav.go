package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var navCmd = &cobra.Command{
	Use:   "nav",
	Short: "Interactive Directory Browser & Git Cockpit",
	Long:  `Launches a side-by-side terminal tree navigator and staging/commit control center.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewNavModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting directory cockpit: %v\n", err)
			os.Exit(1)
		}
	},
}
