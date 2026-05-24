package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with your GitHub account",
	Long:  `Guides you through browser-based GitHub OAuth Device Flow authentication and stores tokens locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewAuthModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error during authentication: %v\n", err)
			os.Exit(1)
		}
	},
}
