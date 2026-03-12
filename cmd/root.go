package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "envsync",
	Short: "Encrypted .env sync for dev teams",
	Long:  "envsync encrypts .env files per-value using age encryption and syncs them through git using GitHub SSH keys for identity.",
}

func Execute() error {
	return rootCmd.Execute()
}
