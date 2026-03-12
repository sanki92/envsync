package cmd

import (
	"fmt"
	"os"

	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join an existing envsync vault as a new team member",
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
			fmt.Println("[error] no SSH key found")
			fmt.Println()
			fmt.Println("Run:")
			fmt.Println("  ssh-keygen -t ed25519 -C \"your-email@example.com\"")
			fmt.Println("  Then add to GitHub: github.com/settings/keys")
			fmt.Println()
			fmt.Println("Then run: envsync join again")
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

		if config.HasKeypair() {
			fmt.Println("  [ok] age keypair already exists in ~/.envsync/")
		} else {
			privKey, pubKey, err := crypto.GenerateKeypair()
			if err != nil {
				return fmt.Errorf("generate keypair: %w", err)
			}
			if err := config.SaveKeypair(privKey, pubKey); err != nil {
				return fmt.Errorf("save keypair: %w", err)
			}
			fmt.Println("  [ok] generated age keypair in ~/.envsync/")
		}

		pubKey, err := config.LoadPublicKey()
		if err != nil {
			return fmt.Errorf("load public key: %w", err)
		}

		if err := gitutil.InstallPostMergeHook(repoRoot); err != nil {
			fmt.Printf("  [warn] could not install post-merge hook: %v\n", err)
		} else {
			fmt.Println("  [ok] installed post-merge git hook")
		}

		fmt.Println()
		fmt.Println("Almost done! Ask a team admin to run:")
		fmt.Printf("  envsync add <your-github-username>\n")
		fmt.Println()
		fmt.Printf("Your age public key: %s\n", pubKey)
		fmt.Println()
		fmt.Println("Once added, run: envsync unlock")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}
