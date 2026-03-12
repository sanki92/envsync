package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <github-username>",
	Short: "Remove a team member's vault access",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		envPath := repoRoot + "/.env.local"
		vaultPath := repoRoot + "/.env.vault"
		teamPath := repoRoot + "/.envteam"

		tf, err := team.ReadTeamFile(teamPath)
		if err != nil {
			return fmt.Errorf("read .envteam: %w", err)
		}

		if _, exists := tf.GetMember(username); !exists {
			return fmt.Errorf("%s is not a team member", username)
		}

		if err := tf.RemoveMember(username); err != nil {
			return fmt.Errorf("remove member: %w", err)
		}

		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}
		fmt.Printf("  [ok] removed %s from .envteam\n", username)

		entries, err := vault.ReadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("read .env.local: %w", err)
		}

		pubKeys := tf.GetPublicKeys()
		recipients, err := crypto.ParseRecipients(pubKeys)
		if err != nil {
			return fmt.Errorf("parse recipients: %w", err)
		}

		vaultEntries, err := vault.LockVault(entries, recipients, tf.MemberNames())
		if err != nil {
			return fmt.Errorf("re-encrypt vault: %w", err)
		}

		if err := vault.WriteVaultFile(vaultPath, vaultEntries); err != nil {
			return fmt.Errorf("write vault: %w", err)
		}
		fmt.Printf("  [ok] re-encrypted vault for %d recipients\n", len(recipients))

		fmt.Println()
		fmt.Println("[note] " + username + " already saw the secret values.")
		fmt.Println("   You should rotate all secrets that " + username + " had access to.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Rotate actual secret values in .env.local")
		fmt.Println("  2. envsync lock")
		fmt.Println("  3. git add .env.vault .envteam")
		fmt.Printf("  4. git commit -m \"chore: remove %s from envsync\"\n", username)
		fmt.Println("  5. git push")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
