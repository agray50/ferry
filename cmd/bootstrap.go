package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/bootstrap"
	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/format"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/ssh"
	"github.com/anthropics/ferry/internal/store"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap <target>",
	Short: "Bootstrap a remote target with the bundle",
	Args:  cobra.ExactArgs(1),
	RunE:  runBootstrap,
}

func init() {
	bootstrapCmd.Flags().String("profile", "default", "Profile name to bootstrap")
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	target := args[0]
	profile, _ := cmd.Flags().GetString("profile")

	fmt.Printf("⛴  ferry bootstrap %s\n\n", target)

	// SSH connection
	c, err := ssh.Connect(target)
	if err != nil {
		fmt.Printf("  ✗ SSH connection to %s failed: %v\n", target, err)
		return err
	}
	defer c.Close()
	fmt.Printf("  ✓ SSH connection to %s\n", target)

	// Detect remote environment
	env, err := ssh.DetectRemoteEnv(c)
	if err != nil {
		return fmt.Errorf("detecting remote environment: %w", err)
	}
	fmt.Printf("  ✓ arch detected: %s\n", env.Arch)
	fmt.Printf("  ✓ libc detected: %s\n", env.Libc)

	// Find manifest
	manifest, err := store.FindManifest(profile, env.Arch, env.Libc)
	if err != nil {
		fmt.Printf("  ✗ bundle not found for profile=%s arch=%s libc=%s\n", profile, env.Arch, env.Libc)
		fmt.Printf("    run: ferry bundle --arch %s --libc %s\n", env.Arch, env.Libc)
		return err
	}

	// Compute bundle size
	var bundleSize int64
	for _, comp := range manifest.Components {
		bundleSize += comp.SizeCompressed
	}

	// Disk space check
	needed := bundleSize * 2
	if env.DiskFree < needed {
		fmt.Printf("  ✗ insufficient disk space: %s available, need %s\n",
			format.Bytes(env.DiskFree), format.Bytes(needed))
		return fmt.Errorf("insufficient disk space")
	}
	fmt.Printf("  ✓ disk space: %s available (need ~%s)\n",
		format.Bytes(env.DiskFree), format.Bytes(needed))

	// Zsh version check — actually compare versions
	lock, err := config.ReadLockFile()
	if err != nil {
		return err
	}
	if env.ZshVersion != "" {
		ok, err := zshVersionOK(env.ZshVersion, lock.Bundle.Requires.MinZshVersion)
		if err == nil && ok {
			fmt.Printf("  ✓ zsh %s found (need >= %s)\n", env.ZshVersion, lock.Bundle.Requires.MinZshVersion)
		} else {
			fmt.Printf("  ✗ zsh %s found but need >= %s\n", env.ZshVersion, lock.Bundle.Requires.MinZshVersion)
			return fmt.Errorf("zsh %s < required %s", env.ZshVersion, lock.Bundle.Requires.MinZshVersion)
		}
	} else {
		fmt.Printf("  ✗ zsh not found (need >= %s)\n", lock.Bundle.Requires.MinZshVersion)
		return fmt.Errorf("zsh not found on target")
	}

	// Required packages
	pkgSet := make(map[string]bool)
	for _, p := range env.InstalledPackages {
		pkgSet[p] = true
	}
	allPkgsOK := true
	for _, req := range lock.Bundle.Requires.Packages {
		if pkgSet[req] {
			fmt.Printf("  ✓ %s found\n", req)
		} else {
			fmt.Printf("  ✗ %s not found\n", req)
			allPkgsOK = false
		}
	}
	if !allPkgsOK {
		return fmt.Errorf("missing required packages on target")
	}

	// Confirm with user
	fmt.Printf("\nready to bootstrap %s\n\n", target)
	fmt.Printf("  profile:  %s\n", profile)
	fmt.Printf("  arch:     %s-%s\n", env.Arch, env.Libc)
	fmt.Printf("  bundle:   %s (%d components)\n", format.Bytes(bundleSize), len(manifest.Components))
	fmt.Printf("\n  estimated transfer time:\n")
	fmt.Printf("    10 Mbps upload:  ~%s\n", estimateTime(bundleSize, 10))
	fmt.Printf("    50 Mbps upload:  ~%s\n", estimateTime(bundleSize, 50))
	fmt.Printf("    100 Mbps upload: ~%s\n", estimateTime(bundleSize, 100))
	fmt.Printf("\n  proceed? [Y/n] ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())
	if answer != "" && strings.ToLower(answer) != "y" {
		fmt.Println("  aborted")
		return nil
	}

	// Create remote ferry dirs
	if err := c.MkdirAll("$HOME/.ferry/store"); err != nil {
		return fmt.Errorf("creating remote ferry dir: %w", err)
	}
	if err := c.MkdirAll("$HOME/.ferry/incoming"); err != nil {
		return fmt.Errorf("creating remote incoming dir: %w", err)
	}

	// Stream components
	s, err := store.NewStore()
	if err != nil {
		return err
	}

	fmt.Printf("\nshipping %d components (%s)...\n", len(manifest.Components), format.Bytes(bundleSize))
	var shipped int64
	for i, comp := range manifest.Components {
		data, err := s.Get(comp.Hash)
		if err != nil {
			return fmt.Errorf("reading component %s: %w", comp.ID, err)
		}
		remotePath := fmt.Sprintf("$HOME/.ferry/store/%s.tar.zst", comp.Hash)
		var prev int64
		err = c.StreamUpload(data, remotePath, func(written int64) {
			shipped += written - prev
			prev = written
			pct := float64(shipped) / float64(bundleSize) * 100
			fmt.Printf("\r  [%3.0f%%] %d/%d components  %s",
				pct, i+1, len(manifest.Components), format.Bytes(shipped))
		})
		if err != nil {
			return fmt.Errorf("uploading component %s: %w", comp.ID, err)
		}
	}
	fmt.Println()

	// Resolve languages for this profile so plugin/ferry.lua and lua/ferry.lua
	// are populated with the correct LSP server and treesitter parser lists,
	// and so env.sh contains the correct runtime PATH entries.
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

	// Generate and upload install.sh
	script, err := bootstrap.GenerateInstallScript(manifest, lock, langs)
	if err != nil {
		return err
	}
	if err := c.UploadBytes([]byte(script), "$HOME/.ferry/install.sh", 0755); err != nil {
		return fmt.Errorf("uploading install.sh: %w", err)
	}

	// Upload env.sh (runtime PATH entries) to incoming/ — install.sh copies it into place.
	envSh := bootstrap.GenerateEnvSh(langs)
	if err := c.UploadBytes([]byte(envSh), "$HOME/.ferry/incoming/env.sh", 0644); err != nil {
		return fmt.Errorf("uploading env.sh: %w", err)
	}

	// Upload manifest to incoming/
	manifestData, err := manifestJSON(manifest)
	if err != nil {
		return err
	}
	if err := c.UploadBytes(manifestData, "$HOME/.ferry/incoming/manifest.json", 0600); err != nil {
		return fmt.Errorf("uploading manifest: %w", err)
	}

	fmt.Printf("\nrunning install.sh on target...\n")
	stdout, stderr, code, err := c.RunWithStdin("sh $HOME/.ferry/install.sh", []byte("\n"))
	if err != nil || code != 0 {
		fmt.Printf("  install.sh failed (exit %d):\n%s\n%s\n", code, stdout, stderr)
		return fmt.Errorf("install.sh failed")
	}
	fmt.Print(stdout)

	// Validation
	fmt.Println()
	result := bootstrap.Validate(c, manifest)
	bootstrap.PrintValidation(result)

	if result.Errors > 0 {
		return fmt.Errorf("bootstrap validation failed with %d error(s)", result.Errors)
	}

	// Update targets.json
	if err := config.UpsertTarget(target, config.Target{
		Arch:           env.Arch,
		Libc:           env.Libc,
		Profile:        profile,
		BundleHash:     manifest.LockfileHash,
		BootstrappedAt: time.Now().UTC().Format(time.RFC3339),
		LastUpdated:    time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	fmt.Printf("\nbootstrap complete ✓\n")
	fmt.Printf("connect with: ssh %s\n", target)
	return nil
}

// zshVersionOK returns true if have >= need (e.g. "5.9" >= "5.8").
func zshVersionOK(have, need string) (bool, error) {
	h, err := semver.NewVersion(have)
	if err != nil {
		return false, fmt.Errorf("unparseable zsh version %q: %w", have, err)
	}
	n, err := semver.NewVersion(need)
	if err != nil {
		return false, fmt.Errorf("unparseable required version %q: %w", need, err)
	}
	return h.GreaterThanEqual(n), nil
}


func estimateTime(bytes int64, mbps int) string {
	// mbps is megabits per second; 1 Mbps = 1,000,000 bits/s
	bitsPerSec := int64(mbps) * 1_000_000
	secs := (bytes * 8) / bitsPerSec
	d := time.Duration(secs) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func manifestJSON(m *store.Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
