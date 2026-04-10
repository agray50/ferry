package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all registered targets and their status",
	RunE:  runLs,
}

func init() {
	lsCmd.Flags().Bool("verbose", false, "Show detailed target information")
}

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	greyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func runLs(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	targets, err := config.ReadTargets()
	if err != nil {
		return err
	}

	fmt.Printf("⛴  ferry ls\n\n")

	if len(targets) == 0 {
		fmt.Printf("  no targets bootstrapped yet\n")
		fmt.Printf("  run: ferry bootstrap <user@host>\n")
		return nil
	}

	// compute local lockfile hash (best effort)
	localHash, _ := config.LockFileHash()

	isTTY := isTerminal()
	if verbose {
		for host, t := range targets {
			status, style := targetStatus(t.BundleHash, localHash)
			statusStr := status
			if isTTY {
				statusStr = style.Render(status)
			}
			fmt.Printf("%s\n", host)
			fmt.Printf("  arch:          %s (%s)\n", t.Arch, t.Libc)
			fmt.Printf("  profile:       %s\n", t.Profile)
			fmt.Printf("  bundle:        %s\n", t.BundleHash[:8])
			fmt.Printf("  status:        %s\n", statusStr)
			fmt.Printf("  bootstrapped:  %s\n", t.BootstrappedAt)
			fmt.Printf("  last updated:  %s\n", t.LastUpdated)
			if status == "stale" {
				fmt.Printf("  update with:   ferry update %s\n", host)
			}
			fmt.Println()
		}
		return nil
	}

	// table output
	fmt.Printf("  %-25s %-8s %-8s %-10s %-8s %s\n",
		"TARGET", "ARCH", "LIBC", "PROFILE", "BUNDLE", "STATUS")

	for host, t := range targets {
		status, style := targetStatus(t.BundleHash, localHash)
		bundleShort := t.BundleHash
		if len(bundleShort) > 8 {
			bundleShort = bundleShort[:8]
		}
		statusStr := status
		if isTTY {
			statusStr = style.Render(status)
		}
		fmt.Printf("  %-25s %-8s %-8s %-10s %-8s %s\n",
			host, t.Arch, t.Libc, t.Profile, bundleShort, statusStr)
	}

	return nil
}

func targetStatus(bundleHash, localHash string) (string, lipgloss.Style) {
	if localHash == "" {
		return "unknown", greyStyle
	}
	if bundleHash == localHash {
		return "up-to-date", greenStyle
	}
	return "stale", yellowStyle
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
