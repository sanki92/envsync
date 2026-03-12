package cmd

import (
	"fmt"
	"os"

	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var unlockEnvFile string
var unlockQuiet bool

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Decrypt .env.vault → .env.local",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultPath := repoRoot + "/.env.vault"
		envPath := repoRoot + "/" + unlockEnvFile

		vaultEntries, err := vault.ReadVaultFile(vaultPath)
		if err != nil {
			return fmt.Errorf("read .env.vault: %w (has 'envsync init' been run?)", err)
		}

		identities, err := loadIdentity()
		if err != nil {
			return fmt.Errorf("load identity: %w", err)
		}

		entries, err := vault.UnlockVault(vaultEntries, identities)
		if err != nil {
			return fmt.Errorf("decrypt: %w", err)
		}

		if err := vault.WriteEnvFile(envPath, entries); err != nil {
			return fmt.Errorf("write %s: %w", unlockEnvFile, err)
		}

		if !unlockQuiet {
			fmt.Printf("[unlock] %d keys -> %s\n", len(entries), unlockEnvFile)
		}

		return nil
	},
}

func init() {
	unlockCmd.Flags().StringVar(&unlockEnvFile, "env", ".env.local", "Output file")
	unlockCmd.Flags().BoolVar(&unlockQuiet, "quiet", false, "Suppress output (for git hooks)")
	rootCmd.AddCommand(unlockCmd)
}
