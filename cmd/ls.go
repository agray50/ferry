package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/tui"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List profiles and their deployment status",
	RunE:  runLs,
}

func init() {
	lsCmd.Flags().Bool("interactive", false, "Open interactive profile manager (same as ferry init)")
}

func runLs(cmd *cobra.Command, args []string) error {
	interactive, _ := cmd.Flags().GetBool("interactive")

	lf, err := config.ReadLockFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("⛴  ferry ls\n\n  no ferry.lock found — run: ferry init")
			return nil
		}
		return err
	}

	if interactive {
		return runProfileManagerLoop(lf)
	}

	// Static table output
	table, err := tui.RenderProfileTable(lf)
	if err != nil {
		return err
	}
	fmt.Print(table)
	return nil
}
