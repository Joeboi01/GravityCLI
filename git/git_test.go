package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStatusPorcelain(t *testing.T) {
	input := " M modified.go\n?? new-file.txt\nR  old-name.go -> new-name.go\n"

	got := parseStatusPorcelain(input)
	want := []string{"modified.go", "new-file.txt", "new-name.go"}

	if len(got) != len(want) {
		t.Fatalf("got %d files, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("file %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestInitCreatesGitRepository(t *testing.T) {
	if _, _, err := runGit("", "version"); err != nil {
		t.Skip("git is not available")
	}

	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if !IsGitRepository(dir) {
		t.Fatalf("expected %s to be a git repository", dir)
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected .git directory: %v", err)
	}
}
