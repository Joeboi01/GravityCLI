package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "GitHub Repository operations",
	Long:  `Manage GitHub repositories from the terminal (e.g. creating new projects).`,
}

var repoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new GitHub repository interactively",
	Long:  `Launches a wizard to specify name, description, privacy, and README auto-initialization.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewRepoModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error during repository creation: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	repoCmd.AddCommand(repoCreateCmd)
}
