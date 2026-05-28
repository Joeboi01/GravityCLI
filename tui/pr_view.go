package tui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"GravityCLI/config"
	"GravityCLI/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v60/github"
)

type prStep int

const (
	prStepLoading prStep = iota
	prStepHubMenu
	prStepList
	prStepDetails
	prStepCheckingOut
	prStepSelectBase
	prStepTitle
	prStepBody
	prStepCreating
	prStepSuccess
	prStepError
)

type prListItem struct {
	number    int
	title     string
	author    string
	state     string
	createdAt time.Time
	headRef   string
	baseRef   string
	body      string
	htmlURL   string
}

type PRModel struct {
	step    prStep
	dir     string
	spinner spinner.Model
	err     error

	// Git & GitHub Context
	repoOwner     string
	repoName      string
	currentBranch string

	// Base branch selection
	remoteBranches []string
	baseBranchIdx  int

	// User Inputs
	titleInput textinput.Model
	bodyInput  textarea.Model

	// Results
	createdURL string
	prNumber   int

	// PR Hub additions
	hubIndex     int // 0 = View PRs, 1 = Create PR
	prList       []prListItem
	prIndex      int
	selectedPR   prListItem
	toastMessage string
}

func NewPRModel() PRModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	ta := textarea.New()
	ta.Placeholder = "Write pull request description (markdown supported)..."
	ta.CharLimit = 1000
	ta.SetWidth(60)
	ta.SetHeight(10)

	cwd, _ := os.Getwd()

	return PRModel{
		step:       prStepLoading,
		dir:        cwd,
		spinner:    s,
		titleInput: ti,
		bodyInput:  ta,
		hubIndex:   0,
	}
}

func (m PRModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadPRContextCmd(),
	)
}

type prContextMsg struct {
	owner         string
	repo          string
	currentBranch string
	branches      []string
	lastCommit    string
}

type prCreateSuccessMsg struct {
	url    string
	number int
}
type prErrorMsg struct{ err error }
type prListLoadedMsg []prListItem
type prCheckoutSuccessMsg string

