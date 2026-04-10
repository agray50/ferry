package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/tui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively discover and configure your environment",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("update", false, "Update an existing ferry.lock")
}

func runInit(cmd *cobra.Command, args []string) error {
	update, _ := cmd.Flags().GetBool("update")

	var existing *config.LockFile
	if update {
		lf, err := config.ReadLockFile()
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("--update: %w", err)
			}
		} else {
			existing = lf
		}
	}

	result, err := tui.RunInitWizard(existing)
	if err != nil {
		return err
	}
	if result.Aborted {
		fmt.Println("  aborted — ferry.lock not written")
		return nil
	}

	if err := config.WriteLockFile(result.LockFile); err != nil {
		return err
	}

	fmt.Println("✓ wrote ferry.lock")
	fmt.Println("  run ferry bundle --arch all to build bundles")
	return nil
}
