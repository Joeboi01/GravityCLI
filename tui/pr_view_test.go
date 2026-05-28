package tui

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		owner  string
		repo   string
	}{
		{
			name:   "https",
			remote: "https://github.com/owner/repo.git",
			owner:  "owner",
			repo:   "repo",
		},
		{
			name:   "ssh",
			remote: "git@github.com:owner/repo.git",
			owner:  "owner",
			repo:   "repo",
		},
		{
			name:   "repo name with dots",
			remote: "https://github.com/owner/my.repo.git",
			owner:  "owner",
			repo:   "my.repo",
		},
		{
			name:   "ssh url",
			remote: "ssh://git@github.com/owner/my.repo",
			owner:  "owner",
			repo:   "my.repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRemote(tt.remote)
			if err != nil {
				t.Fatalf("parseGitHubRemote() error = %v", err)
			}
			if owner != tt.owner || repo != tt.repo {
				t.Fatalf("parseGitHubRemote() = %q/%q, want %q/%q", owner, repo, tt.owner, tt.repo)
			}
		})
	}
}

func TestParseGitHubRemoteRejectsNonGitHubRemote(t *testing.T) {
	if _, _, err := parseGitHubRemote("https://example.com/owner/repo.git"); err == nil {
		t.Fatal("expected non-GitHub remote to fail")
	}
}
