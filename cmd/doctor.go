package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose your envsync setup and environment health",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("envsync doctor")
		fmt.Println()

		home, _ := os.UserHomeDir()
		issues := 0

		sshPubKey, sshPath, err := crypto.ReadLocalSSHPublicKey(home)
		if err != nil {
			fmt.Println("SSH key          [MISS] No SSH key found at ~/.ssh/id_ed25519 or ~/.ssh/id_rsa")
			fmt.Println("                        Fix: ssh-keygen -t ed25519 -C \"your-email@example.com\"")
			issues++
		} else {
			info, _ := crypto.FingerprintSSHPublicKey(sshPubKey)
			fmt.Printf("SSH key          [ OK ] Found at %s (%s)\n", sshPath, info.Type)

			privKeyPath, found := crypto.HasSSHPrivateKey(home)
			if found {
				hasPass, _ := crypto.HasSSHPassphrase(privKeyPath)
				if !hasPass {
					fmt.Println("SSH passphrase   [warn] no passphrase set, anyone with laptop access can decrypt")
					fmt.Printf("                        fix: ssh-keygen -p -f %s\n", privKeyPath)
				} else {
					fmt.Println("SSH passphrase   [ OK ] Passphrase protected")
				}
			}
		}

		if config.HasKeypair() {
			fmt.Println("age identity     [ OK ] Found in ~/.envsync/")
		} else {
			fmt.Println("age identity     [miss] not found, run 'envsync init' or 'envsync join'")
			issues++
		}

		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			fmt.Println("Git repo         [MISS] Not inside a git repository")
			issues++
			return printDoctorSummary(issues)
		}

		vaultPath := repoRoot + "/.env.vault"
		teamPath := repoRoot + "/.envteam"
		envPath := repoRoot + "/.env.local"

		if _, err := os.Stat(vaultPath); err != nil {
			fmt.Println("Vault            [miss] .env.vault not found, run 'envsync init'")
			issues++
		} else {
			header, err := vault.ReadVaultHeader(vaultPath)
			if err != nil {
				fmt.Println("Vault            [ERR ] Cannot read .env.vault header")
				issues++
			} else {
				vaultEntries, _ := vault.ReadVaultFile(vaultPath)
				keyCount := 0
				for _, e := range vaultEntries {
					if e.Key != "" {
						keyCount++
					}
				}
				lastUpdate := "unknown"
				if !header.LastUpdated.IsZero() {
					lastUpdate = header.LastUpdated.Format("2006-01-02 15:04")
				}
				fmt.Printf("Vault            [ OK ] %d keys, last updated %s\n", keyCount, lastUpdate)
			}
		}

		if _, err := os.Stat(teamPath); err != nil {
			fmt.Println("Team             [MISS] .envteam not found")
			issues++
		} else {
			tf, err := team.ReadTeamFile(teamPath)
			if err != nil {
				fmt.Println("Team             [ERR ] Cannot parse .envteam")
				issues++
			} else {
				names := tf.MemberNames()
				fmt.Printf("Team             [ OK ] %d members: %s\n", len(names), strings.Join(names, ", "))
			}
		}

		if gitutil.HasPostMergeHook(repoRoot) {
			fmt.Println("Post-merge hook  [ OK ] Installed")
		} else {
			fmt.Println("Post-merge hook  [miss] not installed, run 'envsync init' or 'envsync join'")
			issues++
		}

		if _, err := os.Stat(envPath); err == nil {
			if _, err := os.Stat(vaultPath); err == nil {
				localEntries, _ := vault.ReadEnvFile(envPath)
				vaultEntries, _ := vault.ReadVaultFile(vaultPath)
				localKeys := vault.EnvMap(localEntries)
				vaultKeys := vault.VaultKeys(vaultEntries)

				var missing []string
				for _, k := range vaultKeys {
					if _, ok := localKeys[k]; !ok {
						missing = append(missing, k)
					}
				}
				if len(missing) > 0 {
					fmt.Println()
					fmt.Println("Missing from .env.local:")
					for _, k := range missing {
						fmt.Printf("  %s  (in vault, not in .env.local, run: envsync unlock)\n", k)
					}
					issues++
				}
			}
		}

		return printDoctorSummary(issues)
	},
}

func printDoctorSummary(issues int) error {
	fmt.Println()
	if issues == 0 {
		fmt.Println("All checks passed.")
	} else {
		fmt.Printf("%d issue(s) found.\n", issues)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
