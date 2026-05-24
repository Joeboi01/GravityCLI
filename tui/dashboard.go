package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"GravityCLI/config"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type activeView int

const (
	viewDashboard activeView = iota
	viewAuth
	viewClone
	viewRepo
	viewBranches
	viewPR
	viewNav
)

type DashboardModel struct {
	active     activeView
	menuIndex  int
	spinner    spinner.Model
	loading    bool
	isLoggedIn bool

	// Logged-in profile cache
	username    string
	realName    string
	bio         string
	followers   int
	publicRepos int

	// Sub-models
	authModel   AuthModel
	cloneModel  CloneModel
	repoModel   RepoModel
	branchModel BranchModel
	prModel     PRModel
	navModel    NavModel
}

func NewDashboardModel() DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	return DashboardModel{
		active:    viewDashboard,
		menuIndex: 0,
		spinner:   s,
		loading:   true,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadProfileCmd(),
	)
}

type profileLoadedMsg struct {
	username    string
	realName    string
	bio         string
	followers   int
	publicRepos int
}
type profileFailedMsg struct{}
type BackMsg struct{} // Universal back navigation message

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// If we are in a sub-view, delegate message routing to the sub-model.
	// All sub-views send BackMsg to return to the dashboard.
	switch m.active {
	case viewAuth:
		newModel, subCmd := m.authModel.Update(msg)
		m.authModel = newModel.(AuthModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil

	case viewClone:
		newModel, subCmd := m.cloneModel.Update(msg)
		m.cloneModel = newModel.(CloneModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil

	case viewRepo:
		newModel, subCmd := m.repoModel.Update(msg)
		m.repoModel = newModel.(RepoModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil

	case viewBranches:
		newModel, subCmd := m.branchModel.Update(msg)
		m.branchModel = newModel.(BranchModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil

	case viewPR:
		newModel, subCmd := m.prModel.Update(msg)
		m.prModel = newModel.(PRModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil

	case viewNav:
		newModel, subCmd := m.navModel.Update(msg)
		m.navModel = newModel.(NavModel)
		if subCmd != nil {
			return m, subCmd
		}
		return m, nil
	}

	// Main dashboard key handles
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			m.menuIndex = (m.menuIndex - 1 + 7) % 7

		case "down", "j":
			m.menuIndex = (m.menuIndex + 1) % 7

		case "enter":
			switch m.menuIndex {
			case 0: // Directory Cockpit
				m.active = viewNav
				m.navModel = NewNavModel()
				return m, m.navModel.Init()
			case 1: // Clone Repository
				m.active = viewClone
				m.cloneModel = NewCloneModel()
				return m, m.cloneModel.Init()
			case 2: // Create Repository
				m.active = viewRepo
				m.repoModel = NewRepoModel()
				return m, m.repoModel.Init()
			case 3: // Branch switcher
				m.active = viewBranches
				m.branchModel = NewBranchModel()
				return m, m.branchModel.Init()
			case 4: // Pull request
				m.active = viewPR
				m.prModel = NewPRModel()
				return m, m.prModel.Init()
			case 5: // Authenticate
				m.active = viewAuth
				m.authModel = NewAuthModel()
				return m, m.authModel.Init()
			case 6: // Exit
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case profileLoadedMsg:
		m.isLoggedIn = true
		m.username = msg.username
		m.realName = msg.realName
		m.bio = msg.bio
		m.followers = msg.followers
		m.publicRepos = msg.publicRepos
		m.loading = false

	case profileFailedMsg:
		m.isLoggedIn = false
		m.loading = false

	case BackMsg:
		m.active = viewDashboard
		m.loading = true
		return m, m.loadProfileCmd()
	}

	return m, tea.Batch(cmds...)
}

func (m DashboardModel) View() string {
	// If active is a sub-view, draw it instead
	switch m.active {
	case viewAuth:
		return m.authModel.View()
	case viewClone:
		return m.cloneModel.View()
	case viewRepo:
		return m.repoModel.View()
	case viewBranches:
		return m.branchModel.View()
	case viewPR:
		return m.prModel.View()
	case viewNav:
		return m.navModel.View()
	}

	var s strings.Builder

	s.WriteString(RenderHeader("Main Dashboard"))

	// Left Side: Action Menu
	var menuPanel strings.Builder
	menuPanel.WriteString(StyleSubtitle.Render("⚡ Select Quick Action:") + "\n\n")

	menuItems := []string{
		"📁 Directory Browser & Commit Engine",
		"📥 Search & Clone Repository",
		"🗂️  Repository Manager (Create / Edit / Delete)",
		"🌿 Switch & Create Branches",
		"🔀 Create Pull Request",
		"🔑 Connect GitHub Profile",
		"❌ Exit GravityCLI",
	}

	for i, item := range menuItems {
		if i == m.menuIndex {
			menuPanel.WriteString(StyleActiveItem.Render(fmt.Sprintf("  ❯ %s", item)) + "\n")
		} else {
			menuPanel.WriteString(StyleInactiveItem.Render(fmt.Sprintf("    %s", item)) + "\n")
		}
	}

	// Right Side: GitHub Profile Card
	var profilePanel strings.Builder

	if m.loading {
		profilePanel.WriteString(fmt.Sprintf("\n\n  %s Querying session cache...\n\n", m.spinner.View()))
	} else if !m.isLoggedIn {
		profilePanel.WriteString(StylePanelTitle.Render("⚡ CONNECTIONS STATUS") + "\n\n")
		profilePanel.WriteString(StyleTextError.Render("❌ Offline / Unconnected") + "\n\n")
		profilePanel.WriteString("Authorize GitHub OAuth to unlock\n")
		profilePanel.WriteString("repository cloning, creation,\n")
		profilePanel.WriteString("and Pull Requests.\n\n")
		profilePanel.WriteString(StyleHighlight.Render("💡 Choose 'Connect GitHub Profile' below.") + "\n")
	} else {
		profilePanel.WriteString(StylePanelTitle.Render("👤 GITHUB ACCOUNT") + "\n\n")
		profilePanel.WriteString(fmt.Sprintf("Username:    %s\n", StyleHighlight.Render("@"+m.username)))
		if m.realName != "" {
			profilePanel.WriteString(fmt.Sprintf("Name:        %s\n", m.realName))
		}
		profilePanel.WriteString(fmt.Sprintf("Repos:       %d\n", m.publicRepos))
		profilePanel.WriteString(fmt.Sprintf("Followers:   %d\n\n", m.followers))

		if m.bio != "" {
			profilePanel.WriteString(StyleTextMuted.Render("Bio:") + "\n")
			bioText := m.bio
			if len(bioText) > 80 {
				bioText = bioText[:77] + "..."
			}
			profilePanel.WriteString(bioText + "\n")
		}
	}

	// Dynamic layout rendering using JoinHorizontal
	sideBySide := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(50).Render(menuPanel.String()),
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDarkSlate).
			Padding(1, 2).
			Width(38).
			Render(profilePanel.String()),
	)

	s.WriteString(sideBySide + "\n")
	s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate action menu • [Enter] Launch • [q] Exit CLI") + "\n")

	return s.String()
}

func (m DashboardModel) loadProfileCmd() tea.Cmd {
	return func() tea.Msg {
		// UX delay
		time.Sleep(300 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil || !cfg.IsAuthenticated() {
			return profileFailedMsg{}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		user, _, err := client.Users.Get(ctx, "")
		if err != nil {
			return profileFailedMsg{}
		}

		bio := ""
		if user.Bio != nil {
			bio = *user.Bio
		}
		realName := ""
		if user.Name != nil {
			realName = *user.Name
		}

		return profileLoadedMsg{
			username:    user.GetLogin(),
			realName:    realName,
			bio:         bio,
			followers:   user.GetFollowers(),
			publicRepos: user.GetPublicRepos(),
		}
	}
}
