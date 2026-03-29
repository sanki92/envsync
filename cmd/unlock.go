package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var unlockFile string
var unlockQuiet bool

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Decrypt .env.vault -> .env.local",
	Example: `  envsync unlock
  envsync unlock --env staging
  envsync unlock --file .env.custom`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultPath := envpath.VaultPath(repoRoot, envFlag)
		outFile := unlockFile
		if outFile == "" {
			outFile = envpath.LocalPath(repoRoot, envFlag)
		} else {
			outFile = repoRoot + "/" + unlockFile
		}

		vaultEntries, err := vault.ReadVaultFile(vaultPath)
		if err != nil {
			return fmt.Errorf("read %s: %w (has 'envsync init' been run?)", envpath.VaultFilename(envFlag), err)
		}

		home, _ := os.UserHomeDir()
		entries, err := vault.UnlockVaultSSH(vaultEntries, home)
		if err != nil {
			return fmt.Errorf("decrypt: %w\n\n  Make sure your SSH key is added as a team recipient.\n  Ask an admin to run: envsync add <your-github-username>", err)
		}

		if err := vault.WriteEnvFile(outFile, entries); err != nil {
			return fmt.Errorf("write %s: %w", outFile, err)
		}

		if output.JSONMode {
			output.PrintJSON(output.Result{
				Command: "unlock",
				Success: true,
				Data: map[string]interface{}{
					"keys":        len(entries),
					"output":      envpath.LocalFilename(envFlag),
					"environment": envFlag,
				},
			})
		} else if !unlockQuiet {
			fmt.Printf("[unlock] %d keys -> %s\n", len(entries), envpath.LocalFilename(envFlag))
		}

		return nil
	},
}

func init() {
	unlockCmd.Flags().StringVar(&unlockFile, "file", "", "Output file (overrides default path)")
	unlockCmd.Flags().BoolVar(&unlockQuiet, "quiet", false, "Suppress output (for git hooks)")
	rootCmd.AddCommand(unlockCmd)
}
