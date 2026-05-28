package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"GravityCLI/config"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v60/github"
)

// ─── Steps ────────────────────────────────────────────────────────────────────

type repoStep int

const (
	repoStepMenu          repoStep = iota // Top-level menu: Create / Edit / Delete
	repoStepListRepos                     // List repos for Edit or Delete selection
	repoStepCreateForm                    // Create form
	repoStepEditForm                      // Edit form (name, desc, visibility)
	repoStepDeleteConfirm                 // Type repo name to confirm delete
	repoStepWorking                       // Spinner while API call runs
	repoStepSuccess                       // Success card
	repoStepError                         // Error card
)

// repoAction tracks what action was chosen from the top menu
type repoAction int

const (
	repoActionCreate repoAction = iota
	repoActionEdit
	repoActionDelete
)

// ─── Messages ─────────────────────────────────────────────────────────────────

type repoListLoadedMsg struct{ repos []*github.Repository }
type repoListErrorMsg struct{ err error }
type repoOpSuccessMsg string
type repoOpErrorMsg struct{ err error }

// ─── Model ────────────────────────────────────────────────────────────────────

type RepoModel struct {
	step    repoStep
	action  repoAction
	spinner spinner.Model
	err     error

	// Menu
	menuIndex int // 0=Create, 1=Edit, 2=Delete

	// Repo list (for edit/delete selection)
	repos     []*github.Repository
	repoIndex int

	// Create form fields
	nameInput   textinput.Model
	descInput   textinput.Model
	isPrivate   bool
	addReadme   bool
	activeField int // 0=Name 1=Desc 2=Visibility 3=README 4=Submit

	// Edit form fields (pre-loaded from selected repo)
	editName    textinput.Model
	editDesc    textinput.Model
	editPrivate bool
	editField   int // 0=Name 1=Desc 2=Visibility 3=Submit

	// Delete confirm
	deleteInput textinput.Model

	// Success result
	resultMessage string
}

func NewRepoModel() RepoModel {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	nameIn := textinput.New()
	nameIn.Placeholder = "my-awesome-project"
	nameIn.Focus()
	nameIn.CharLimit = 100
	nameIn.Width = 40

	descIn := textinput.New()
	descIn.Placeholder = "A brief description (optional)"
	descIn.CharLimit = 250
	descIn.Width = 60

	editNameIn := textinput.New()
	editNameIn.CharLimit = 100
	editNameIn.Width = 40

	editDescIn := textinput.New()
	editDescIn.CharLimit = 250
	editDescIn.Width = 60

	delIn := textinput.New()
	delIn.Placeholder = "type repo name to confirm"
	delIn.CharLimit = 100
	delIn.Width = 40

	return RepoModel{
		step:        repoStepMenu,
		menuIndex:   0,
		spinner:     sp,
		nameInput:   nameIn,
		descInput:   descIn,
		isPrivate:   false,
		addReadme:   true,
		activeField: 0,
		editName:    editNameIn,
		editDesc:    editDescIn,
		editField:   0,
		deleteInput: delIn,
	}
}