func (m PRModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			switch m.step {
			case prStepHubMenu:
				return m, func() tea.Msg { return BackMsg{} }
			case prStepList:
				m.step = prStepHubMenu
				m.toastMessage = ""
			case prStepDetails:
				m.step = prStepList
			case prStepSelectBase:
				m.step = prStepHubMenu
			case prStepTitle:
				m.step = prStepSelectBase
			case prStepBody:
				m.step = prStepTitle
				m.titleInput.Focus()
			default:
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "up", "k":
			switch m.step {
			case prStepHubMenu:
				m.hubIndex = (m.hubIndex - 1 + 2) % 2
			case prStepList:
				if len(m.prList) > 0 {
					m.prIndex = (m.prIndex - 1 + len(m.prList)) % len(m.prList)
				}
			case prStepSelectBase:
				if len(m.remoteBranches) > 0 {
					m.baseBranchIdx = (m.baseBranchIdx - 1 + len(m.remoteBranches)) % len(m.remoteBranches)
				}
			}

		case "down", "j":
			switch m.step {
			case prStepHubMenu:
				m.hubIndex = (m.hubIndex + 1) % 2
			case prStepList:
				if len(m.prList) > 0 {
					m.prIndex = (m.prIndex + 1) % len(m.prList)
				}
			case prStepSelectBase:
				if len(m.remoteBranches) > 0 {
					m.baseBranchIdx = (m.baseBranchIdx + 1) % len(m.remoteBranches)
				}
			}

		case "c":
			// Checkout branch in list or details view
			if m.step == prStepList && len(m.prList) > 0 {
				m.selectedPR = m.prList[m.prIndex]
				m.step = prStepCheckingOut
				return m, m.checkoutPRCmd(m.selectedPR.number, m.selectedPR.headRef)
			} else if m.step == prStepDetails {
				m.step = prStepCheckingOut
				return m, m.checkoutPRCmd(m.selectedPR.number, m.selectedPR.headRef)
			}

		case "o":
			// Open HTML URL in browser
			if m.step == prStepList && len(m.prList) > 0 {
				_ = OpenBrowser(m.prList[m.prIndex].htmlURL)
			} else if m.step == prStepDetails {
				_ = OpenBrowser(m.selectedPR.htmlURL)
			}

		case "r":
			// Refresh PR list
			if m.step == prStepList {
				m.step = prStepLoading
				return m, m.loadPRListCmd()
			}

		case "enter":
			switch m.step {
			case prStepHubMenu:
				if m.hubIndex == 0 {
					m.step = prStepLoading
					return m, m.loadPRListCmd()
				} else {
					m.step = prStepSelectBase
				}
			case prStepList:
				if len(m.prList) > 0 {
					m.selectedPR = m.prList[m.prIndex]
					m.step = prStepDetails
				}
			case prStepSelectBase:
				if len(m.remoteBranches) > 0 {
					m.step = prStepTitle
					m.titleInput.Focus()
				}
			case prStepTitle:
				if strings.TrimSpace(m.titleInput.Value()) != "" {
					m.step = prStepBody
					m.bodyInput.Focus()
				}
			case prStepBody:
				m.step = prStepCreating
				return m, m.createPRCmd()
			case prStepSuccess, prStepError:
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case prContextMsg:
		m.repoOwner = msg.owner
		m.repoName = msg.repo
		m.currentBranch = msg.currentBranch
		m.remoteBranches = msg.branches
		m.step = prStepHubMenu

		// Pre-populate title with last commit message
		m.titleInput.SetValue(msg.lastCommit)

		// Pre-select main/master if available
		m.baseBranchIdx = 0
		for i, br := range m.remoteBranches {
			if br == "main" || br == "master" {
				m.baseBranchIdx = i
				break
			}
		}

	case prListLoadedMsg:
		m.prList = msg
		m.step = prStepList
		m.prIndex = 0

	case prCheckoutSuccessMsg:
		m.step = prStepSuccess
		m.toastMessage = string(msg)

	case prCreateSuccessMsg:
		m.step = prStepSuccess
		m.createdURL = msg.url
		m.prNumber = msg.number
		m.toastMessage = ""
		// Open in browser
		_ = OpenBrowser(m.createdURL)

	case prErrorMsg:
		m.step = prStepError
		m.err = msg.err
	}

	if m.step == prStepTitle {
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.step == prStepBody {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m PRModel) View() string {
	var s strings.Builder

	headerTitle := "Pull Request Hub"
	if m.step == prStepSelectBase || m.step == prStepTitle || m.step == prStepBody || m.step == prStepCreating {
		headerTitle = "Create Pull Request"
	}
	s.WriteString(RenderHeader(headerTitle))

	switch m.step {
	case prStepLoading:
		s.WriteString(fmt.Sprintf("\n%s Loading...\n\n", m.spinner.View()))

	case prStepHubMenu:
		s.WriteString(fmt.Sprintf("Repository: %s\n", StyleHighlight.Render(m.repoOwner+"/"+m.repoName)))
		s.WriteString(fmt.Sprintf("Head Branch: %s\n\n", StyleTextSuccess.Render(m.currentBranch)))
		s.WriteString(StyleSubtitle.Render("⚡ Pull Request Options:") + "\n\n")

		opts := []string{
			"🌐 View Active Pull Requests",
			"🆕 Create New Pull Request",
		}

		for i, opt := range opts {
			if i == m.hubIndex {
				s.WriteString(StyleActiveItem.Render(fmt.Sprintf("  ❯ %s", opt)) + "\n")
			} else {
				s.WriteString(StyleInactiveItem.Render(fmt.Sprintf("    %s", opt)) + "\n")
			}
		}
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Select • [Esc] Back to Dashboard") + "\n")

	case prStepList:
		s.WriteString(fmt.Sprintf("Repository: %s\n", StyleHighlight.Render(m.repoOwner+"/"+m.repoName)))
		s.WriteString(StyleSubtitle.Render("🌐 Active Pull Requests:") + "\n\n")

		if len(m.prList) == 0 {
			s.WriteString(StyleTextMuted.Render("  No active pull requests found.") + "\n\n")
		} else {
			var listPane strings.Builder

			// Display 6 items at a time
			start := m.prIndex - 2
			if start < 0 {
				start = 0
			}
			end := start + 6
			if end > len(m.prList) {
				end = len(m.prList)
				start = end - 6
				if start < 0 {
					start = 0
				}
			}

			for i := start; i < end; i++ {
				pr := m.prList[i]
				bullet := "  "
				var titleStr string

				if i == m.prIndex {
					bullet = " ❯ "
					titleStr = StyleActiveItem.Render(fmt.Sprintf("#%d %s", pr.number, pr.title))
				} else {
					titleStr = StyleInactiveItem.Render(fmt.Sprintf("#%d %s", pr.number, pr.title))
				}

				listPane.WriteString(bullet + titleStr + "\n")
				listPane.WriteString(fmt.Sprintf("    by @%s • %s ➔ %s\n\n",
					pr.author, StyleTextMuted.Render(pr.headRef), StyleTextSuccess.Render(pr.baseRef)))
			}

			// Right side details preview
			var previewPane strings.Builder
			if m.prIndex < len(m.prList) {
				activePR := m.prList[m.prIndex]
				previewPane.WriteString(StylePanelTitle.Render("⚡ PR PREVIEW") + "\n\n")
				previewPane.WriteString(fmt.Sprintf("Title:    %s\n", activePR.title))
				previewPane.WriteString(fmt.Sprintf("Author:   @%s\n\n", activePR.author))
				previewPane.WriteString(fmt.Sprintf("Branches: %s ➔ %s\n\n", activePR.headRef, activePR.baseRef))

				desc := activePR.body
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
				if desc == "" {
					desc = "No description provided."
				}
				previewPane.WriteString(StyleTextMuted.Render("Description:") + "\n")
				previewPane.WriteString(desc + "\n")
			}

			sideBySide := lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(50).Render(listPane.String()),
				lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ColorDarkSlate).
					Padding(1, 2).
					Width(35).
					Render(previewPane.String()),
			)
			s.WriteString(sideBySide + "\n")
		}

		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Full Details • [c] Checkout Local • [o] Open Browser • [r] Refresh • [Esc] Back") + "\n")

	case prStepDetails:
		pr := m.selectedPR
		s.WriteString(StyleSubtitle.Render(fmt.Sprintf("⚡ Pull Request #%d Details", pr.number)) + "\n\n")

		var infoPane strings.Builder
		infoPane.WriteString(fmt.Sprintf("Title:      %s\n", StyleHighlight.Render(pr.title)))
		infoPane.WriteString(fmt.Sprintf("Author:     @%s\n", pr.author))
		infoPane.WriteString(fmt.Sprintf("From:       %s\n", StyleTextWarning.Render(pr.headRef)))
		infoPane.WriteString(fmt.Sprintf("To (Base):  %s\n", StyleTextSuccess.Render(pr.baseRef)))
		infoPane.WriteString(fmt.Sprintf("Created:    %s\n\n", pr.createdAt.Format("2006-01-02 15:04:05")))

		infoPane.WriteString(StyleTextMuted.Render("Description:") + "\n")
		bodyText := pr.body
		if bodyText == "" {
			bodyText = "No description provided."
		}
		bodyLines := strings.Split(bodyText, "\n")
		maxLines := 10
		for i, line := range bodyLines {
			if i >= maxLines {
				infoPane.WriteString("...\n")
				break
			}
			infoPane.WriteString("  " + line + "\n")
		}

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDarkSlate).
			Padding(1, 2).
			Width(75).
			Render(infoPane.String())

		s.WriteString(box + "\n\n")
		s.WriteString(StyleTextMuted.Render("[c] Fetch & Checkout Locally • [o] Open in Browser • [Esc] Back to List") + "\n")

	case prStepCheckingOut:
		s.WriteString(fmt.Sprintf("\n%s Fetching pull request ref and checking out local branch...\n\n", m.spinner.View()))

	case prStepSelectBase:
		s.WriteString(fmt.Sprintf("Repository: %s\n", StyleHighlight.Render(m.repoOwner+"/"+m.repoName)))
		s.WriteString(fmt.Sprintf("Head Branch: %s\n\n", StyleTextSuccess.Render(m.currentBranch)))
		s.WriteString(StyleSubtitle.Render("🎯 Select TARGET base branch:") + "\n\n")

		if len(m.remoteBranches) == 0 {
			s.WriteString(StyleTextMuted.Render("  No remote branches found.") + "\n\n")
		} else {
			for i, b := range m.remoteBranches {
				bullet := "  "
				var label string

				if i == m.baseBranchIdx {
					bullet = " ❯ "
					label = StyleActiveItem.Render(b)
				} else {
					label = StyleInactiveItem.Render(b)
				}
				s.WriteString(bullet + label + "\n")
			}
		}
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Choose Base Branch • [Esc] Exit") + "\n")

	case prStepTitle:
		s.WriteString(StyleSubtitle.Render("✍️ Enter Pull Request Title") + "\n\n")
		s.WriteString(fmt.Sprintf("Title (suggested default: %s):\n", StyleTextMuted.Render("last commit message")))
		s.WriteString(m.titleInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Confirm & Next • [Esc] Back") + "\n")

	case prStepBody:
		s.WriteString(StyleSubtitle.Render("📝 Enter Pull Request Body") + "\n\n")
		s.WriteString("Description:\n")
		s.WriteString(m.bodyInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Create PR • [Esc] Back") + "\n")

	case prStepCreating:
		s.WriteString(fmt.Sprintf("\n%s Creating Pull Request on GitHub...\n\n", m.spinner.View()))

	case prStepSuccess:
		var successStr string
		if m.toastMessage != "" {
			successStr = m.toastMessage
		} else {
			successStr = fmt.Sprintf("Pull Request #%d created successfully!\n\nURL:\n%s\n\nOpening pull request in your web browser...",
				m.prNumber,
				StyleTextSuccess.Render(m.createdURL),
			)
		}
		s.WriteString(StyleSuccessCard.Render("🎉 SUCCESS!\n\n"+successStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter/q] Back to main dashboard") + "\n")

	case prStepError:
		errStr := fmt.Sprintf("❌ PR HUB FAILURE\n\nError: %s", m.err.Error())
		s.WriteString(StyleErrorCard.Render(errStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")
	}

	return s.String()
}

func (m PRModel) loadPRContextCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)

		if !git.IsGitRepository(m.dir) {
			return prErrorMsg{err: fmt.Errorf("active folder is not a Git repository")}
		}

		remoteURL, err := git.GetRemoteURL(m.dir, "origin")
		if err != nil {
			return prErrorMsg{err: fmt.Errorf("could not fetch Git remote origin URL.\nPlease add a remote origin to push to GitHub")}
		}

		owner, repo, err := parseGitHubRemote(remoteURL)
		if err != nil {
			return prErrorMsg{err: err}
		}

		cfg, err := config.Load()
		if err != nil {
			return prErrorMsg{err: err}
		}

		if !cfg.IsAuthenticated() {
			return prErrorMsg{err: fmt.Errorf("you are not authenticated. Run 'gravity auth' first")}
		}

		_, currentBranch, err := git.LocalBranches(m.dir)
		if err != nil {
			return prErrorMsg{err: err}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		// Fetch branches from GitHub
		var branches []string
		branchOpts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 100}}
		for {
			ghBranches, resp, err := client.Repositories.ListBranches(ctx, owner, repo, branchOpts)
			if err != nil {
				return prErrorMsg{err: fmt.Errorf("failed to list GitHub repository branches: %w", err)}
			}
			for _, b := range ghBranches {
				branches = append(branches, b.GetName())
			}
			if resp == nil || resp.NextPage == 0 {
				break
			}
			branchOpts.Page = resp.NextPage
		}

		// Try to parse last commit message for smart default title
		lastCommit, _ := git.GetLastCommitMessage(m.dir)

		return prContextMsg{
			owner:         owner,
			repo:          repo,
			currentBranch: currentBranch,
			branches:      branches,
			lastCommit:    lastCommit,
		}
	}
}

func (m PRModel) createPRCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		cfg, _ := config.Load()
		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		title := strings.TrimSpace(m.titleInput.Value())
		body := strings.TrimSpace(m.bodyInput.Value())
		base := m.remoteBranches[m.baseBranchIdx]

		newPR := &github.NewPullRequest{
			Title: github.String(title),
			Body:  github.String(body),
			Head:  github.String(m.currentBranch),
			Base:  github.String(base),
		}

		pr, _, err := client.PullRequests.Create(ctx, m.repoOwner, m.repoName, newPR)
		if err != nil {
			return prErrorMsg{err: err}
		}

		return prCreateSuccessMsg{
			url:    pr.GetHTMLURL(),
			number: pr.GetNumber(),
		}
	}
}

