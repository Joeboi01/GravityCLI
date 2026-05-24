package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"GravityCLI/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type navStep int

const (
	navStepNormal navStep = iota
	navStepCommitInput
	navStepRunning
	navStepAutoCommitInput
	navStepSuccess
	navStepError
)

type fileItem struct {
	name  string
	isDir bool
}

type statusFileItem struct {
	path   string
	staged bool
}

type NavModel struct {
	step       navStep
	currentDir string
	files      []fileItem
	browserIdx int

	// Git specific state
	isGitRepo       bool
	gitBranch       string
	modifiedFiles   []statusFileItem
	modifiedFileIdx int
	inGitPane       bool // false = focus left browser, true = focus right git files

	// Inputs
	textInput textinput.Model
	spinner   spinner.Model
	err       error

	// Results
	toastMessage string
}

func NewNavModel() NavModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	ti := textinput.New()
	ti.Placeholder = "feat: add landing page layout"
	ti.CharLimit = 150
	ti.Width = 40

	cwd, _ := os.Getwd()

	m := NavModel{
		step:       navStepNormal,
		currentDir: cwd,
		textInput:  ti,
		spinner:    s,
		inGitPane:  false,
	}
	m.readDir()
	m.scanGit()

	return m
}

func (m NavModel) Init() tea.Cmd {
	return m.spinner.Tick
}

type navActionSuccessMsg string
type navErrorMsg struct{ err error }

