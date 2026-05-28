package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runGit runs a git command in a specific directory and returns stdout and stderr.
func runGit(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// IsGitRepository checks if a path contains a Git repository.
func IsGitRepository(dir string) bool {
	_, _, err := runGit(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// Init initializes a new Git repository at the target directory.
func Init(dir string) error {
	_, stderr, err := runGit(dir, "init")
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// Clone clones a repository to the target path.
func Clone(url, targetPath string) error {
	// git clone doesn't run inside the target path, it runs in its parent
	_, stderr, err := runGit("", "clone", url, targetPath)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// LocalBranches lists all local branches and returns them along with the current branch name.
func LocalBranches(dir string) ([]string, string, error) {
	stdout, stderr, err := runGit(dir, "branch", "--format=%(refname:short) %(HEAD)")
	if err != nil {
		return nil, "", errors.New(stderr)
	}

	var branches []string
	var current string

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		branchName := parts[0]
		branches = append(branches, branchName)

		if len(parts) > 1 && parts[1] == "*" {
			current = branchName
		}
	}

	return branches, current, nil
}

// Checkout switches the active Git branch to the specified one.
func Checkout(dir, branch string) error {
	_, stderr, err := runGit(dir, "checkout", branch)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// CreateBranch creates a new branch and switches to it.
func CreateBranch(dir, name string) error {
	_, stderr, err := runGit(dir, "checkout", "-b", name)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// GetStatusFiles lists modified, deleted, and untracked files.
// Returns a slice of file paths that are not yet committed.
func GetStatusFiles(dir string) ([]string, error) {
	stdout, stderr, err := runGit(dir, "status", "--porcelain=v1")
	if err != nil {
		return nil, errors.New(stderr)
	}

	return parseStatusPorcelain(stdout), nil
}

func parseStatusPorcelain(stdout string) []string {
	var files []string
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		// Status output format: XY filepath. Renames are represented as
		// "old -> new"; staging the new path is the useful default.
		if len(line) > 3 {
			path := strings.TrimSpace(line[3:])
			if strings.Contains(path, " -> ") {
				parts := strings.Split(path, " -> ")
				path = parts[len(parts)-1]
			}
			files = append(files, strings.Trim(path, `"`))
		}
	}
	return files
}

// Add stages the selected files.
func Add(dir string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add"}, files...)
	_, stderr, err := runGit(dir, args...)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// Commit creates a new commit with the given message.
func Commit(dir, message string) error {
	_, stderr, err := runGit(dir, "commit", "-m", message)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// Push pushes the current branch to origin.
// If the branch doesn't have an upstream, sets it.
func Push(dir string) error {
	_, currentBranch, err := LocalBranches(dir)
	if err != nil {
		return err
	}

	// Try standard push first
	_, stderr, err := runGit(dir, "push")
	if err != nil {
		// If pushing failed because there's no upstream branch set, set it!
		if strings.Contains(stderr, "no upstream branch") || strings.Contains(stderr, "set-upstream") {
			_, stderr, err = runGit(dir, "push", "--set-upstream", "origin", currentBranch)
			if err != nil {
				return errors.New(stderr)
			}
			return nil
		}
		return errors.New(stderr)
	}
	return nil
}

// AddRemote adds a remote origin URL.
func AddRemote(dir, name, url string) error {
	_, stderr, err := runGit(dir, "remote", "add", name, url)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}

// GetRemoteURL returns the URL of the specified remote (usually "origin").
func GetRemoteURL(dir, remoteName string) (string, error) {
	stdout, stderr, err := runGit(dir, "remote", "get-url", remoteName)
	if err != nil {
		return "", errors.New(stderr)
	}
	return stdout, nil
}

// GetLastCommitMessage fetches the latest commit message for smart defaults.
func GetLastCommitMessage(dir string) (string, error) {
	stdout, stderr, err := runGit(dir, "log", "-1", "--pretty=%B")
	if err != nil {
		return "", errors.New(stderr)
	}
	return strings.TrimSpace(stdout), nil
}

// CheckoutPR fetches a pull request by number and checks it out to a local branch.
func CheckoutPR(dir string, number int, headBranch string) error {
	// We'll create a local branch named pr/<number>-<headBranch>
	localBranch := fmt.Sprintf("pr/%d-%s", number, headBranch)

	// Fetch and create/overwrite local branch: git fetch origin +pull/<number>/head:<localBranch>
	_, stderr, err := runGit(dir, "fetch", "origin", fmt.Sprintf("+pull/%d/head:%s", number, localBranch))
	if err != nil {
		return errors.New(stderr)
	}

	// Checkout the branch
	_, stderr, err = runGit(dir, "checkout", localBranch)
	if err != nil {
		return errors.New(stderr)
	}
	return nil
}
