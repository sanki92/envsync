package git

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const hookScript = `#!/bin/sh
# envsync post-merge hook - auto-decrypt vault on git pull
npx envsync unlock --quiet 2>/dev/null || true
`

func InstallPostMergeHook(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "post-merge")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	existing, err := os.ReadFile(hookPath)
	if err == nil {

		if strings.Contains(string(existing), "envsync") {
			return nil
		}

		content := string(existing) + "\n" + hookScript
		return writeExecutable(hookPath, content)
	}

	return writeExecutable(hookPath, hookScript)
}

func UninstallPostMergeHook(repoRoot string) error {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-merge")

	data, err := os.ReadFile(hookPath)
	if err != nil {
		return nil
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "envsync") {
			continue
		}
		lines = append(lines, line)
	}

	cleaned := strings.TrimSpace(strings.Join(lines, "\n"))
	if cleaned == "" || cleaned == "#!/bin/sh" {
		return os.Remove(hookPath)
	}

	return writeExecutable(hookPath, cleaned+"\n")
}

func HasPostMergeHook(repoRoot string) bool {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-merge")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "envsync")
}

func FindRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (no .git found)")
		}
		dir = parent
	}
}

func UpdateGitignore(repoRoot string, entries []string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	content := existing
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}
	content += "\n# envsync\n"
	for _, entry := range toAdd {
		content += entry + "\n"
	}

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

func writeExecutable(path, content string) error {
	perm := os.FileMode(0755)
	if runtime.GOOS == "windows" {
		perm = 0644
	}
	return os.WriteFile(path, []byte(content), perm)
}
