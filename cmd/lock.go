package cmd

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"

	gitutil "github.com/sanki92/envsync/internal/git"
)

var lockEnvFile string
var lockPush bool

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Encrypt .env.local → .env.vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		envPath := repoRoot + "/" + lockEnvFile
		vaultPath := repoRoot + "/.env.vault"
		teamPath := repoRoot + "/.envteam"

		entries, err := vault.ReadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", lockEnvFile, err)
		}

		tf, err := team.ReadTeamFile(teamPath)
		if err != nil {
			return fmt.Errorf("read .envteam: %w (run 'envsync init' first)", err)
		}

		pubKeys := tf.GetPublicKeys()
		if len(pubKeys) == 0 {
			return fmt.Errorf("no team members found in .envteam")
		}

		recipients, err := crypto.ParseRecipients(pubKeys)
		if err != nil {
			return fmt.Errorf("parse recipient keys: %w", err)
		}

		vaultEntries, err := vault.LockVault(entries, recipients, tf.MemberNames())
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}

		if err := vault.WriteVaultFile(vaultPath, vaultEntries); err != nil {
			return fmt.Errorf("write vault: %w", err)
		}

		kvCount := 0
		for _, e := range entries {
			if e.Key != "" {
				kvCount++
			}
		}
		fmt.Printf("[lock] %d keys -> .env.vault\n", kvCount)

		if lockPush {
			fmt.Println("  (auto-push not implemented yet, commit manually)")
		}

		return nil
	},
}

func init() {
	lockCmd.Flags().StringVar(&lockEnvFile, "env", ".env.local", "Source file")
	lockCmd.Flags().BoolVar(&lockPush, "push", false, "Auto git add + commit")
	rootCmd.AddCommand(lockCmd)
}

func loadIdentity() ([]age.Identity, error) {
	privKeyData, err := config.LoadPrivateKey()
	if err != nil {
		return nil, err
	}

	privKey := strings.TrimSpace(privKeyData)
	id, err := crypto.ParseIdentity(privKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return []age.Identity{id}, nil
}
