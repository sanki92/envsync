package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/output"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Diagnose your envsync setup and environment health",
	Example: `  envsync doctor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		type check struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail,omitempty"`
			Fix    string `json:"fix,omitempty"`
		}
		var checks []check
		issues := 0

		if !output.JSONMode {
			fmt.Println("envsync doctor")
			fmt.Println()
		}

		home, _ := os.UserHomeDir()

		sshPubKey, sshPath, err := crypto.ReadLocalSSHPublicKey(home)
		if err != nil {
			checks = append(checks, check{"ssh_key", "miss", "No SSH key found", "ssh-keygen -t ed25519 -C \"your-email@example.com\""})
			if !output.JSONMode {
				fmt.Println("SSH key          [MISS] No SSH key found at ~/.ssh/id_ed25519 or ~/.ssh/id_rsa")
				fmt.Println("                        Fix: ssh-keygen -t ed25519 -C \"your-email@example.com\"")
				fmt.Println("                        Then add to GitHub: github.com/settings/keys")
			}
			issues++
		} else {
			info, _ := crypto.FingerprintSSHPublicKey(sshPubKey)
			checks = append(checks, check{"ssh_key", "ok", fmt.Sprintf("%s (%s)", sshPath, info.Type), ""})
			if !output.JSONMode {
				fmt.Printf("SSH key          [ OK ] Found at %s (%s)\n", sshPath, info.Type)
			}

			privKeyPath, found := crypto.HasSSHPrivateKey(home)
			if found {
				hasPass, _ := crypto.HasSSHPassphrase(privKeyPath)
				if !hasPass {
					checks = append(checks, check{"ssh_passphrase", "warn", "no passphrase set", fmt.Sprintf("ssh-keygen -p -f %s", privKeyPath)})
					if !output.JSONMode {
						fmt.Println("SSH passphrase   [warn] no passphrase set, anyone with laptop access can decrypt")
						fmt.Printf("                        fix: ssh-keygen -p -f %s\n", privKeyPath)
					}
				} else {
					checks = append(checks, check{"ssh_passphrase", "ok", "passphrase protected", ""})
					if !output.JSONMode {
						fmt.Println("SSH passphrase   [ OK ] Passphrase protected")
					}
				}
			}
		}

		cwd, _ := os.Getwd()
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			checks = append(checks, check{"git_repo", "miss", "not inside a git repository", ""})
			if !output.JSONMode {
				fmt.Println("Git repo         [MISS] Not inside a git repository")
			}
			issues++
			return printDoctorResult(checks, issues)
		}

		vaultPath := repoRoot + "/.env.vault"
		teamPath := repoRoot + "/.envteam"
		envPath := repoRoot + "/.env.local"

		if _, err := os.Stat(vaultPath); err != nil {
			checks = append(checks, check{"vault", "miss", ".env.vault not found", "envsync init"})
			if !output.JSONMode {
				fmt.Println("Vault            [miss] .env.vault not found, run 'envsync init'")
			}
			issues++
		} else {
			header, err := vault.ReadVaultHeader(vaultPath)
			if err != nil {
				checks = append(checks, check{"vault", "err", "cannot read .env.vault header", ""})
				if !output.JSONMode {
					fmt.Println("Vault            [ERR ] Cannot read .env.vault header")
				}
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
				checks = append(checks, check{"vault", "ok", fmt.Sprintf("%d keys, last updated %s", keyCount, lastUpdate), ""})
				if !output.JSONMode {
					fmt.Printf("Vault            [ OK ] %d keys, last updated %s\n", keyCount, lastUpdate)
				}
			}
		}

		if _, err := os.Stat(teamPath); err != nil {
			checks = append(checks, check{"team", "miss", ".envteam not found", ""})
			if !output.JSONMode {
				fmt.Println("Team             [MISS] .envteam not found")
			}
			issues++
		} else {
			tf, err := team.ReadTeamFile(teamPath)
			if err != nil {
				checks = append(checks, check{"team", "err", "cannot parse .envteam", ""})
				if !output.JSONMode {
					fmt.Println("Team             [ERR ] Cannot parse .envteam")
				}
				issues++
			} else {
				names := tf.MemberNames()
				checks = append(checks, check{"team", "ok", fmt.Sprintf("%d members: %s", len(names), strings.Join(names, ", ")), ""})
				if !output.JSONMode {
					fmt.Printf("Team             [ OK ] %d members: %s\n", len(names), strings.Join(names, ", "))
				}
			}
		}

		if gitutil.HasPostMergeHook(repoRoot) {
			checks = append(checks, check{"post_merge_hook", "ok", "installed", ""})
			if !output.JSONMode {
				fmt.Println("Post-merge hook  [ OK ] Installed")
			}
		} else {
			checks = append(checks, check{"post_merge_hook", "miss", "not installed", "envsync init or envsync join"})
			if !output.JSONMode {
				fmt.Println("Post-merge hook  [miss] not installed, run 'envsync init' or 'envsync join'")
			}
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
					if !output.JSONMode {
						fmt.Println()
						fmt.Println("Missing from .env.local:")
						for _, k := range missing {
							fmt.Printf("  %s  (in vault, not in .env.local, run: envsync unlock)\n", k)
						}
					}
					issues++
				}
			}
		}

		return printDoctorResult(checks, issues)
	},
}

func printDoctorResult(checks interface{}, issues int) error {
	if output.JSONMode {
		output.PrintJSON(output.Result{
			Command: "doctor",
			Success: issues == 0,
			Data: map[string]interface{}{
				"checks": checks,
				"issues": issues,
			},
		})
		return nil
	}

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
