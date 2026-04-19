package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/bootstrap"
	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/format"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/ssh"
	"github.com/anthropics/ferry/internal/store"
	"github.com/anthropics/ferry/internal/tui"
	"github.com/anthropics/ferry/internal/update"
)

var updateCmd = &cobra.Command{
	Use:   "update <target>",
	Short: "Send delta updates to a bootstrapped target",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().Bool("force", false, "Force update even if already up to date")
	updateCmd.Flags().Bool("dry-run", false, "Show what would be updated without applying")
	updateCmd.Flags().String("profile", "", "Switch to a different profile")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	target := args[0]
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	newProfile, _ := cmd.Flags().GetString("profile")

	// 1. read target from targets.json
	t, err := config.GetTarget(target)
	if err != nil {
		return err
	}
	profile := t.Profile
	if newProfile != "" {
		profile = newProfile
	}

	// 2. check if lockfile hash changed
	localHash, err := config.LockFileHash()
	if err != nil {
		return err
	}

	if !force && localHash == t.BundleHash && newProfile == "" {
		fmt.Printf("⛴  %s is already up to date\n", target)
		return nil
	}

	// 3. find local manifest
	localManifest, err := store.FindManifest(profile, t.Arch, t.Libc)
	if err != nil {
		return fmt.Errorf("no bundle found: %w\n  run: ferry bundle --arch %s --libc %s", err, t.Arch, t.Libc)
	}

	// 4. connect and download remote manifest
	fmt.Printf("⛴  ferry update  %s\n\n", target)
	c, err := ssh.Connect(target)
	if err != nil {
		return err
	}
	defer c.Close()

	remoteData, err := c.DownloadBytes("$HOME/.ferry/manifest.json")
	if err != nil {
		return fmt.Errorf("no remote manifest found — run ferry bootstrap first")
	}
	var remoteManifest store.Manifest
	if err := json.Unmarshal(remoteData, &remoteManifest); err != nil {
		return fmt.Errorf("parsing remote manifest: %w", err)
	}

	// 5. diff
	diff := store.DiffManifests(localManifest, &remoteManifest)

	if len(diff.New) == 0 && len(diff.Changed) == 0 && len(diff.Removed) == 0 && !force {
		fmt.Printf("  nothing to update\n")
		return nil
	}

	// 6. check for active nvim
	stdout, _, _, _ := c.Run("pgrep -x nvim 2>/dev/null")
	if strings.TrimSpace(stdout) != "" {
		pids := strings.Fields(strings.TrimSpace(stdout))
		fmt.Printf("  ⚠ nvim is currently running on %s (%d process(es): %s)\n", target, len(pids), strings.Join(pids, ", "))
		fmt.Printf("    updating may cause instability in the active session\n\n")
		fmt.Printf("    [1] wait for session to end (checks every 30s)\n")
		fmt.Printf("    [2] force update anyway\n")
		fmt.Printf("    [3] cancel\n")
		fmt.Printf("  > ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())
		switch choice {
		case "1":
			fmt.Printf("  waiting for nvim to exit")
			for {
				time.Sleep(30 * time.Second)
				stdout, _, _, _ := c.Run("pgrep -x nvim 2>/dev/null")
				if strings.TrimSpace(stdout) == "" {
					fmt.Printf("\n  nvim exited — proceeding\n")
					break
				}
				fmt.Printf(".")
			}
		case "2":
			// proceed
		default:
			fmt.Println("  cancelled")
			return nil
		}
	}

	// 7. check update lock
	if _, exists, _ := update.LockInfo(c); exists {
		fmt.Printf("  ⚠ update lock found on %s\n  another update may be in progress\n", target)
		fmt.Printf("  if stale: ssh %s \"rm ~/.ferry/.update.lock\"\n\n", target)
		fmt.Printf("  [1] wait 30s and retry\n  [2] cancel\n  > ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(scanner.Text()) != "1" {
			return nil
		}
		time.Sleep(30 * time.Second)
	}

	// 8. dry run output
	if dryRun {
		fmt.Printf("  would ship %s (%d components):\n", format.Bytes(diff.DiffSize()), len(diff.New)+len(diff.Changed))
		for _, c := range diff.New {
			fmt.Printf("    ↑ %-30s new     %s\n", c.ID, format.Bytes(c.SizeCompressed))
		}
		for _, c := range diff.Changed {
			fmt.Printf("    ↑ %-30s updated %s\n", c.ID, format.Bytes(c.SizeCompressed))
		}
		if len(diff.Removed) > 0 {
			fmt.Printf("\n  would remove:\n")
			for _, c := range diff.Removed {
				fmt.Printf("    ✗ %s\n", c.ID)
			}
		}
		fmt.Printf("\n  (dry run — nothing shipped)\n")
		return nil
	}

	// 9. interactive review TUI
	reviewModel := tui.NewUpdateReview(target, diff)
	p := tea.NewProgram(reviewModel)
	final, err := p.Run()
	if err != nil {
		return err
	}
	rm := final.(tui.UpdateReviewModel)
	if rm.Aborted {
		fmt.Println("  cancelled")
		return nil
	}
	selected := rm.SelectedComponents()

	// Build a filtered diff containing only the selected components so the
	// apply script doesn't reference archives that were never uploaded.
	// Removals are always applied regardless of TUI selection.
	selectedHashes := make(map[string]bool, len(selected))
	for _, c := range selected {
		selectedHashes[c.Hash] = true
	}
	applyDiff := store.DiffResult{Removed: diff.Removed}
	for _, c := range diff.New {
		if selectedHashes[c.Hash] {
			applyDiff.New = append(applyDiff.New, c)
		}
	}
	for _, c := range diff.Changed {
		if selectedHashes[c.Hash] {
			applyDiff.Changed = append(applyDiff.Changed, c)
		}
	}

	// 10. acquire lock
	if err := update.AcquireLock(c); err != nil {
		return err
	}
	defer update.ReleaseLock(c)

	// 11. stream selected components
	s, err := store.NewStore()
	if err != nil {
		return err
	}

	var totalSize int64
	for _, comp := range selected {
		totalSize += comp.SizeCompressed
	}

	fmt.Printf("\n  shipping %d components (%s)...\n", len(selected), format.Bytes(totalSize))
	if err := c.MkdirAll("$HOME/.ferry/store"); err != nil {
		return err
	}

	var shipped int64
	for _, comp := range selected {
		data, err := s.Get(comp.Hash)
		if err != nil {
			return fmt.Errorf("reading %s: %w", comp.ID, err)
		}
		remotePath := fmt.Sprintf("$HOME/.ferry/store/%s.tar.zst", comp.Hash)
		var prev int64
		if err := c.StreamUpload(data, remotePath, func(w int64) {
			shipped += w - prev
			prev = w
			pct := float64(shipped) / float64(totalSize) * 100
			fmt.Printf("\r  [%3.0f%%] %s", pct, format.Bytes(shipped))
		}); err != nil {
			return err
		}
	}
	fmt.Println()

	// upload env.sh and manifest to incoming/ — apply script copies both into place
	if err := c.MkdirAll("$HOME/.ferry/incoming"); err != nil {
		return err
	}

	lock, err := config.ReadLockFile()
	if err != nil {
		return err
	}
	tools, err := config.LoadToolsFile()
	if err != nil {
		return fmt.Errorf("loading tools file: %w", err)
	}
	var langs []registry.ResolvedLanguage
	if prof, ok := lock.Profiles[profile]; ok && len(prof.Languages) > 0 {
		if resolved, resolveErr := registry.ResolveFromProfile(prof.Languages, tools); resolveErr == nil {
			langs = resolved
		}
	}
	envSh := bootstrap.GenerateEnvSh(langs)
	if err := c.UploadBytes([]byte(envSh), "$HOME/.ferry/incoming/env.sh", 0644); err != nil {
		return fmt.Errorf("uploading env.sh: %w", err)
	}

	manifestData, err := json.MarshalIndent(localManifest, "", "  ")
	if err != nil {
		return err
	}
	if err := c.UploadBytes(manifestData, "$HOME/.ferry/incoming/manifest.json", 0600); err != nil {
		return err
	}

	// 12. apply on target
	fmt.Printf("\n  applying on target...\n")
	applyScript := update.GenerateApplyScript(applyDiff)
	_, stderr, code, err := c.Run(applyScript)
	if err != nil || code != 0 {
		return fmt.Errorf("apply failed (exit %d): %s", code, stderr)
	}

	fmt.Printf("  ✓ updated %d components\n", len(selected))
	if len(diff.Removed) > 0 {
		fmt.Printf("  ✓ removed %d components\n", len(diff.Removed))
	}

	// 13. update targets.json
	t.BundleHash = localHash
	t.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if newProfile != "" {
		t.Profile = newProfile
	}
	if err := config.UpsertTarget(target, t); err != nil {
		return err
	}

	fmt.Printf("\n  reconnect to pick up changes\n")
	return nil
}
