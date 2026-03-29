package cmd

import (
	"github.com/sanki92/envsync/internal/output"
	"github.com/spf13/cobra"
)

var envFlag string

var rootCmd = &cobra.Command{
	Use:   "envsync",
	Short: "Encrypted .env sync for dev teams",
	Long: `envsync encrypts .env files per-value using age encryption and syncs them
through git using GitHub SSH keys for identity.

No server. No vendor. No shared secrets. Just git.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&envFlag, "env", "", "Target environment (default: development)")
	rootCmd.PersistentFlags().BoolVar(&output.JSONMode, "json", false, "Output in JSON format")
}
