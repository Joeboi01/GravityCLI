package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadMissingConfigReturnsBlankConfig(t *testing.T) {
	t.Setenv("AppData", filepath.Join(t.TempDir(), "AppData"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.IsAuthenticated() {
		t.Fatal("new config should not be authenticated")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("AppData", filepath.Join(t.TempDir(), "AppData"))

	want := &Config{
		GitHubToken:    "token",
		GitHubUsername: "octocat",
		ClientID:       "client-id",
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected config file: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %v, want 0600", info.Mode().Perm())
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.GitHubToken != want.GitHubToken || got.GitHubUsername != want.GitHubUsername || got.ClientID != want.ClientID {
		t.Fatalf("loaded config = %#v, want %#v", got, want)
	}
}
