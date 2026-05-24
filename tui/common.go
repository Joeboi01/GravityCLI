package tui

import (
	"context"

	"GravityCLI/internal/browser"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// GetGitHubClient initializes and returns an authenticated GitHub client.
func GetGitHubClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// OpenBrowser opens the specified URL in the user's default browser.
func OpenBrowser(url string) error {
	return browser.Open(url)
}

