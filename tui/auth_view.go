package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"GravityCLI/config"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type authStep int

const (
	stepSelectMethod authStep = iota
	stepEnterClientID
	stepEnterPAT
	stepPollingDevice
	stepSuccess
	stepError
)

type AuthModel struct {
	step        authStep
	methodIndex int // 0 = OAuth Device Flow, 1 = Personal Access Token
	textInput   textinput.Model
	spinner     spinner.Model
	err         error

	// Device flow data
	clientID        string
	userCode        string
	verificationURI string
	deviceCode      string
	interval        int
	pollTicker      *time.Ticker
	expiresAt       time.Time

	// Success details
	username string
	token    string
}

func NewAuthModel() AuthModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	return AuthModel{
		step:        stepSelectMethod,
		textInput:   ti,
		spinner:     s,
		methodIndex: 0,
	}
}

func (m AuthModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

type tickMsg time.Time
type authSuccessMsg struct {
	token    string
	username string
}
type authErrorMsg struct{ err error }
type deviceCodeMsg struct {
	userCode        string
	deviceCode      string
	verificationURI string
	interval        int
	expiresIn       int
}

func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.step == stepEnterClientID || m.step == stepEnterPAT {
				// Allow typing 'q' into the input fields
				break
			}
			if m.step == stepPollingDevice {
				// Stop polling
				return m, func() tea.Msg { return BackMsg{} }
			}
			// Let root command handle quitting, or return dashboard
			return m, func() tea.Msg { return BackMsg{} }

		case "up", "k":
			if m.step == stepSelectMethod {
				m.methodIndex = 0
			}

		case "down", "j":
			if m.step == stepSelectMethod {
				m.methodIndex = 1
			}

		case "enter":
			switch m.step {
			case stepSelectMethod:
				if m.methodIndex == 0 {
					m.step = stepEnterClientID
					m.textInput.Placeholder = "Paste GitHub OAuth Client ID (or press Enter for setup guide)"
					m.textInput.SetValue("")
				} else {
					m.step = stepEnterPAT
					m.textInput.Placeholder = "Paste your GitHub Personal Access Token (PAT)"
					m.textInput.SetValue("")
				}
			case stepEnterClientID:
				m.clientID = strings.TrimSpace(m.textInput.Value())
				if m.clientID == "" {
					// Guided workflow for creating client ID
					m.step = stepError
					m.err = fmt.Errorf("client ID is required. Please create an OAuth application on GitHub settings")
					return m, nil
				}
				m.step = stepPollingDevice
				return m, m.requestDeviceCode()

			case stepEnterPAT:
				token := strings.TrimSpace(m.textInput.Value())
				if token == "" {
					m.err = fmt.Errorf("token cannot be empty")
					return m, nil
				}
				m.step = stepPollingDevice // Reuse polling screen for validation spinner
				return m, m.validateAndSavePAT(token)

			case stepSuccess, stepError:
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "esc":
			if m.step == stepEnterClientID || m.step == stepEnterPAT {
				m.step = stepSelectMethod
				m.err = nil
			} else {
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case deviceCodeMsg:
		m.userCode = msg.userCode
		m.deviceCode = msg.deviceCode
		m.verificationURI = msg.verificationURI
		m.interval = msg.interval
		m.expiresAt = time.Now().Add(time.Duration(msg.expiresIn) * time.Second)

		// Try to copy to clipboard
		_ = clipboard.WriteAll(m.userCode)

		// Open browser
		_ = OpenBrowser(m.verificationURI)

		// Start polling tick loop
		return m, m.pollForToken()

	case tickMsg:
		if m.step != stepPollingDevice {
			return m, nil
		}
		if time.Now().After(m.expiresAt) {
			m.step = stepError
			m.err = fmt.Errorf("device verification code expired. Please try again")
			return m, nil
		}
		return m, m.pollForToken()

	case authSuccessMsg:
		m.step = stepSuccess
		m.username = msg.username
		m.token = msg.token
		return m, nil

	case authErrorMsg:
		m.step = stepError
		m.err = msg.err
		return m, nil
	}

	if m.step == stepEnterClientID || m.step == stepEnterPAT {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AuthModel) View() string {
	var s strings.Builder

	s.WriteString(RenderHeader("Authentication Setup"))

	switch m.step {
	case stepSelectMethod:
		s.WriteString(StyleSubtitle.Render("Select authentication method:") + "\n\n")

		opts := []string{
			"🌐 OAuth Device Flow (Recommended - Browser Based)",
			"🔑 Personal Access Token (PAT - Manual Paste)",
		}

		for i, opt := range opts {
			if i == m.methodIndex {
				s.WriteString(StyleActiveItem.Render(fmt.Sprintf("  ❯ %s", opt)) + "\n")
			} else {
				s.WriteString(StyleInactiveItem.Render(fmt.Sprintf("    %s", opt)) + "\n")
			}
		}
		s.WriteString(StyleTextMuted.Render("\n[↑/↓] Navigate • [Enter] Select • [Esc] Back to Dashboard") + "\n")

	case stepEnterClientID:
		s.WriteString(StyleSubtitle.Render("GitHub OAuth App Client ID") + "\n\n")
		s.WriteString("To authenticate using OAuth, you need a free GitHub OAuth Client ID.\n")
		s.WriteString("1. Visit " + StyleHighlight.Render("https://github.com/settings/developers") + "\n")
		s.WriteString("2. Register a new OAuth Application (Application Name: GravityCLI)\n")
		s.WriteString("3. Set Homepage & Callback URL to " + StyleHighlight.Render("http://localhost") + "\n")
		s.WriteString("4. Copy the " + StyleHighlight.Render("Client ID") + " and paste it below.\n\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Start OAuth Flow • [Esc] Back • [q] Cancel") + "\n")

	case stepEnterPAT:
		s.WriteString(StyleSubtitle.Render("Enter Personal Access Token") + "\n\n")
		s.WriteString("Generate a classic token with " + StyleHighlight.Render("repo") + " and " + StyleHighlight.Render("read:user") + " scopes.\n")
		s.WriteString("Go to: " + StyleHighlight.Render("https://github.com/settings/tokens") + "\n\n")
		s.WriteString(m.textInput.View() + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Validate & Save • [Esc] Back • [q] Cancel") + "\n")

	case stepPollingDevice:
		if m.deviceCode != "" {
			s.WriteString(StyleSubtitle.Render("Authorize GravityCLI") + "\n\n")
			s.WriteString("We opened the authorization page in your browser. If not, open:\n")
			s.WriteString(StyleHighlight.Render("   "+m.verificationURI) + "\n\n")
			s.WriteString("And enter the following code (copied to your clipboard!):\n\n")

			codeBox := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(ColorSecondary).
				Padding(1, 4).
				Bold(true).
				Foreground(ColorWhite).
				Render(m.userCode)

			s.WriteString(codeBox + "\n\n")
			s.WriteString(fmt.Sprintf("%s Waiting for GitHub authorization... (Expires in %0.0fs)\n\n",
				m.spinner.View(), time.Until(m.expiresAt).Seconds()))
		} else {
			s.WriteString(fmt.Sprintf("\n%s Connecting to GitHub services...\n\n", m.spinner.View()))
		}
		s.WriteString(StyleTextMuted.Render("[q] Cancel Authentication") + "\n")

	case stepSuccess:
		successContent := fmt.Sprintf("🎉 SUCCESS!\n\nAuthenticated successfully as %s\nToken and credentials saved securely.",
			StyleHighlight.Render("@"+m.username))

		s.WriteString(StyleSuccessCard.Render(successContent) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter/q] Back to main dashboard") + "\n")

	case stepError:
		errStr := fmt.Sprintf("❌ AUTHENTICATION FAILURE\n\nFailed to authenticate: %s", m.err.Error())
		s.WriteString(StyleErrorCard.Render(errStr) + "\n\n")
		s.WriteString(StyleTextMuted.Render("[Enter] Back to Dashboard") + "\n")
	}

	return s.String()
}

// requestDeviceCode performs the device code request to GitHub.
func (m AuthModel) requestDeviceCode() tea.Cmd {
	return func() tea.Msg {
		form := url.Values{}
		form.Add("client_id", m.clientID)
		form.Add("scope", "repo,read:user")

		req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(form.Encode()))
		if err != nil {
			return authErrorMsg{err: err}
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := httpClient.Do(req)
		if err != nil {
			return authErrorMsg{err: err}
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return authErrorMsg{err: fmt.Errorf("GitHub server returned status %d: %s", resp.StatusCode, string(body))}
		}

		var res struct {
			DeviceCode      string `json:"device_code"`
			UserCode        string `json:"user_code"`
			VerificationURI string `json:"verification_uri"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
			Error           string `json:"error"`
		}

		if err := json.Unmarshal(body, &res); err != nil {
			return authErrorMsg{err: err}
		}

		if res.Error != "" {
			return authErrorMsg{err: fmt.Errorf("GitHub Error: %s", res.Error)}
		}

		return deviceCodeMsg{
			userCode:        res.UserCode,
			deviceCode:      res.DeviceCode,
			verificationURI: res.VerificationURI,
			interval:        res.Interval,
			expiresIn:       res.ExpiresIn,
		}
	}
}

// pollForToken polls GitHub for the access token during device flow.
func (m AuthModel) pollForToken() tea.Cmd {
	return tea.Tick(time.Duration(m.interval)*time.Second, func(t time.Time) tea.Msg {
		form := url.Values{}
		form.Add("client_id", m.clientID)
		form.Add("device_code", m.deviceCode)
		form.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
		if err != nil {
			return authErrorMsg{err: err}
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := httpClient.Do(req)
		if err != nil {
			return authErrorMsg{err: err}
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var res struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
		}

		if err := json.Unmarshal(body, &res); err != nil {
			return authErrorMsg{err: err}
		}

		if res.Error != "" {
			switch res.Error {
			case "authorization_pending":
				// Continue polling
				return tickMsg(t)
			case "slow_down":
				m.interval += 5
				return tickMsg(t)
			default:
				return authErrorMsg{err: fmt.Errorf("polling error: %s", res.Error)}
			}
		}

		// Success! Verify token and get username
		username, err := fetchGitHubUsername(res.AccessToken)
		if err != nil {
			return authErrorMsg{err: err}
		}

		// Save config
		cfg, _ := config.Load()
		cfg.GitHubToken = res.AccessToken
		cfg.GitHubUsername = username
		cfg.ClientID = m.clientID
		if err := config.Save(cfg); err != nil {
			return authErrorMsg{err: err}
		}

		return authSuccessMsg{token: res.AccessToken, username: username}
	})
}

// validateAndSavePAT validates the provided PAT and stores it.
func (m AuthModel) validateAndSavePAT(token string) tea.Cmd {
	return func() tea.Msg {
		// Small buffer wait for UX feel
		time.Sleep(800 * time.Millisecond)

		username, err := fetchGitHubUsername(token)
		if err != nil {
			return authErrorMsg{err: fmt.Errorf("invalid token or network error: %w", err)}
		}

		cfg, _ := config.Load()
		cfg.GitHubToken = token
		cfg.GitHubUsername = username
		cfg.ClientID = "" // No OAuth app client ID used
		if err := config.Save(cfg); err != nil {
			return authErrorMsg{err: err}
		}

		return authSuccessMsg{token: token, username: username}
	}
}

// fetchGitHubUsername calls GitHub user API to check identity.
func fetchGitHubUsername(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received bad status %d from API", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}

	return user.Login, nil
}
