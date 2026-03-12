package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanki92/envsync/internal/git"
)

func TestInstallPostMergeHook(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	if err := git.InstallPostMergeHook(dir); err != nil {
		t.Fatalf("InstallPostMergeHook: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-merge")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(data), "envsync") {
		t.Fatal("hook should contain envsync")
	}

	if err := git.InstallPostMergeHook(dir); err != nil {
		t.Fatalf("second install: %v", err)
	}
	data2, _ := os.ReadFile(hookPath)
	if strings.Count(string(data2), "envsync") != strings.Count(string(data), "envsync") {
		t.Fatal("duplicate install should be idempotent")
	}
}

func TestHasPostMergeHook(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	if git.HasPostMergeHook(dir) {
		t.Fatal("should not have hook yet")
	}

	git.InstallPostMergeHook(dir)
	if !git.HasPostMergeHook(dir) {
		t.Fatal("should have hook after install")
	}
}

func TestUninstallPostMergeHook(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	git.InstallPostMergeHook(dir)
	if err := git.UninstallPostMergeHook(dir); err != nil {
		t.Fatalf("UninstallPostMergeHook: %v", err)
	}

	if git.HasPostMergeHook(dir) {
		t.Fatal("hook should be removed")
	}
}

func TestFindRepoRoot(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	subdir := filepath.Join(dir, "src", "pkg")
	os.MkdirAll(subdir, 0755)

	root, err := git.FindRepoRoot(subdir)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}

	dirAbs, _ := filepath.Abs(dir)
	if root != dirAbs {
		t.Fatalf("expected %s, got %s", dirAbs, root)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := git.FindRepoRoot(dir)
	if err == nil {
		t.Fatal("expected error when no .git found")
	}
}

func TestUpdateGitignore(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	os.WriteFile(gitignorePath, []byte("node_modules/\n"), 0644)

	err := git.UpdateGitignore(dir, []string{".env.local", ".envsync/"})
	if err != nil {
		t.Fatalf("UpdateGitignore: %v", err)
	}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)
	if !strings.Contains(content, ".env.local") {
		t.Fatal("should contain .env.local")
	}
	if !strings.Contains(content, ".envsync/") {
		t.Fatal("should contain .envsync/")
	}
	if !strings.Contains(content, "node_modules/") {
		t.Fatal("should preserve existing entries")
	}

	err = git.UpdateGitignore(dir, []string{".env.local", ".envsync/"})
	if err != nil {
		t.Fatalf("second UpdateGitignore: %v", err)
	}
	data2, _ := os.ReadFile(gitignorePath)
	if strings.Count(string(data2), ".env.local") != 1 {
		t.Fatal("should not duplicate entries")
	}
}