// parseGitHubRemote converts various Git remote URL styles to Owner and Repo.
func parseGitHubRemote(remote string) (string, string, error) {
	// Support HTTPS: https://github.com/owner/repo.git or https://github.com/owner/repo
	// Support SSH: git@github.com:owner/repo.git or ssh://git@github.com/owner/repo.git
	remote = strings.TrimSpace(remote)

	// Regexp to match owner and repo from SSH or HTTPS URLs
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/(.+?)(?:\.git)?/?$`)
	matches := re.FindStringSubmatch(remote)
	if len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// Try standard url.Parse as fallback
	u, err := url.Parse(remote)
	if err == nil {
		host := strings.TrimPrefix(u.Host, "www.")
		if host != "github.com" {
			return "", "", fmt.Errorf("remote URL is not a GitHub repository: %s", remote)
		}
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
	}

	return "", "", fmt.Errorf("could not parse GitHub repository owner/name from remote URL: %s", remote)
}

func (m PRModel) loadPRListCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)

		cfg, err := config.Load()
		if err != nil {
			return prErrorMsg{err: err}
		}

		client := GetGitHubClient(cfg.GitHubToken)
		ctx := context.Background()

		opt := &github.PullRequestListOptions{
			State:       "open",
			ListOptions: github.ListOptions{PerPage: 50},
		}

		var items []prListItem
		for {
			prs, resp, err := client.PullRequests.List(ctx, m.repoOwner, m.repoName, opt)
			if err != nil {
				return prErrorMsg{err: fmt.Errorf("failed to fetch active pull requests: %w", err)}
			}
			for _, pr := range prs {
				body := ""
				if pr.Body != nil {
					body = *pr.Body
				}
				items = append(items, prListItem{
					number:    pr.GetNumber(),
					title:     pr.GetTitle(),
					author:    pr.GetUser().GetLogin(),
					state:     pr.GetState(),
					createdAt: pr.GetCreatedAt().Time,
					headRef:   pr.GetHead().GetRef(),
					baseRef:   pr.GetBase().GetRef(),
					body:      body,
					htmlURL:   pr.GetHTMLURL(),
				})
			}
			if resp == nil || resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		return prListLoadedMsg(items)
	}
}

func (m PRModel) checkoutPRCmd(number int, headBranch string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		err := git.CheckoutPR(m.dir, number, headBranch)
		if err != nil {
			return prErrorMsg{err: err}
		}

		return prCheckoutSuccessMsg(fmt.Sprintf("Successfully fetched and checked out local branch %s!", StyleHighlight.Render(fmt.Sprintf("pr/%d-%s", number, headBranch))))
	}
}
