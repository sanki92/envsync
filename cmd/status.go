package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current vault state and team",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultPath := repoRoot + "/.env.vault"
		teamPath := repoRoot + "/.envteam"
		envPath := repoRoot + "/.env.local"

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

		fmt.Printf("Vault:    .env.vault (%d keys, last updated %s)\n", vaultKeyCount, lastUpdate)

		tf, err := team.ReadTeamFile(teamPath)
		if err != nil {
			fmt.Println("Team:     .envteam not found")
		} else {
			names := tf.MemberNames()
			fmt.Printf("Team:     %s\n", strings.Join(names, ", "))
		}

		if _, err := os.Stat(envPath); err != nil {
			fmt.Println("Local:    .env.local not found (run: envsync unlock)")
		} else {
			localEntries, _ := vault.ReadEnvFile(envPath)
			localKeyCount := 0
			for _, e := range localEntries {
				if e.Key != "" {
					localKeyCount++
				}
			}

			vaultKeys := vault.VaultKeys(vaultEntries)
			localMap := vault.EnvMap(localEntries)
			missing := 0
			for _, k := range vaultKeys {
				if _, ok := localMap[k]; !ok {
					missing++
				}
			}

			if missing == 0 && localKeyCount == vaultKeyCount {
				fmt.Printf("Local:    .env.local (%d keys, in sync)\n", localKeyCount)
			} else if missing > 0 {
				fmt.Printf("Local:    .env.local (%d keys, %d missing, run: envsync unlock)\n", localKeyCount, missing)
			} else {
				fmt.Printf("Local:    .env.local (%d keys, vault has %d)\n", localKeyCount, vaultKeyCount)
			}
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
