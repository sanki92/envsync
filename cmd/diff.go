package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var diffFrom string
var diffTo string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show what changed in the vault between commits",
	Example: `  envsync diff
  envsync diff --from HEAD~3
  envsync diff --from HEAD~1 --to working`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultFile := envpath.VaultFilename(envFlag)
		oldContent, err := gitShowFile(repoRoot, diffFrom, vaultFile)
		if err != nil {
			return fmt.Errorf("cannot read %s at %s: %w", vaultFile, diffFrom, err)
		}

		var newEntries []vault.EnvEntry
		if diffTo == "working" {
			newEntries, err = vault.ReadVaultFile(envpath.VaultPath(repoRoot, envFlag))
			if err != nil {
				return fmt.Errorf("read current %s: %w", vaultFile, err)
			}
		} else {
			newContent, err := gitShowFile(repoRoot, diffTo, vaultFile)
			if err != nil {
				return fmt.Errorf("cannot read %s at %s: %w", vaultFile, diffTo, err)
			}
			newEntries = parseEnvString(newContent)
		}

		oldEntries := parseEnvString(oldContent)
		oldMap := vault.EnvMap(oldEntries)
		newMap := vault.EnvMap(newEntries)

		type change struct {
			key    string
			action string
		}
		var changes []change

		for k, v := range newMap {
			oldVal, existed := oldMap[k]
			if !existed {
				changes = append(changes, change{k, "added"})
			} else if oldVal != v {
				changes = append(changes, change{k, "changed"})
			}
		}
		for k := range oldMap {
			if _, exists := newMap[k]; !exists {
				changes = append(changes, change{k, "removed"})
			}
		}

		sort.Slice(changes, func(i, j int) bool {
			return changes[i].key < changes[j].key
		})

		if len(changes) == 0 {
			fmt.Printf("No changes to %s between %s and %s\n", vaultFile, diffFrom, diffTo)
			return nil
		}

		fmt.Printf("Changes to %s (%s..%s):\n", vaultFile, diffFrom, diffTo)
		for _, c := range changes {
			switch c.action {
			case "added":
				fmt.Printf("  + %s\n", c.key)
			case "removed":
				fmt.Printf("  - %s\n", c.key)
			case "changed":
				fmt.Printf("  ~ %s\n", c.key)
			}
		}

		return nil
	},
}

func gitShowFile(repoRoot, ref, file string) (string, error) {
	gitCmd := exec.Command("git", "show", ref+":"+file)
	gitCmd.Dir = repoRoot
	out, err := gitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show %s:%s: %w", ref, file, err)
	}
	return string(out), nil
}

func parseEnvString(content string) []vault.EnvEntry {
	var entries []vault.EnvEntry
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.IndexByte(trimmed, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		value := strings.TrimSpace(trimmed[idx+1:])
		entries = append(entries, vault.EnvEntry{Key: key, Value: value})
	}
	return entries
}

func init() {
	diffCmd.Flags().StringVar(&diffFrom, "from", "HEAD~1", "Git ref to compare from")
	diffCmd.Flags().StringVar(&diffTo, "to", "HEAD", "Git ref to compare to (or 'working' for current file)")
	rootCmd.AddCommand(diffCmd)
}
