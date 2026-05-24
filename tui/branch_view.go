package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"GravityCLI/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type branchStep int

const (
	branchStepCheckRepo branchStep = iota
	branchStepBrowsing
	branchStepCreateInput
	branchStepRunning
	branchStepSuccess
	branchStepError
)

type BranchModel struct {
	step          branchStep
	dir           string
	branches      []string
	currentBranch string
	selectedIndex int
	err           error

	textInput textinput.Model
	spinner   spinner.Model

	successMessage string
}

func NewBranchModel() BranchModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	ti := textinput.New()
	ti.Placeholder = "feature/awesome-ui"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 45

	cwd, _ := os.Getwd()

	return BranchModel{
		step:      branchStepCheckRepo,
		dir:       cwd,
		textInput: ti,
		spinner:   s,
	}
}

func (m BranchModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadBranchesCmd(),
	)
}

type branchesLoadedMsg struct {
	branches []string
	current  string
}
type branchActionSuccessMsg string
type branchErrorMsg struct{ err error }

func (m BranchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.step == branchStepCreateInput {
				m.step = branchStepBrowsing
				m.err = nil
			} else {
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "up", "k":
			if m.step == branchStepBrowsing && len(m.branches) > 0 {
				m.selectedIndex = (m.selectedIndex - 1 + len(m.branches)) % len(m.branches)
			}

		case "down", "j":
			if m.step == branchStepBrowsing && len(m.branches) > 0 {
				m.selectedIndex = (m.selectedIndex + 1) % len(m.branches)
			}

		case "n":
			if m.step == branchStepBrowsing {
				m.step = branchStepCreateInput
				m.textInput.SetValue("")
				m.textInput.Focus()
			}

		case "enter":
			switch m.step {
			case branchStepBrowsing:
				if len(m.branches) > 0 {
					target := m.branches[m.selectedIndex]
					m.step = branchStepRunning
					return m, m.checkoutBranchCmd(target)
				}
			case branchStepCreateInput:
				newName := strings.TrimSpace(m.textInput.Value())
				if newName == "" {
					return m, nil
				}
				m.step = branchStepRunning
				return m, m.createNewBranchCmd(newName)
			case branchStepSuccess, branchStepError:
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case branchesLoadedMsg:
		m.branches = msg.branches
		m.currentBranch = msg.current
		m.step = branchStepBrowsing

		// Pre-select current branch in index
		for i, b := range m.branches {
			if b == m.currentBranch {
				m.selectedIndex = i
				break
			}
		}

	case branchActionSuccessMsg:
		m.step = branchStepSuccess
		m.successMessage = string(msg)

	case branchErrorMsg:
		m.step = branchStepError
		m.err = msg.err
	}

	if m.step == branchStepCreateInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m BranchModel) View() string {
	var s strings.Builder

	s.WriteString(RenderHeader("Branch Manager"))

	switch m.step {
	case branchStepCheckRepo:
		s.WriteString(fmt.Sprintf("\n%s Reading git repository metrics...\n\n", m.spinner.View()))

	case branchStepBrowsing:
		s.WriteString(StyleSubtitle.Render(fmt.Sprintf("🌿 Local Branches (Current: %s)", StyleHighlight.Render(m.currentBranch))) + "\n\n")

		if len(m.branches) == 0 {
			s.WriteString(StyleTextMuted.Render("  No branches detected.") + "\n\n")
		} else {
			for i, branch := range m.branches {
				bullet := "  "
				var label string

				if i == m.selectedIndex {
					bullet = " ❯ "
					if branch == m.currentBranch {
						label = StyleActiveItem.Render(fmt.Sprintf("🌿 %s (active)", branch))
					} else {
						label = StyleActiveItem.Render(branch)
					}
				} else {
					if branch == m.currentBranch {
						label = StyleTextSuccess.Render(fmt.Sprintf("🌿 %s (active)", branch))
					} else {
						label = StyleInactiveItem.Render(branch)
					}
				}
				s.WriteString(bullet + label + "\n")
			}
		}

		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Checkout • [n] New Branch • [Esc] Back to Dashboard") + "\n")

	case branchStepCreateInput:
		s.WriteString(StyleSubtitle.Render("✨ Create New Branch") + "\n\n")
		s.WriteString(fmt.Sprintf("Creating a new branch branching off from: %s\n\n", StyleHighlight.Render(m.currentBranch)))
		s.WriteString("Enter new branch name:\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Create & Checkout • [Esc] Cancel") + "\n")

	case branchStepRunning:
		s.WriteString(fmt.Sprintf("\n%s Performing Git branch operation...\n\n", m.spinner.View()))

	case branchStepSuccess:
		s.WriteString(StyleSuccessCard.Render(fmt.Sprintf("🎉 SUCCESS!\n\n%s", m.successMessage)) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")

	case branchStepError:
		errStr := fmt.Sprintf("❌ BRANCH ERROR\n\nFailed to perform branch operation: %s", m.err.Error())
		s.WriteString(StyleErrorCard.Render(errStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")
	}

	return s.String()
}

func (m BranchModel) loadBranchesCmd() tea.Cmd {
	return func() tea.Msg {
		// Wait a bit for UI transition feel
		time.Sleep(300 * time.Millisecond)

		if !git.IsGitRepository(m.dir) {
			return branchErrorMsg{err: fmt.Errorf("active directory is not a Git repository.\nPlease initialize Git or navigate to a project folder")}
		}

		branches, current, err := git.LocalBranches(m.dir)
		if err != nil {
			return branchErrorMsg{err: err}
		}

		return branchesLoadedMsg{
			branches: branches,
			current:  current,
		}
	}
}

func (m BranchModel) checkoutBranchCmd(target string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		err := git.Checkout(m.dir, target)
		if err != nil {
			return branchErrorMsg{err: err}
		}

		return branchActionSuccessMsg(fmt.Sprintf("Checked out to branch %s successfully!", StyleHighlight.Render(target)))
	}
}

func (m BranchModel) createNewBranchCmd(name string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		err := git.CreateBranch(m.dir, name)
		if err != nil {
			return branchErrorMsg{err: err}
		}

		return branchActionSuccessMsg(fmt.Sprintf("Successfully spawned and checked out branch %s!", StyleHighlight.Render(name)))
	}
}
