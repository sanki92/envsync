package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sanki92/envsync/internal/crypto"
	"github.com/sanki92/envsync/internal/envpath"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize envsync in the current repository",
	Example: `  envsync init
  envsync init --env staging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		envPath := envpath.LocalPath(repoRoot, envFlag)
		vaultPath := envpath.VaultPath(repoRoot, envFlag)
		teamPath := repoRoot + "/.envteam"

		if _, err := os.Stat(vaultPath); err == nil {
			return fmt.Errorf("%s already exists, already initialized", envpath.VaultFilename(envFlag))
		}

		if _, err := os.Stat(envPath); err != nil {
			return fmt.Errorf("no %s found, create one first with your environment variables", envpath.LocalFilename(envFlag))
		}

		output.Step(1, 5, "checking SSH key...")

		home, _ := os.UserHomeDir()
		sshPubKey, sshPath, err := crypto.ReadLocalSSHPublicKey(home)
		if err != nil {
			return fmt.Errorf("no SSH key found, run: ssh-keygen -t ed25519 -C \"your-email@example.com\"")
		}

		info, err := crypto.FingerprintSSHPublicKey(sshPubKey)
		if err != nil {
			return fmt.Errorf("fingerprint SSH key: %w", err)
		}
		output.OK(fmt.Sprintf("SSH key found at %s (%s)", sshPath, info.Type))

		username := getGitUsername()
		if username == "" {
			username = "owner"
		}

		output.Step(2, 5, "creating team manifest...")
		tf := team.NewTeamFile(username, info.Fingerprint, sshPubKey, username)
		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}
		output.OK("created .envteam")

		output.Step(3, 5, "encrypting vault...")
		entries, err := vault.ReadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", envpath.LocalFilename(envFlag), err)
		}

		vaultEntries, err := vault.LockVaultSSH(entries, []string{sshPubKey}, []string{username})
		if err != nil {
			return fmt.Errorf("encrypt vault: %w", err)
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
		output.OK(fmt.Sprintf("encrypted %d keys -> %s", kvCount, envpath.VaultFilename(envFlag)))

		output.Step(4, 5, "installing git hook...")
		if err := gitutil.InstallPostMergeHook(repoRoot); err != nil {
			output.Warn(fmt.Sprintf("could not install post-merge hook: %v", err))
		} else {
			output.OK("installed post-merge git hook")
		}

		output.Step(5, 5, "updating .gitignore...")
		if err := gitutil.UpdateGitignore(repoRoot, []string{envpath.LocalFilename(envFlag)}); err != nil {
			output.Warn(fmt.Sprintf("could not update .gitignore: %v", err))
		} else {
			output.OK("updated .gitignore")
		}

		if output.JSONMode {
			output.PrintJSON(output.Result{
				Command: "init",
				Success: true,
				Data: map[string]interface{}{
					"keys":       kvCount,
					"vault":      envpath.VaultFilename(envFlag),
					"team":       []string{username},
					"repository": repoRoot,
				},
			})
		} else {
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  git add %s .envteam .gitignore\n", envpath.VaultFilename(envFlag))
			fmt.Println("  git commit -m \"chore: initialize envsync\"")
			fmt.Println("  git push")
		}

		return nil
	},
}

func getGitUsername() string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(home + "/.gitconfig")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(initCmd)
}
