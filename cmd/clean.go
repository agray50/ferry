package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/ssh"
	"github.com/anthropics/ferry/internal/store"
)

var cleanCmd = &cobra.Command{
	Use:   "clean <target>",
	Short: "Remove ferry installation from a target",
	Args:  cobra.ExactArgs(1),
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().Bool("full", false, "Full teardown: remove all ferry data and installed environment")
}

func runClean(cmd *cobra.Command, args []string) error {
	target := args[0]
	full, _ := cmd.Flags().GetBool("full")

	fmt.Printf("⛴  ferry clean %s\n\n", target)

	c, err := ssh.Connect(target)
	if err != nil {
		return err
	}
	defer c.Close()

	if !full {
		return runDefaultClean(c, target)
	}
	return runFullClean(c, target)
}

func runDefaultClean(c *ssh.Client, target string) error {
	// measure store size
	storeSize, _ := remoteSize(c, "$HOME/.ferry/store")

	_, _, _, _ = c.Run("rm -rf $HOME/.ferry/store $HOME/.ferry/incoming")

	fmt.Printf("  removed ~/.ferry/store/ (%s)\n", formatBytes(storeSize))
	fmt.Printf("  removed ~/.ferry/incoming/ (if present)\n\n")
	fmt.Printf("  installed environment preserved\n")
	fmt.Printf("  note: ferry update will re-ship components if needed\n")
	return nil
}

func runFullClean(c *ssh.Client, target string) error {
	// read remote manifest to know which binaries to remove
	manifestData, err := c.DownloadBytes("$HOME/.ferry/manifest.json")
	var manifest *store.Manifest
	if err == nil {
		var m store.Manifest
		if json.Unmarshal(manifestData, &m) == nil {
			manifest = &m
		}
	}

	fmt.Printf("  ⚠ this will remove the entire ferry environment from %s:\n", target)
	fmt.Printf("    ~/.ferry/\n")
	fmt.Printf("    ~/.local/bin/nvim (and all ferry-installed binaries)\n")
	fmt.Printf("    ~/.local/share/nvim/\n")
	fmt.Printf("    ~/.local/share/nvim-dist/\n")
	fmt.Printf("    ~/.local/share/pyenv/\n")
	fmt.Printf("    ~/.local/share/nvm/ (if present)\n")
	fmt.Printf("    ~/.config/nvim/\n")
	fmt.Printf("    ~/.zshrc  (backup will be made first)\n")
	fmt.Printf("    ~/.p10k.zsh\n\n")
	fmt.Printf("  this cannot be undone.\n")
	fmt.Printf("  type \"yes\" to confirm: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != "yes" {
		fmt.Println("  aborted")
		return nil
	}

	// backup shell config
	c.Run("cp ~/.zshrc ~/.zshrc.ferry-backup-$(date +%s) 2>/dev/null || true")
	c.Run("cp ~/.p10k.zsh ~/.p10k.zsh.ferry-backup-$(date +%s) 2>/dev/null || true")
	fmt.Printf("  backed up shell configs\n")

	// remove ferry-managed directories
	for _, dir := range []string{
		"$HOME/.ferry",
		"$HOME/.config/nvim",
		"$HOME/.local/share/nvim",
		"$HOME/.local/share/nvim-dist",
		"$HOME/.pyenv",
		"$HOME/.nvm",
	} {
		c.Run(fmt.Sprintf("rm -rf %s 2>/dev/null || true", dir))
		fmt.Printf("  removed %s\n", dir)
	}

	// remove only known ferry-installed binaries
	if manifest != nil {
		var removed int
		for _, comp := range manifest.Components {
			if strings.HasPrefix(comp.ID, "cli/") {
				binary := strings.TrimPrefix(comp.ID, "cli/")
				c.Run(fmt.Sprintf("rm -f $HOME/.local/bin/%s 2>/dev/null || true", binary))
				removed++
			}
		}
		c.Run("rm -f $HOME/.local/bin/nvim 2>/dev/null || true")
		removed++
		fmt.Printf("  removed %d ferry-installed binaries from ~/.local/bin/\n", removed)
	} else {
		fmt.Printf("  ⚠ could not read manifest — skipping binary cleanup\n")
		fmt.Printf("    check ~/.local/bin/ manually\n")
	}

	// clean up shell files
	c.Run("rm -f ~/.zshrc ~/.p10k.zsh 2>/dev/null || true")

	// remove from targets.json
	if err := config.RemoveTarget(target); err != nil {
		fmt.Printf("  ⚠ could not remove target from registry: %v\n", err)
	} else {
		fmt.Printf("\n  target removed from ferry registry\n")
	}

	return nil
}

func remoteSize(c *ssh.Client, path string) (int64, error) {
	// du -sk is POSIX (kilobytes); -sb (bytes) is GNU-only and fails on Alpine/BusyBox.
	stdout, _, code, err := c.Run(fmt.Sprintf("du -sk %s 2>/dev/null | awk '{print $1}'", path))
	if err != nil || code != 0 {
		return 0, fmt.Errorf("could not measure size")
	}
	var kb int64
	fmt.Sscanf(strings.TrimSpace(stdout), "%d", &kb)
	return kb * 1024, nil
}
