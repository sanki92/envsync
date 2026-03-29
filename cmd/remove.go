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

var removeCmd = &cobra.Command{
	Use:   "remove <github-username>",
	Short: "Remove a team member's vault access",
	Example: `  envsync remove alice
  envsync remove bob --env staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		localPath := envpath.LocalPath(repoRoot, envFlag)
		vaultPath := envpath.VaultPath(repoRoot, envFlag)
		teamPath := repoRoot + "/.envteam"

		tf, err := team.ReadTeamFile(teamPath)
		if err != nil {
			return fmt.Errorf("read .envteam: %w", err)
		}

		if _, exists := tf.GetMember(username); !exists {
			return fmt.Errorf("%s is not a team member", username)
		}

		if envFlag != "" {
			if err := tf.RemoveEnvFromMember(username, envFlag); err != nil {
				return fmt.Errorf("remove env access: %w", err)
			}
			output.OK(fmt.Sprintf("removed %s access from %s", envFlag, username))
		} else {
			if err := tf.RemoveMember(username); err != nil {
				return fmt.Errorf("remove member: %w", err)
			}
			output.OK(fmt.Sprintf("removed %s from .envteam", username))
		}

		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}

		entries, err := vault.ReadEnvFile(localPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", envpath.LocalFilename(envFlag), err)
		}

		sshPubKeys := tf.GetSSHPublicKeysForEnv(envFlag)
		memberNames := tf.MemberNamesForEnv(envFlag)
		vaultEntries, err := vault.LockVaultSSH(entries, sshPubKeys, memberNames)
		if err != nil {
			return fmt.Errorf("re-encrypt vault: %w", err)
		}

		if err := vault.WriteVaultFile(vaultPath, vaultEntries); err != nil {
			return fmt.Errorf("write vault: %w", err)
		}
		output.OK(fmt.Sprintf("re-encrypted %s for %d recipients", envpath.VaultFilename(envFlag), len(sshPubKeys)))

		if output.JSONMode {
			output.PrintJSON(output.Result{
				Command: "remove",
				Success: true,
				Data: map[string]interface{}{
					"member":      username,
					"recipients":  len(sshPubKeys),
					"environment": envFlag,
				},
			})
		} else {
			fmt.Println()
			fmt.Println("[note] " + username + " already saw the secret values.")
			fmt.Println("   You should rotate all secrets that " + username + " had access to.")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Rotate actual secret values in " + envpath.LocalFilename(envFlag))
			fmt.Println("  2. envsync lock" + envFlagStr())
			fmt.Println("  3. git add " + envpath.VaultFilename(envFlag) + " .envteam")
			fmt.Printf("  4. git commit -m \"chore: remove %s from envsync\"\n", username)
			fmt.Println("  5. git push")
		}

		return nil
	},
}

func envFlagStr() string {
	if envFlag != "" {
		return " --env " + envFlag
	}
	return ""
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
