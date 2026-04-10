package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:     "ferry",
	Short:   "Ship your dev environment to remote machines",
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(cleanCmd)
}
