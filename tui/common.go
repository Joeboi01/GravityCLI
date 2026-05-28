package tui

import (
	"context"
	"net/http"
	"time"

	"GravityCLI/internal/browser"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// GetGitHubClient initializes and returns an authenticated GitHub client.
func GetGitHubClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	tc.Timeout = httpClient.Timeout
	return github.NewClient(tc)
}

// OpenBrowser opens the specified URL in the user's default browser.
func OpenBrowser(url string) error {
	return browser.Open(url)
}
