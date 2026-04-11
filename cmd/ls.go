package cmd

import (
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
		if os.IsNotExist(err) {
			fmt.Println("⛴  ferry ls\n\n  no ferry.lock found — run: ferry init")
			return nil
		}
		return err
	}

	if interactive {
		result, err := tui.RunProfileManager(lf, false)
		if err != nil {
			return err
		}
		_ = result
		return nil
	}

	// Static table output
	table, err := tui.RenderProfileTable(lf)
	if err != nil {
		return err
	}
	fmt.Print(table)
	return nil
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
