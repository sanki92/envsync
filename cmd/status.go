package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show current vault state and team",
	Example: `  envsync status
  envsync status --env staging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultPath := envpath.VaultPath(repoRoot, envFlag)
		localPath := envpath.LocalPath(repoRoot, envFlag)
		teamPath := repoRoot + "/.envteam"

		header, err := vault.ReadVaultHeader(vaultPath)
		if err != nil {
			return fmt.Errorf("read vault: %w (run 'envsync init' first)", err)
		}

		vaultEntries, err := vault.ReadVaultFile(vaultPath)
		if err != nil {
			return fmt.Errorf("read vault file: %w", err)
		}

		vaultKeyCount := 0
		for _, e := range vaultEntries {
			if e.Key != "" {
				vaultKeyCount++
			}
		}

		lastUpdate := "unknown"
		if !header.LastUpdated.IsZero() {
			lastUpdate = formatRelativeTime(header.LastUpdated)
		}

		var members []string
		tf, tfErr := team.ReadTeamFile(teamPath)
		if tfErr == nil {
			if envFlag != "" {
				members = tf.MemberNamesForEnv(envFlag)
			} else {
				members = tf.MemberNames()
			}
		}

		localKeyCount := 0
		missing := 0
		localExists := false
		if _, err := os.Stat(localPath); err == nil {
			localExists = true
			localEntries, _ := vault.ReadEnvFile(localPath)
			for _, e := range localEntries {
				if e.Key != "" {
					localKeyCount++
				}
			}

			vaultKeys := vault.VaultKeys(vaultEntries)
			localMap := vault.EnvMap(localEntries)
			for _, k := range vaultKeys {
				if _, ok := localMap[k]; !ok {
					missing++
				}
			}
		}

		if output.JSONMode {
			data := map[string]interface{}{
				"vault_keys":   vaultKeyCount,
				"last_updated": lastUpdate,
				"team":         members,
				"local_exists": localExists,
				"local_keys":   localKeyCount,
				"missing_keys": missing,
				"in_sync":      localExists && missing == 0 && localKeyCount == vaultKeyCount,
				"environment":  envFlag,
			}
			output.PrintJSON(output.Result{
				Command: "status",
				Success: true,
				Data:    data,
			})
			return nil
		}

		vaultLabel := envpath.VaultFilename(envFlag)
		localLabel := envpath.LocalFilename(envFlag)

		fmt.Printf("Vault:    %s (%d keys, last updated %s)\n", vaultLabel, vaultKeyCount, lastUpdate)

		if tfErr != nil {
			fmt.Println("Team:     .envteam not found")
		} else {
			fmt.Printf("Team:     %s\n", strings.Join(members, ", "))
		}

		if !localExists {
			fmt.Printf("Local:    %s not found (run: envsync unlock%s)\n", localLabel, envFlagStr())
		} else if missing == 0 && localKeyCount == vaultKeyCount {
			fmt.Printf("Local:    %s (%d keys, in sync)\n", localLabel, localKeyCount)
		} else if missing > 0 {
			fmt.Printf("Local:    %s (%d keys, %d missing, run: envsync unlock%s)\n", localLabel, localKeyCount, missing, envFlagStr())
		} else {
			fmt.Printf("Local:    %s (%d keys, vault has %d)\n", localLabel, localKeyCount, vaultKeyCount)
		}

		return nil
	},
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		d := int(diff.Hours() / 24)
		if d == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", d)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