func (m RepoModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m RepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			switch m.step {
			case repoStepListRepos, repoStepCreateForm, repoStepEditForm, repoStepDeleteConfirm:
				m.step = repoStepMenu
				m.err = nil
			case repoStepSuccess, repoStepError:
				return m, func() tea.Msg { return BackMsg{} }
			default:
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "up", "k":
			switch m.step {
			case repoStepMenu:
				m.menuIndex = (m.menuIndex - 1 + 3) % 3
			case repoStepListRepos:
				if len(m.repos) > 0 {
					m.repoIndex = (m.repoIndex - 1 + len(m.repos)) % len(m.repos)
				}
			case repoStepCreateForm:
				m.blurCreateAll()
				m.activeField = (m.activeField - 1 + 5) % 5
				m.focusCreateActive()
			case repoStepEditForm:
				m.blurEditAll()
				m.editField = (m.editField - 1 + 4) % 4
				m.focusEditActive()
			}

		case "down", "j":
			switch m.step {
			case repoStepMenu:
				m.menuIndex = (m.menuIndex + 1) % 3
			case repoStepListRepos:
				if len(m.repos) > 0 {
					m.repoIndex = (m.repoIndex + 1) % len(m.repos)
				}
			case repoStepCreateForm:
				m.blurCreateAll()
				m.activeField = (m.activeField + 1) % 5
				m.focusCreateActive()
			case repoStepEditForm:
				m.blurEditAll()
				m.editField = (m.editField + 1) % 4
				m.focusEditActive()
			}

		case "tab":
			switch m.step {
			case repoStepCreateForm:
				m.blurCreateAll()
				m.activeField = (m.activeField + 1) % 5
				m.focusCreateActive()
			case repoStepEditForm:
				m.blurEditAll()
				m.editField = (m.editField + 1) % 4
				m.focusEditActive()
			}

		case "shift+tab":
			switch m.step {
			case repoStepCreateForm:
				m.blurCreateAll()
				m.activeField = (m.activeField - 1 + 5) % 5
				m.focusCreateActive()
			case repoStepEditForm:
				m.blurEditAll()
				m.editField = (m.editField - 1 + 4) % 4
				m.focusEditActive()
			}

		case "left", "h", "right", "l", "space":
			switch m.step {
			case repoStepCreateForm:
				if m.activeField == 2 {
					m.isPrivate = !m.isPrivate
				} else if m.activeField == 3 {
					m.addReadme = !m.addReadme
				}
			case repoStepEditForm:
				if m.editField == 2 {
					m.editPrivate = !m.editPrivate
				}
			}

		case "enter":
			switch m.step {
			case repoStepMenu:
				switch m.menuIndex {
				case 0: // Create
					m.action = repoActionCreate
					m.step = repoStepCreateForm
					m.activeField = 0
					m.nameInput.SetValue("")
					m.descInput.SetValue("")
					m.isPrivate = false
					m.addReadme = true
					m.blurCreateAll()
					m.focusCreateActive()
				case 1: // Edit
					m.action = repoActionEdit
					m.step = repoStepWorking
					return m, m.loadReposCmd()
				case 2: // Delete
					m.action = repoActionDelete
					m.step = repoStepWorking
					return m, m.loadReposCmd()
				}

			case repoStepListRepos:
				if len(m.repos) == 0 {
					return m, nil
				}
				selected := m.repos[m.repoIndex]
				if m.action == repoActionEdit {
					m.step = repoStepEditForm
					m.editField = 0
					m.editName.SetValue(selected.GetName())
					m.editDesc.SetValue(selected.GetDescription())
					m.editPrivate = selected.GetPrivate()
					m.blurEditAll()
					m.focusEditActive()
				} else { // delete
					m.step = repoStepDeleteConfirm
					m.deleteInput.SetValue("")
					m.deleteInput.Focus()
					m.deleteInput.Placeholder = fmt.Sprintf("type '%s' to confirm", selected.GetName())
				}

			case repoStepCreateForm:
				if m.activeField == 4 || m.nameInput.Value() != "" {
					if strings.TrimSpace(m.nameInput.Value()) == "" {
						m.activeField = 0
						m.blurCreateAll()
						m.focusCreateActive()
						return m, nil
					}
					m.step = repoStepWorking
					return m, m.createRepoCmd()
				}
				m.blurCreateAll()
				m.activeField = (m.activeField + 1) % 5
				m.focusCreateActive()

			case repoStepEditForm:
				if m.editField == 3 {
					m.step = repoStepWorking
					return m, m.editRepoCmd()
				}
				m.blurEditAll()
				m.editField = (m.editField + 1) % 4
				m.focusEditActive()

			case repoStepDeleteConfirm:
				if len(m.repos) == 0 {
					return m, nil
				}
				selected := m.repos[m.repoIndex]
				typed := strings.TrimSpace(m.deleteInput.Value())
				if typed != selected.GetName() {
					m.err = fmt.Errorf("name does not match. Type exactly: %s", selected.GetName())
					return m, nil
				}
				m.step = repoStepWorking
				return m, m.deleteRepoCmd(selected.GetName())

			case repoStepSuccess, repoStepError:
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case repoListLoadedMsg:
		m.repos = msg.repos
		m.repoIndex = 0
		m.step = repoStepListRepos

	case repoListErrorMsg:
		m.step = repoStepError
		m.err = msg.err

	case repoOpSuccessMsg:
		m.step = repoStepSuccess
		m.resultMessage = string(msg)

	case repoOpErrorMsg:
		m.step = repoStepError
		m.err = msg.err
	}

	// Feed key events into active text inputs
	switch m.step {
	case repoStepCreateForm:
		if m.activeField == 0 {
			m.nameInput, cmd = m.nameInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.activeField == 1 {
			m.descInput, cmd = m.descInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	case repoStepEditForm:
		if m.editField == 0 {
			m.editName, cmd = m.editName.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.editField == 1 {
			m.editDesc, cmd = m.editDesc.Update(msg)
			cmds = append(cmds, cmd)
		}
	case repoStepDeleteConfirm:
		m.deleteInput, cmd = m.deleteInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── Focus helpers ────────────────────────────────────────────────────────────

func (m *RepoModel) blurCreateAll() {
	m.nameInput.Blur()
	m.descInput.Blur()
}
func (m *RepoModel) focusCreateActive() {
	if m.activeField == 0 {
		m.nameInput.Focus()
	} else if m.activeField == 1 {
		m.descInput.Focus()
	}
}
func (m *RepoModel) blurEditAll() {
	m.editName.Blur()
	m.editDesc.Blur()
}
func (m *RepoModel) focusEditActive() {
	if m.editField == 0 {
		m.editName.Focus()
	} else if m.editField == 1 {
		m.editDesc.Focus()
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m RepoModel) View() string {
	var s strings.Builder
	s.WriteString(RenderHeader("Repository Manager"))

	switch m.step {
	// ── Top menu ──────────────────────────────────────────────────────────
	case repoStepMenu:
		s.WriteString(StyleSubtitle.Render("🗂️  What would you like to do?") + "\n\n")
		items := []string{
			"✨ Create a new repository",
			"✏️  Edit an existing repository",
			"🗑️  Delete a repository",
		}
		for i, item := range items {
			if i == m.menuIndex {
				s.WriteString(StyleActiveItem.Render(fmt.Sprintf("  ❯ %s", item)) + "\n")
			} else {
				s.WriteString(StyleInactiveItem.Render(fmt.Sprintf("    %s", item)) + "\n")
			}
		}
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Select • [Esc] Back to Dashboard") + "\n")

	// ── Repo list (for edit/delete) ────────────────────────────────────────
	case repoStepListRepos:
		verb := "edit"
		if m.action == repoActionDelete {
			verb = "delete"
		}
		s.WriteString(StyleSubtitle.Render(fmt.Sprintf("📋 Select a repository to %s:", verb)) + "\n\n")

		if len(m.repos) == 0 {
			s.WriteString(StyleTextMuted.Render("  No repositories found on your account.") + "\n\n")
		} else {
			for i, r := range m.repos {
				vis := "🟢 Public"
				if r.GetPrivate() {
					vis = "🔴 Private"
				}
				line := fmt.Sprintf("%s  %s", r.GetName(), StyleTextMuted.Render("("+vis+")"))
				if i == m.repoIndex {
					s.WriteString(StyleActiveItem.Render("  ❯ "+line) + "\n")
				} else {
					s.WriteString(StyleInactiveItem.Render("    "+line) + "\n")
				}
			}
		}
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Select • [Esc] Back") + "\n")

	// ── Create form ───────────────────────────────────────────────────────
	case repoStepCreateForm:
		s.WriteString(StyleSubtitle.Render("✨ Create New Repository") + "\n\n")

		// Name
		labelName := "📛 Repository Name:"
		if m.activeField == 0 {
			s.WriteString(StyleActiveItem.Render("❯ "+labelName) + "\n")
		} else {
			s.WriteString(StyleInactiveItem.Render("  "+labelName) + "\n")
		}
		s.WriteString("   " + m.nameInput.View() + "\n\n")

		// Desc
		labelDesc := "✍️  Description (optional):"
		if m.activeField == 1 {
			s.WriteString(StyleActiveItem.Render("❯ "+labelDesc) + "\n")
		} else {
			s.WriteString(StyleInactiveItem.Render("  "+labelDesc) + "\n")
		}
		s.WriteString("   " + m.descInput.View() + "\n\n")

		// Visibility
		labelVis := "🔒 Visibility:"
		if m.activeField == 2 {
			s.WriteString(StyleActiveItem.Render("❯ " + labelVis))
		} else {
			s.WriteString(StyleInactiveItem.Render("  " + labelVis))
		}
		visStr := "  [🟢 Public]  ⚪ Private "
		if m.isPrivate {
			visStr = "   ⚪ Public  [🔴 Private]"
		}
		s.WriteString(visStr + "\n\n")

		// README
		labelReadme := "📄 Auto-Initialize with README:"
		if m.activeField == 3 {
			s.WriteString(StyleActiveItem.Render("❯ " + labelReadme))
		} else {
			s.WriteString(StyleInactiveItem.Render("  " + labelReadme))
		}
		readmeStr := "  [✅ Yes]  ⚪ No "
		if !m.addReadme {
			readmeStr = "   ⚪ Yes  [❌ No]"
		}
		s.WriteString(readmeStr + "\n\n")

		// Submit
		btnStyle := lipgloss.NewStyle().Padding(0, 3).Background(ColorDarkSlate).Foreground(ColorWhite)
		if m.activeField == 4 {
			btnStyle = btnStyle.Background(ColorSecondary).Bold(true)
		}
		s.WriteString("   " + btnStyle.Render("🚀 CREATE REPOSITORY") + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Tab/↑/↓] Switch fields • [←/→/Space] Toggle • [Enter] Submit • [Esc] Back") + "\n")

	// ── Edit form ─────────────────────────────────────────────────────────
	case repoStepEditForm:
		if len(m.repos) == 0 {
			break
		}
		original := m.repos[m.repoIndex].GetName()
		s.WriteString(StyleSubtitle.Render(fmt.Sprintf("✏️  Edit Repository: %s", StyleHighlight.Render(original))) + "\n\n")

		// Name
		labelName := "📛 New Name:"
		if m.editField == 0 {
			s.WriteString(StyleActiveItem.Render("❯ "+labelName) + "\n")
		} else {
			s.WriteString(StyleInactiveItem.Render("  "+labelName) + "\n")
		}
		s.WriteString("   " + m.editName.View() + "\n\n")

		// Desc
		labelDesc := "✍️  Description:"
		if m.editField == 1 {
			s.WriteString(StyleActiveItem.Render("❯ "+labelDesc) + "\n")
		} else {
			s.WriteString(StyleInactiveItem.Render("  "+labelDesc) + "\n")
		}
		s.WriteString("   " + m.editDesc.View() + "\n\n")

		// Visibility
		labelVis := "🔒 Visibility:"
		if m.editField == 2 {
			s.WriteString(StyleActiveItem.Render("❯ " + labelVis))
		} else {
			s.WriteString(StyleInactiveItem.Render("  " + labelVis))
		}
		visStr := "  [🟢 Public]  ⚪ Private "
		if m.editPrivate {
			visStr = "   ⚪ Public  [🔴 Private]"
		}
		s.WriteString(visStr + "\n\n")

		// Submit
		btnStyle := lipgloss.NewStyle().Padding(0, 3).Background(ColorDarkSlate).Foreground(ColorWhite)
		if m.editField == 3 {
			btnStyle = btnStyle.Background(ColorSecondary).Bold(true)
		}
		s.WriteString("   " + btnStyle.Render("💾 SAVE CHANGES") + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Tab/↑/↓] Switch fields • [←/→/Space] Toggle visibility • [Enter] Submit • [Esc] Back") + "\n")

	// ── Delete confirm ────────────────────────────────────────────────────
	case repoStepDeleteConfirm:
		if len(m.repos) == 0 {
			break
		}
		selected := m.repos[m.repoIndex]
		warnBox := StyleErrorCard.Render(fmt.Sprintf(
			"⚠️  DANGER ZONE\n\nYou are about to permanently delete:\n  %s\n\nThis action CANNOT be undone!",
			StyleHighlight.Render(selected.GetName()),
		))
		s.WriteString(warnBox + "\n\n")
		s.WriteString("To confirm, type the repository name exactly:\n")
		s.WriteString(m.deleteInput.View() + "\n\n")
		if m.err != nil {
			s.WriteString(StyleTextError.Render("❌ "+m.err.Error()) + "\n\n")
		}
		s.WriteString(StyleTextMuted.Render("[Enter] Confirm Delete • [Esc] Cancel") + "\n")

	// ── Working spinner ───────────────────────────────────────────────────
	case repoStepWorking:
		s.WriteString(fmt.Sprintf("\n%s Please wait...\n\n", m.spinner.View()))

	// ── Success ───────────────────────────────────────────────────────────
	case repoStepSuccess:
		s.WriteString(StyleSuccessCard.Render("🎉 SUCCESS!\n\n"+m.resultMessage) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter/Esc] Back to Dashboard") + "\n")

	// ── Error ─────────────────────────────────────────────────────────────
	case repoStepError:
		errStr := fmt.Sprintf("❌ OPERATION FAILED\n\n%s", m.err.Error())
		s.WriteString(StyleErrorCard.Render(errStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter/Esc] Back to Dashboard") + "\n")
	}

	return s.String()
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func (m RepoModel) loadReposCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil || !cfg.IsAuthenticated() {
			return repoListErrorMsg{err: fmt.Errorf("not authenticated. Please connect your GitHub profile first")}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		opts := &github.RepositoryListByAuthenticatedUserOptions{
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: github.ListOptions{PerPage: 50},
		}
		var allRepos []*github.Repository
		for {
			repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opts)
			if err != nil {
				return repoListErrorMsg{err: fmt.Errorf("failed to list repositories: %w", err)}
			}
			allRepos = append(allRepos, repos...)
			if resp == nil || resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		return repoListLoadedMsg{repos: allRepos}
	}
}

func (m RepoModel) createRepoCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil || !cfg.IsAuthenticated() {
			return repoOpErrorMsg{err: fmt.Errorf("not authenticated")}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		repoName := strings.TrimSpace(m.nameInput.Value())
		repoDesc := strings.TrimSpace(m.descInput.Value())

		newRepo := &github.Repository{
			Name:        github.String(repoName),
			Private:     github.Bool(m.isPrivate),
			AutoInit:    github.Bool(m.addReadme),
			Description: github.String(repoDesc),
		}

		created, _, err := client.Repositories.Create(ctx, "", newRepo)
		if err != nil {
			return repoOpErrorMsg{err: err}
		}

		visLabel := "Public"
		if m.isPrivate {
			visLabel = "Private"
		}
		return repoOpSuccessMsg(fmt.Sprintf(
			"Repository %s created!\nVisibility: %s\nURL: %s",
			StyleHighlight.Render(created.GetName()),
			visLabel,
			StyleTextSuccess.Render(created.GetHTMLURL()),
		))
	}
}

func (m RepoModel) editRepoCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(400 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil || !cfg.IsAuthenticated() {
			return repoOpErrorMsg{err: fmt.Errorf("not authenticated")}
		}

		if len(m.repos) == 0 {
			return repoOpErrorMsg{err: fmt.Errorf("no repository selected")}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		original := m.repos[m.repoIndex]
		owner := original.GetOwner().GetLogin()
		oldName := original.GetName()

		newName := strings.TrimSpace(m.editName.Value())
		if newName == "" {
			newName = oldName
		}
		newDesc := strings.TrimSpace(m.editDesc.Value())

		update := &github.Repository{
			Name:        github.String(newName),
			Description: github.String(newDesc),
			Private:     github.Bool(m.editPrivate),
		}

		updated, _, err := client.Repositories.Edit(ctx, owner, oldName, update)
		if err != nil {
			return repoOpErrorMsg{err: fmt.Errorf("failed to update repository: %w", err)}
		}

		visLabel := "Public"
		if updated.GetPrivate() {
			visLabel = "Private"
		}
		return repoOpSuccessMsg(fmt.Sprintf(
			"Repository updated!\nNew name: %s\nVisibility: %s\nURL: %s",
			StyleHighlight.Render(updated.GetName()),
			visLabel,
			StyleTextSuccess.Render(updated.GetHTMLURL()),
		))
	}
}

func (m RepoModel) deleteRepoCmd(repoName string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(400 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil || !cfg.IsAuthenticated() {
			return repoOpErrorMsg{err: fmt.Errorf("not authenticated")}
		}

		if len(m.repos) == 0 {
			return repoOpErrorMsg{err: fmt.Errorf("no repository selected")}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		owner := m.repos[m.repoIndex].GetOwner().GetLogin()

		_, err = client.Repositories.Delete(ctx, owner, repoName)
		if err != nil {
			return repoOpErrorMsg{err: fmt.Errorf("failed to delete repository: %w", err)}
		}

		return repoOpSuccessMsg(fmt.Sprintf(
			"Repository %s has been permanently deleted.",
			StyleHighlight.Render(repoName),
		))
	}
}