func (m NavModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.step == navStepCommitInput || m.step == navStepAutoCommitInput {
				m.step = navStepNormal
				m.err = nil
			} else {
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "tab":
			if m.step == navStepNormal && m.isGitRepo {
				m.inGitPane = !m.inGitPane
			}

		case "up", "k":
			if m.step == navStepNormal {
				if m.inGitPane {
					if len(m.modifiedFiles) > 0 {
						m.modifiedFileIdx = (m.modifiedFileIdx - 1 + len(m.modifiedFiles)) % len(m.modifiedFiles)
					}
				} else {
					if len(m.files) > 0 {
						m.browserIdx = (m.browserIdx - 1 + len(m.files)) % len(m.files)
					}
				}
			}

		case "down", "j":
			if m.step == navStepNormal {
				if m.inGitPane {
					if len(m.modifiedFiles) > 0 {
						m.modifiedFileIdx = (m.modifiedFileIdx + 1) % len(m.modifiedFiles)
					}
				} else {
					if len(m.files) > 0 {
						m.browserIdx = (m.browserIdx + 1) % len(m.files)
					}
				}
			}

		case "space":
			// Staging a file using spacebar in the git files list
			if m.step == navStepNormal && m.inGitPane && len(m.modifiedFiles) > 0 {
				idx := m.modifiedFileIdx
				m.modifiedFiles[idx].staged = !m.modifiedFiles[idx].staged
			}

		case "i":
			// Trigger Git Init
			if m.step == navStepNormal && !m.isGitRepo {
				m.step = navStepRunning
				return m, m.initGitCmd()
			}

		case "c":
			// Trigger Commit message input
			if m.step == navStepNormal && m.isGitRepo {
				m.step = navStepCommitInput
				m.textInput.SetValue("")
				m.textInput.Focus()
			}

		case "A":
			// Trigger Auto Commit & Push
			if m.step == navStepNormal && m.isGitRepo {
				m.step = navStepAutoCommitInput
				m.textInput.SetValue("")
				m.textInput.Focus()
			}

		case "p":
			// Trigger Push
			if m.step == navStepNormal && m.isGitRepo {
				m.step = navStepRunning
				return m, m.pushGitCmd()
			}

		case "enter":
			switch m.step {
			case navStepNormal:
				if !m.inGitPane && len(m.files) > 0 {
					selected := m.files[m.browserIdx]
					if selected.name == ".." {
						parent := filepath.Dir(m.currentDir)
						switch {
						case parent == m.currentDir && runtime.GOOS == "windows":
							// At drive root on Windows — show drive selector
							m.currentDir = ""
						case parent == m.currentDir:
							// At filesystem root on Linux/macOS — stay put
						default:
							m.currentDir = parent
						}
						m.readDir()
						m.scanGit()
					} else if selected.isDir {
						if m.currentDir == "" {
							// Entering a drive letter on Windows (e.g. "C:\")
							m.currentDir = selected.name
						} else {
							m.currentDir = filepath.Join(m.currentDir, selected.name)
						}
						m.readDir()
						m.scanGit()
					}
				}

			case navStepCommitInput:
				msgText := strings.TrimSpace(m.textInput.Value())
				if msgText == "" {
					return m, nil
				}
				m.step = navStepRunning
				return m, m.commitGitCmd(msgText)

			case navStepAutoCommitInput:
				msgText := strings.TrimSpace(m.textInput.Value())
				if msgText == "" {
					return m, nil
				}
				m.step = navStepRunning
				return m, m.autoCommitPushGitCmd(msgText)

			case navStepSuccess, navStepError:
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case navActionSuccessMsg:
		m.step = navStepSuccess
		m.toastMessage = string(msg)

	case navErrorMsg:
		m.step = navStepError
		m.err = msg.err
	}

	if m.step == navStepCommitInput || m.step == navStepAutoCommitInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m NavModel) View() string {
	var s strings.Builder

	s.WriteString(RenderHeader("Directory Cockpit"))

	switch m.step {
	case navStepRunning:
		s.WriteString(fmt.Sprintf("\n%s Executing Git command in current directory...\n\n", m.spinner.View()))

	case navStepSuccess:
		s.WriteString(StyleSuccessCard.Render(fmt.Sprintf("🎉 SUCCESS!\n\n%s", m.toastMessage)) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")

	case navStepError:
		s.WriteString(StyleErrorCard.Render(fmt.Sprintf("❌ GIT ERROR\n\n%s", m.err.Error())) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")

	case navStepCommitInput:
		s.WriteString(StyleSubtitle.Render("✍️ Create Commit") + "\n\n")
		s.WriteString("Enter commit message:\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Commit Staged Files • [Esc] Cancel") + "\n")

	case navStepAutoCommitInput:
		s.WriteString(StyleSubtitle.Render("🚀 Auto-Push All Changes") + "\n\n")
		s.WriteString("Enter commit message (will auto-stage, commit, & push):\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Auto-Push • [Esc] Cancel") + "\n")

	case navStepNormal:
		// Left Pane: File tree browser
		var leftPane strings.Builder
		leftPane.WriteString(StylePanelTitle.Render("📁 DIRECTORY BROWSER") + "\n")
		dirLabel := m.currentDir
		if dirLabel == "" {
			dirLabel = "💻 Select a Drive"
		}
		leftPane.WriteString(StyleTextMuted.Render(dirLabel) + "\n\n")

		focusLeft := !m.inGitPane
		for i, f := range m.files {
			bullet := "  "
			var nameStr string

			if focusLeft && i == m.browserIdx {
				bullet = " ❯ "
				if f.isDir {
					nameStr = StyleActiveItem.Render("📁 " + f.name)
				} else {
					nameStr = StyleActiveItem.Render("📄 " + f.name)
				}
			} else {
				if f.isDir {
					nameStr = StyleInactiveItem.Render("📁 " + f.name)
				} else {
					nameStr = StyleTextMuted.Render("📄 " + f.name)
				}
			}
			leftPane.WriteString(bullet + nameStr + "\n")
		}

		// Right Pane: Git Cockpit
		var rightPane strings.Builder
		rightPane.WriteString(StylePanelTitle.Render("🌿 GIT CONTROL COCKPIT") + "\n")

		if !m.isGitRepo {
			rightPane.WriteString(StyleTextWarning.Render("⚠️  Git Not Initialized") + "\n\n")
			rightPane.WriteString("This folder is not a Git repository.\n\n")
			rightPane.WriteString(StyleHighlight.Render("💡 Press [i] to run 'git init'") + "\n")
			rightPane.WriteString("to start tracking version history here.\n")
		} else {
			rightPane.WriteString(fmt.Sprintf("Branch: %s\n\n", StyleHighlight.Render(m.gitBranch)))
			rightPane.WriteString(StyleSubtitle.Render("Staging Files:") + "\n")

			if len(m.modifiedFiles) == 0 {
				rightPane.WriteString(StyleTextSuccess.Render("  ✨ Clean working tree.") + "\n\n")
			} else {
				focusRight := m.inGitPane
				for i, mf := range m.modifiedFiles {
					bullet := "  "
					checkbox := "[ ]"
					if mf.staged {
						checkbox = "[✅]"
					}

					var fileLine string
					if focusRight && i == m.modifiedFileIdx {
						bullet = " ❯ "
						fileLine = StyleActiveItem.Render(fmt.Sprintf("%s %s", checkbox, mf.path))
					} else {
						fileLine = StyleInactiveItem.Render(fmt.Sprintf("%s %s", checkbox, mf.path))
					}
					rightPane.WriteString(bullet + fileLine + "\n")
				}
				rightPane.WriteString("\n")
			}

			// Git Action buttons list
			rightPane.WriteString(StyleTextMuted.Render("Quick Actions:") + "\n")
			rightPane.WriteString(fmt.Sprintf("  %s Auto Add & Push All\n", StyleHighlight.Render("[Shift+A]")))
			rightPane.WriteString(fmt.Sprintf("  %s Commit staged files\n", StyleHighlight.Render("[c]")))
			rightPane.WriteString(fmt.Sprintf("  %s Push commits to GitHub\n\n", StyleHighlight.Render("[p]")))
			rightPane.WriteString(StyleTextMuted.Render("[Space] Toggle staging file • [Tab] Switch pane") + "\n")
		}

		// Render left and right panels side-by-side using Lip Gloss layout
		sideBySide := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(45).Render(leftPane.String()),
			lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDarkSlate).
				Padding(1, 2).
				Width(40).
				Render(rightPane.String()),
		)
		s.WriteString(sideBySide + "\n")
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Open Folder • [Esc] Back to Dashboard") + "\n")
	}

	return s.String()
}

