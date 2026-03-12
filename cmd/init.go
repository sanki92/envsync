package cmd

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var initEnvFile string
var initVaultFile string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize envsync in the current repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		envPath := repoRoot + "/" + initEnvFile
		vaultPath := repoRoot + "/" + initVaultFile
		teamPath := repoRoot + "/.envteam"

		if _, err := os.Stat(vaultPath); err == nil {
			return fmt.Errorf(".env.vault already exists, already initialized")
		}

		if _, err := os.Stat(envPath); err != nil {
			return fmt.Errorf("no %s found, create one first with your environment variables", initEnvFile)
		}

		fmt.Println("[init] starting...")

		privKey, pubKey, err := crypto.GenerateKeypair()
		if err != nil {
			return fmt.Errorf("generate keypair: %w", err)
		}

		if err := config.SaveKeypair(privKey, pubKey); err != nil {
			return fmt.Errorf("save keypair: %w", err)
		}
		fmt.Println("  [ok] generated age keypair in ~/.envsync/")

		username := getGitUsername()
		if username == "" {
			username = "owner"
		}

		home, _ := os.UserHomeDir()
		sshFingerprint := ""
		sshPubKey, _, err := crypto.ReadLocalSSHPublicKey(home)
		if err == nil {
			info, err := crypto.FingerprintSSHPublicKey(sshPubKey)
			if err == nil {
				sshFingerprint = info.Fingerprint
			}
		}

		tf := team.NewTeamFile(username, sshFingerprint, pubKey, username)
		if err := team.WriteTeamFile(teamPath, tf); err != nil {
			return fmt.Errorf("write .envteam: %w", err)
		}
		fmt.Println("  [ok] created .envteam")

		entries, err := vault.ReadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", initEnvFile, err)
		}

		recipient, err := crypto.ParseRecipient(pubKey)
		if err != nil {
			return fmt.Errorf("parse own public key: %w", err)
		}

		vaultEntries, err := vault.LockVault(entries, []age.Recipient{recipient}, []string{username})
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
		fmt.Printf("  [ok] encrypted %d keys -> %s\n", kvCount, initVaultFile)

		if err := gitutil.InstallPostMergeHook(repoRoot); err != nil {
			fmt.Printf("  [warn] could not install post-merge hook: %v\n", err)
		} else {
			fmt.Println("  [ok] installed post-merge git hook")
		}

		if err := gitutil.UpdateGitignore(repoRoot, []string{initEnvFile, ".envsync/"}); err != nil {
			fmt.Printf("  [warn] could not update .gitignore: %v\n", err)
		} else {
			fmt.Println("  [ok] updated .gitignore")
		}

		fmt.Println()
		fmt.Println("Done! Next steps:")
		fmt.Printf("  git add %s .envteam .gitignore\n", initVaultFile)
		fmt.Println("  git commit -m \"chore: initialize envsync\"")
		fmt.Println("  git push")

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
	initCmd.Flags().StringVar(&initEnvFile, "env", ".env.local", "Source env file")
	initCmd.Flags().StringVar(&initVaultFile, "vault", ".env.vault", "Vault output file")
	rootCmd.AddCommand(initCmd)
}
