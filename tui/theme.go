package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary    = lipgloss.Color("63")      // Indigo / Purple Hex: #4F46E5 (~63 ANSI)
	ColorSecondary  = lipgloss.Color("30")      // Teal Hex: #0D9488 (~30 ANSI)
	ColorDarkSlate  = lipgloss.Color("236")     // Dark Slate Hex: #1E293B (~236 ANSI)
	ColorGray       = lipgloss.Color("243")     // Slate Gray Hex: #64748B (~243 ANSI)
	ColorSuccess    = lipgloss.Color("42")      // Emerald Green Hex: #10B981 (~42 ANSI)
	ColorError      = lipgloss.Color("161")     // Crimson Red Hex: #EF4444 (~161 ANSI)
	ColorWarning    = lipgloss.Color("214")     // Amber Yellow Hex: #F59E0B (~214 ANSI)
	ColorWhite      = lipgloss.Color("255")     // Off-white Hex: #F8FAFC (~255 ANSI)
	ColorHighlight  = lipgloss.Color("205")     // Hot Pink Hex: #EC4899 (~205 ANSI)
)

var (
	// Styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	StyleSubtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			MarginBottom(1)

	StyleSuccessCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	StyleErrorCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	StyleBorderBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDarkSlate).
			Padding(1, 2)

	StyleActiveItem = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	StyleInactiveItem = lipgloss.NewStyle().
			Foreground(ColorWhite)

	StyleTextSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleTextError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleTextWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleTextMuted   = lipgloss.NewStyle().Foreground(ColorGray)
	StyleHighlight   = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)

	// Sidebar / panel headers
	StylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorDarkSlate).
			Padding(0, 1).
			MarginBottom(1)
)

// RenderHeader outputs a standard, beautifully styled application header.
func RenderHeader(title string) string {
	banner := `
  ___                     _   _          ___ _    ___ 
 / __|_ _ __ ___ _  _  __| |_| |_  _    / __| |  |_ _|
| (_ | '_/ _§ \ V / || | _§  |  _| || |  | (__| |__ | | 
 \___|_| \__,_|\_/ \_, |____/ \__|\_, |   \___|____|___|
                   |__/           |__/                  
`
	banner = strings.ReplaceAll(banner, "§", "`")
	headerStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	titleStr := StyleTitle.Render(" " + title + " ")
	return headerStyle.Render(banner) + "\n" + titleStr + "\n\n"
}
