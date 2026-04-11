package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
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

	// TODO(Task 6): rewrite to use tui.RunPresetPicker + tui.RunProfileWizard
	_ = existing
	fmt.Println("  ferry init: use 'ferry init' after Task 6 rewrites cmd/init.go")
	return nil
}
