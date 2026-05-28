package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"GravityCLI/config"
	"GravityCLI/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v60/github"
)

type cloneStep int

const (
	cloneStepLoading cloneStep = iota
	cloneStepBrowsing
	cloneStepConfirmPath
	cloneStepCloning
	cloneStepSuccess
	cloneStepError
)

type repoItem struct {
	name        string
	fullName    string
	description string
	cloneURL    string
	stars       int
	forks       int
	language    string
}

type CloneModel struct {
	step          cloneStep
	repos         []repoItem
	filteredRepos []repoItem
	selectedIndex int
	searchQuery   string
	searching     bool
	spinner       spinner.Model
	textInput     textinput.Model
	err           error

	selectedRepo repoItem
	clonePath    string
}

func NewCloneModel() CloneModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return CloneModel{
		step:      cloneStepLoading,
		spinner:   s,
		textInput: ti,
	}
}

func (m CloneModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchUserRepos(),
	)
}

type reposLoadedMsg []repoItem
type cloneSuccessMsg string
type cloneErrorMsg struct{ err error }

func (m CloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.step == cloneStepSuccess || m.step == cloneStepError {
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "esc":
			if m.step == cloneStepBrowsing {
				if m.searching {
					m.searching = false
					m.searchQuery = ""
					m.filterRepos()
				} else {
					return m, func() tea.Msg { return BackMsg{} }
				}
			} else if m.step == cloneStepConfirmPath {
				m.step = cloneStepBrowsing
			} else {
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "up", "k":
			if m.step == cloneStepBrowsing && !m.searching && len(m.filteredRepos) > 0 {
				m.selectedIndex = (m.selectedIndex - 1 + len(m.filteredRepos)) % len(m.filteredRepos)
			}

		case "down", "j":
			if m.step == cloneStepBrowsing && !m.searching && len(m.filteredRepos) > 0 {
				m.selectedIndex = (m.selectedIndex + 1) % len(m.filteredRepos)
			}

		case "enter":
			switch m.step {
			case cloneStepBrowsing:
				if len(m.filteredRepos) > 0 {
					m.selectedRepo = m.filteredRepos[m.selectedIndex]
					m.step = cloneStepConfirmPath
					defaultPath, _ := os.Getwd()
					m.clonePath = filepath.Join(defaultPath, m.selectedRepo.name)
					m.textInput.SetValue(m.clonePath)
					m.textInput.Focus()
				}
			case cloneStepConfirmPath:
				m.clonePath = m.textInput.Value()
				m.step = cloneStepCloning
				return m, m.cloneRepoCmd()
			case cloneStepSuccess, cloneStepError:
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "backspace":
			if m.step == cloneStepBrowsing && m.searching {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.filterRepos()
				}
			}

		default:
			if m.step == cloneStepBrowsing {
				// Search input handler
				if !m.searching && (msg.String() == "/" || msg.String() == "s") {
					m.searching = true
					m.searchQuery = ""
					m.selectedIndex = 0
					return m, nil
				}

				if m.searching {
					// Runes to build search query
					if len(msg.String()) == 1 {
						m.searchQuery += msg.String()
						m.selectedIndex = 0
						m.filterRepos()
					}
				}
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case reposLoadedMsg:
		m.repos = msg
		m.filteredRepos = msg
		m.step = cloneStepBrowsing

	case cloneSuccessMsg:
		m.step = cloneStepSuccess
		m.clonePath = string(msg)

	case cloneErrorMsg:
		m.step = cloneStepError
		m.err = msg.err
	}

	if m.step == cloneStepConfirmPath {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m CloneModel) View() string {
	var s strings.Builder

	s.WriteString(RenderHeader("Clone Repository"))

	switch m.step {
	case cloneStepLoading:
		s.WriteString(fmt.Sprintf("\n%s Querying GitHub for your repositories...\n\n", m.spinner.View()))

	case cloneStepBrowsing:
		if m.searching {
			s.WriteString(StyleSubtitle.Render(fmt.Sprintf("🔍 Searching: %s █", m.searchQuery)) + "\n\n")
		} else {
			s.WriteString(StyleSubtitle.Render("📂 Select repository to clone (Press [/] to search):") + "\n\n")
		}

		if len(m.filteredRepos) == 0 {
			s.WriteString(StyleTextMuted.Render("   No repositories found.") + "\n\n")
		} else {
			// Display 8 items at a time
			start := m.selectedIndex - 3
			if start < 0 {
				start = 0
			}
			end := start + 8
			if end > len(m.filteredRepos) {
				end = len(m.filteredRepos)
				start = end - 8
				if start < 0 {
					start = 0
				}
			}

			// Left pane: repository list
			var listSection strings.Builder
			for i := start; i < end; i++ {
				repo := m.filteredRepos[i]
				bullet := "  "
				var nameStr string

				if i == m.selectedIndex {
					bullet = " ❯ "
					nameStr = StyleActiveItem.Render(repo.fullName)
				} else {
					nameStr = StyleInactiveItem.Render(repo.fullName)
				}

				desc := repo.description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				if desc == "" {
					desc = "No description provided."
				}

				stats := fmt.Sprintf("⭐ %d  🍴 %d  🎨 %s", repo.stars, repo.forks, repo.language)

				listSection.WriteString(fmt.Sprintf("%s%s\n", bullet, nameStr))
				listSection.WriteString(fmt.Sprintf("    %s\n", StyleTextMuted.Render(desc)))
				listSection.WriteString(fmt.Sprintf("    %s\n\n", StyleTextMuted.Render(stats)))
			}

			// Right pane: details box
			var detailsSection strings.Builder
			if m.selectedIndex < len(m.filteredRepos) {
				activeRepo := m.filteredRepos[m.selectedIndex]
				detailsSection.WriteString(StyleSubtitle.Render("Repository Profile") + "\n")
				detailsSection.WriteString(fmt.Sprintf("Name:        %s\n", StyleHighlight.Render(activeRepo.name)))
				detailsSection.WriteString(fmt.Sprintf("Full:        %s\n", activeRepo.fullName))
				detailsSection.WriteString(fmt.Sprintf("Language:    %s\n", activeRepo.language))
				detailsSection.WriteString(fmt.Sprintf("Stars:       %d\n", activeRepo.stars))
				detailsSection.WriteString(fmt.Sprintf("Forks:       %d\n\n", activeRepo.forks))
				detailsSection.WriteString(StyleTextMuted.Render("URL:") + "\n")
				detailsSection.WriteString(StyleTextMuted.Render(activeRepo.cloneURL) + "\n\n")

				descWordWrap := activeRepo.description
				if descWordWrap == "" {
					descWordWrap = "No description provided."
				}
				detailsSection.WriteString(StyleTextMuted.Render("Description:") + "\n")
				detailsSection.WriteString(descWordWrap + "\n")
			}

			// Render side-by-side using Lip Gloss JoinHorizontal
			sideBySide := lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(50).Render(listSection.String()),
				lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ColorDarkSlate).
					Padding(1, 2).
					Width(35).
					Render(detailsSection.String()),
			)
			s.WriteString(sideBySide + "\n")
		}

		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [/] Filter • [Enter] Choose Repo • [Esc] Back to Dashboard") + "\n")

	case cloneStepConfirmPath:
		s.WriteString(StyleSubtitle.Render("Confirm Destination Path") + "\n\n")
		s.WriteString("Specify the local folder path to clone this repository into:\n\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Confirm & Clone • [Esc] Cancel") + "\n")

	case cloneStepCloning:
		s.WriteString(fmt.Sprintf("\n%s Cloning %s into:\n", m.spinner.View(), StyleHighlight.Render(m.selectedRepo.fullName)))
		s.WriteString(StyleTextMuted.Render(m.clonePath) + "\n\n")
		s.WriteString("Please wait while files are being downloaded...\n")

	case cloneStepSuccess:
		successStr := fmt.Sprintf("🎉 SUCCESS!\n\nCloned %s successfully!\n\nLocation:\n%s",
			StyleHighlight.Render(m.selectedRepo.fullName),
			StyleTextSuccess.Render(m.clonePath),
		)
		s.WriteString(StyleSuccessCard.Render(successStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter/q] Back to main dashboard") + "\n")

	case cloneStepError:
		errStr := fmt.Sprintf("❌ CLONE FAILURE\n\nFailed to clone repository: %s", m.err.Error())
		s.WriteString(StyleErrorCard.Render(errStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")
	}

	return s.String()
}

func (m *CloneModel) filterRepos() {
	if m.searchQuery == "" {
		m.filteredRepos = m.repos
		return
	}

	var filtered []repoItem
	query := strings.ToLower(m.searchQuery)
	for _, repo := range m.repos {
		if strings.Contains(strings.ToLower(repo.fullName), query) ||
			strings.Contains(strings.ToLower(repo.description), query) ||
			strings.Contains(strings.ToLower(repo.language), query) {
			filtered = append(filtered, repo)
		}
	}
	m.filteredRepos = filtered
}

func (m CloneModel) fetchUserRepos() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return cloneErrorMsg{err: err}
		}

		if !cfg.IsAuthenticated() {
			return cloneErrorMsg{err: fmt.Errorf("you are not authenticated. Run 'gravity auth' first")}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		// List user repositories (both owned, collaborator, etc.)
		opt := &github.RepositoryListByAuthenticatedUserOptions{
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: github.ListOptions{PerPage: 100},
		}

		var items []repoItem
		for {
			repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
			if err != nil {
				return cloneErrorMsg{err: err}
			}

			for _, repo := range repos {
				lang := "Unknown"
				if repo.Language != nil {
					lang = *repo.Language
				}
				desc := ""
				if repo.Description != nil {
					desc = *repo.Description
				}

				items = append(items, repoItem{
					name:        repo.GetName(),
					fullName:    repo.GetFullName(),
					description: desc,
					cloneURL:    repo.GetCloneURL(),
					stars:       repo.GetStargazersCount(),
					forks:       repo.GetForksCount(),
					language:    lang,
				})
			}

			if resp == nil || resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		return reposLoadedMsg(items)
	}
}

func (m CloneModel) cloneRepoCmd() tea.Cmd {
	return func() tea.Msg {
		// Wait a bit for UI transition feel
		time.Sleep(500 * time.Millisecond)

		err := git.Clone(m.selectedRepo.cloneURL, m.clonePath)
		if err != nil {
			return cloneErrorMsg{err: err}
		}

		return cloneSuccessMsg(m.clonePath)
	}
}
