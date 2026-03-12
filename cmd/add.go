package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	gh "github.com/sanki92/envsync/internal/github"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var addAgeKey string

var addCmd = &cobra.Command{
	Use:   "add <github-username>",
	Short: "Add a team member as a vault recipient",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		if addAgeKey == "" {
			return fmt.Errorf("--key is required, ask the user to run 'envsync join' and share their age public key")
		}

		if _, err := crypto.ParseRecipient(addAgeKey); err != nil {
			return fmt.Errorf("invalid age public key: %w", err)
		}

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
			return fmt.Errorf("read .envteam: %w (run 'envsync init' first)", err)
		}

		if _, exists := tf.GetMember(username); exists {
			return fmt.Errorf("%s is already a team member", username)
		}

		fmt.Printf("Fetching SSH keys for %s from GitHub...\n", username)

		keys, err := gh.FetchSSHKeys(username)
		if err != nil {
			return fmt.Errorf("fetch SSH keys: %w", err)
		}

		sshKey := gh.FindEd25519Key(keys)
		if sshKey == "" {
			return fmt.Errorf("no SSH keys found for %s on GitHub", username)
		}

		info, err := crypto.FingerprintSSHPublicKey(sshKey)
		if err != nil {
			return fmt.Errorf("fingerprint SSH key: %w", err)
		}

		fmt.Printf("  SSH key: %s (%s)\n", info.Fingerprint, info.Type)

		addedBy := "unknown"
		for name := range tf.Members {
			addedBy = name
			break
		}

		if err := tf.AddMember(username, info.Fingerprint, addAgeKey, addedBy); err != nil {
			return fmt.Errorf("add member: %w", err)
		}

		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}
		fmt.Printf("  [ok] added %s to .envteam\n", username)

		entries, err := vault.ReadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("read .env.local: %w (unlock first if needed)", err)
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
		fmt.Println("Next steps:")
		fmt.Println("  git add .env.vault .envteam")
		fmt.Printf("  git commit -m \"chore: add %s to envsync\"\n", username)
		fmt.Println("  git push")

		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addAgeKey, "key", "", "The user's age public key (from 'envsync join' output)")
	rootCmd.AddCommand(addCmd)
}
