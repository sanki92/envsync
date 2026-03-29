package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var lockFile string

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Encrypt .env.local -> .env.vault",
	Example: `  envsync lock
  envsync lock --env staging
  envsync lock --file .env.custom`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		envFile := lockFile
		if envFile == "" {
			envFile = envpath.LocalPath(repoRoot, envFlag)
		} else {
			envFile = repoRoot + "/" + lockFile
		}
		vaultPath := envpath.VaultPath(repoRoot, envFlag)
		teamPath := repoRoot + "/.envteam"

		entries, err := vault.ReadEnvFile(envFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", envFile, err)
		}

		tf, err := team.ReadTeamFile(teamPath)
		if err != nil {
			return fmt.Errorf("read .envteam: %w (run 'envsync init' first)", err)
		}

		sshPubKeys := tf.GetSSHPublicKeysForEnv(envFlag)
		if len(sshPubKeys) == 0 {
			return fmt.Errorf("no team members with access to '%s' environment", envpath.LocalFilename(envFlag))
		}

		memberNames := tf.MemberNamesForEnv(envFlag)
		vaultEntries, err := vault.LockVaultSSH(entries, sshPubKeys, memberNames)
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

		if output.JSONMode {
			output.PrintJSON(output.Result{
				Command: "lock",
				Success: true,
				Data: map[string]interface{}{
					"keys":        kvCount,
					"recipients":  len(sshPubKeys),
					"vault":       envpath.VaultFilename(envFlag),
					"environment": envFlag,
				},
			})
		} else {
			fmt.Printf("[lock] %d keys -> %s\n", kvCount, envpath.VaultFilename(envFlag))
		}

		return nil
	},
}

func init() {
	lockCmd.Flags().StringVar(&lockFile, "file", "", "Source file (overrides default path)")
	rootCmd.AddCommand(lockCmd)
}
