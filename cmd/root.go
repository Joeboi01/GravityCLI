package cmd

import (
	"fmt"
	"os"

	"GravityCLI/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gravity",
	Short: "GravityCLI is a premium interactive terminal client for GitHub workflows.",
	Long: `GravityCLI is a beautiful, interactive, and terminal-based Git/GitHub workflow simplify client.
It wraps OAuth authentication, repository cloning and creating, local/remote branch managers, 
interactive directory tree staging and commits, and pull requests in stunning console graphics.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch unified main Dashboard
		p := tea.NewProgram(tui.NewDashboardModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting dashboard: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Disable cobra's default behavior that blocks double-clicking the exe on Windows
	cobra.MousetrapHelpText = ""
	
	// Add subcommands
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(branchesCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(navCmd)
}
