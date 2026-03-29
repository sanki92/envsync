package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join an existing envsync vault as a new team member",
	Example: `  envsync join`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return fmt.Errorf("must be inside a git repository: %w", err)
		}

		vaultPath := repoRoot + "/.env.vault"
		if _, err := os.Stat(vaultPath); err != nil {
			return fmt.Errorf("no .env.vault found, this repo has not been initialized with envsync")
		}

		fmt.Println("[join] checking local setup...")

		home, _ := os.UserHomeDir()
		sshPubKey, sshPath, err := crypto.ReadLocalSSHPublicKey(home)
		if err != nil {
			fmt.Println()
			fmt.Println("[err] no SSH key found")
			fmt.Println()
			fmt.Println("  You need an SSH key for envsync encryption.")
			fmt.Println("  You can keep using HTTPS/GCM for git operations.")
			fmt.Println()
			fmt.Println("  Run:")
			fmt.Println("    ssh-keygen -t ed25519 -C \"your-email@example.com\"")
			fmt.Println("    Then add to GitHub: github.com/settings/keys")
			fmt.Println()
			fmt.Println("  Then run: envsync join")
			return fmt.Errorf("no SSH key found")
		}

		info, err := crypto.FingerprintSSHPublicKey(sshPubKey)
		if err != nil {
			return fmt.Errorf("fingerprint SSH key: %w", err)
		}
		fmt.Printf("  [ok] SSH key at %s (%s)\n", sshPath, info.Type)

		privKeyPath, found := crypto.HasSSHPrivateKey(home)
		if found {
			hasPass, _ := crypto.HasSSHPassphrase(privKeyPath)
			if !hasPass {
				fmt.Println("  [warn] SSH key has no passphrase, anyone with laptop access can decrypt")
				fmt.Printf("     fix: ssh-keygen -p -f %s\n", privKeyPath)
			}
		}

		if err := gitutil.InstallPostMergeHook(repoRoot); err != nil {
			fmt.Printf("  [warn] could not install post-merge hook: %v\n", err)
		} else {
			fmt.Println("  [ok] installed post-merge git hook")
		}

		fmt.Println()
		fmt.Println("Done! Ask a team admin to run:")
		fmt.Println("  envsync add <your-github-username>")
		fmt.Println()
		fmt.Println("Once added, pull the latest changes and run:")
		fmt.Println("  envsync unlock")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}
