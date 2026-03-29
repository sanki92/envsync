package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/crypto"
	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	gh "github.com/sanki92/envsync/internal/github"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <github-username>",
	Short: "Add a team member as a vault recipient",
	Example: `  envsync add alice
  envsync add bob --env staging`,
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
			return fmt.Errorf("read .envteam: %w (run 'envsync init' first)", err)
		}

		existingMember, memberExists := tf.GetMember(username)

		if memberExists && envFlag == "" {
			return fmt.Errorf("%s is already a team member", username)
		}

		if memberExists && envFlag != "" {
			for _, e := range existingMember.Environments {
				if e == envFlag {
					return fmt.Errorf("%s already has access to %s", username, envFlag)
				}
			}
		}

		var sshKey string
		var fingerprint string

		if memberExists {
			sshKey = existingMember.SSHPublicKey
			fingerprint = existingMember.SSHFingerprint
			output.OK(fmt.Sprintf("%s already in team, granting %s access", username, envFlag))
		} else {
			output.Step(1, 3, fmt.Sprintf("fetching SSH keys for %s from GitHub...", username))

			keys, err := gh.FetchSSHKeys(username)
			if err != nil {
				return fmt.Errorf("fetch SSH keys: %w", err)
			}

			sshKey = gh.FindEd25519Key(keys)
			if sshKey == "" {
				return fmt.Errorf("no SSH keys found for %s on GitHub\n\n  They need to:\n    1. ssh-keygen -t ed25519\n    2. Add the key at github.com/settings/keys", username)
			}

			info, err := crypto.FingerprintSSHPublicKey(sshKey)
			if err != nil {
				return fmt.Errorf("fingerprint SSH key: %w", err)
			}
			fingerprint = info.Fingerprint
			output.OK(fmt.Sprintf("SSH key: %s (%s)", info.Fingerprint, info.Type))

			if _, err := crypto.ParseSSHRecipient(sshKey); err != nil {
				return fmt.Errorf("SSH key not usable for encryption: %w", err)
			}
		}

		if memberExists && envFlag != "" {
			if err := tf.AddEnvToMember(username, envFlag); err != nil {
				return fmt.Errorf("grant env access: %w", err)
			}
		} else {
			addedBy := "unknown"
			for name := range tf.Members {
				addedBy = name
				break
			}

			var envs []string
			if envFlag != "" {
				envs = []string{envFlag}
			}

			if err := tf.AddMember(username, fingerprint, sshKey, addedBy); err != nil {
				return fmt.Errorf("add member: %w", err)
			}

			if len(envs) > 0 {
				m, _ := tf.GetMember(username)
				m.Environments = envs
				tf.Members[username] = m
			}
		}

		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}

		if !memberExists {
			output.OK(fmt.Sprintf("added %s to .envteam", username))
		}

		output.Step(2, 3, "re-encrypting vault...")

		entries, err := vault.ReadEnvFile(localPath)
		if err != nil {
			return fmt.Errorf("read %s: %w (unlock first if needed)", envpath.LocalFilename(envFlag), err)
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

		output.Step(3, 3, "done")

		if output.JSONMode {
			output.PrintJSON(output.Result{
				Command: "add",
				Success: true,
				Data: map[string]interface{}{
					"member":      username,
					"recipients":  len(sshPubKeys),
					"environment": envFlag,
				},
			})
		} else {
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  git add " + envpath.VaultFilename(envFlag) + " .envteam")
			fmt.Printf("  git commit -m \"chore: add %s to envsync\"\n", username)
			fmt.Println("  git push")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
