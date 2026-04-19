package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:          "ferry",
	Short:        "Ship your dev environment to remote machines",
	Version:      version,
	SilenceUsage: true,
}

func Execute() {
	// SilenceErrors prevents Cobra from printing the error itself; we rely on
	// each RunE function to print user-friendly output before returning.
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(cleanCmd)
}
