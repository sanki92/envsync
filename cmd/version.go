package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print envsync version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("envsync %s (%s, %s/%s)\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