func (m *NavModel) readDir() {
	var items []fileItem

	// Windows Drive Root Selection
	if m.currentDir == "" && runtime.GOOS == "windows" {
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			d := string(drive) + ":\\"
			if _, err := os.Stat(d); err == nil {
				items = append(items, fileItem{name: d, isDir: true})
			}
		}
		m.files = items
		m.browserIdx = 0
		return
	}

	// Add parent option
	items = append(items, fileItem{name: "..", isDir: true})

	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		m.files = items
		return
	}

	var dirs []fileItem
	var files []fileItem

	for _, entry := range entries {
		name := entry.Name()
		// Hide hidden items
		if strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			dirs = append(dirs, fileItem{name: name, isDir: true})
		} else {
			files = append(files, fileItem{name: name, isDir: false})
		}
	}

	items = append(items, dirs...)
	items = append(items, files...)
	m.files = items
	m.browserIdx = 0
}

func (m *NavModel) scanGit() {
	m.isGitRepo = git.IsGitRepository(m.currentDir)
	if !m.isGitRepo {
		m.gitBranch = ""
		m.modifiedFiles = nil
		m.inGitPane = false
		return
	}

	_, current, err := git.LocalBranches(m.currentDir)
	if err == nil {
		m.gitBranch = current
	} else {
		m.gitBranch = "unknown"
	}

	files, err := git.GetStatusFiles(m.currentDir)
	if err == nil {
		// Cache staged/unstaged state
		// In a simple app, we start all modified files as unstaged
		// But if they were already staged, we'll let user toggle.
		// For simplicity, we initialize new items as unstaged, keeping existing toggles intact
		var newList []statusFileItem
		for _, f := range files {
			staged := false
			// Check if we already have it staged in our cached list
			for _, old := range m.modifiedFiles {
				if old.path == f {
					staged = old.staged
					break
				}
			}
			newList = append(newList, statusFileItem{path: f, staged: staged})
		}
		m.modifiedFiles = newList
	} else {
		m.modifiedFiles = nil
	}

	if m.modifiedFileIdx >= len(m.modifiedFiles) {
		m.modifiedFileIdx = 0
	}
}

func (m NavModel) initGitCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		err := git.Init(m.currentDir)
		if err != nil {
			return navErrorMsg{err: err}
		}

		return navActionSuccessMsg("Git repository successfully initialized in this folder!")
	}
}

func (m NavModel) commitGitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		// 1. Stage all marked files
		var stagedFiles []string
		for _, f := range m.modifiedFiles {
			if f.staged {
				stagedFiles = append(stagedFiles, f.path)
			}
		}

		if len(stagedFiles) == 0 {
			return navErrorMsg{err: fmt.Errorf("no files staged for commit. Press [Space] on files to stage them first")}
		}

		err := git.Add(m.currentDir, stagedFiles)
		if err != nil {
			return navErrorMsg{err: err}
		}

		// 2. Commit
		err = git.Commit(m.currentDir, message)
		if err != nil {
			return navErrorMsg{err: err}
		}

		return navActionSuccessMsg(fmt.Sprintf("Created commit '%s' successfully!", StyleHighlight.Render(message)))
	}
}

func (m NavModel) pushGitCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		err := git.Push(m.currentDir)
		if err != nil {
			return navErrorMsg{err: err}
		}

		return navActionSuccessMsg("Commits successfully pushed to remote GitHub repository!")
	}
}

func (m NavModel) autoCommitPushGitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		// 1. Stage ALL changes
		var allModified []string
		for _, f := range m.modifiedFiles {
			allModified = append(allModified, f.path)
		}

		if len(allModified) == 0 {
			return navErrorMsg{err: fmt.Errorf("no modified files to commit")}
		}

		err := git.Add(m.currentDir, allModified)
		if err != nil {
			return navErrorMsg{err: err}
		}

		// 2. Commit
		err = git.Commit(m.currentDir, message)
		if err != nil {
			return navErrorMsg{err: err}
		}

		// 3. Push
		err = git.Push(m.currentDir)
		if err != nil {
			return navErrorMsg{err: err}
		}

		return navActionSuccessMsg(fmt.Sprintf("Auto-pushed '%s' successfully to GitHub!", StyleHighlight.Render(message)))
	}
}
