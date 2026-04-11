package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/bundle"
	"github.com/anthropics/ferry/internal/config"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Build component bundles using Docker",
	RunE:  runBundle,
}

func init() {
	bundleCmd.Flags().String("arch", "all", "Target architecture: x86_64, arm64, or all")
	bundleCmd.Flags().String("os", "all", "target OS: linux|darwin|all")
	bundleCmd.Flags().String("profile", "default", "Profile name to build")
	bundleCmd.Flags().Bool("force", false, "Force rebuild even if bundle is up to date")
}

func runBundle(cmd *cobra.Command, args []string) error {
	arch, _ := cmd.Flags().GetString("arch")
	osFlag, _ := cmd.Flags().GetString("os")
	profile, _ := cmd.Flags().GetString("profile")
	force, _ := cmd.Flags().GetBool("force")

	// Pre-flight: docker
	if err := bundle.DockerAvailable(); err != nil {
		fmt.Printf("✗ Docker not found. Docker is required for ferry bundle.\n")
		fmt.Printf("  Install Docker Desktop: https://docs.docker.com/get-docker/\n")
		return err
	}

	// Pre-flight: ferry.lock
	lock, err := config.ReadLockFile()
	if err != nil {
		return fmt.Errorf("ferry.lock not found — run: ferry init\n  %w", err)
	}

	tracks := bundle.FilterTracks(arch, osFlag)
	fmt.Printf("⛴  ferry bundle\n\n")
	fmt.Printf("building %d tracks...\n\n", len(tracks))

	// Print initial blank lines so the in-place updater has lines to overwrite.
	for range tracks {
		fmt.Println()
	}

	opts := bundle.BuildOptions{
		Arch:    arch,
		OS:      osFlag,
		Profile: profile,
		Force:   force,
		Lock:    lock,
	}

	pp := &progressPrinter{lineCount: len(tracks)}
	results, err := bundle.BuildAll(opts, func(states []bundle.BuildState) {
		pp.render(states)
	})
	if err != nil {
		return err
	}

	fmt.Printf("\n⛴  ferry bundle complete\n\n")
	fmt.Printf("  bundles written to ~/.ferry/bundles/\n\n")
	fmt.Printf("  %-20s %-10s %s\n", "TRACK", "SIZE", "TIME")

	var failures []bundle.BuildResult
	for _, r := range results {
		track := fmt.Sprintf("%s-%s", r.Track.Arch, r.Track.OS)
		if r.Error != nil {
			fmt.Printf("  %-20s FAILED      %s\n", track, r.Duration.Round(time.Second))
			failures = append(failures, r)
		} else if r.Manifest != nil {
			var size int64
			for _, c := range r.Manifest.Components {
				size += c.SizeCompressed
			}
			fmt.Printf("  %-20s %-10s %s\n", track, formatMB(size), r.Duration.Round(time.Second))
		}
	}

	if len(failures) > 0 {
		fmt.Printf("\n  %d track(s) failed:\n", len(failures))
		for _, f := range failures {
			fmt.Printf("  %s-%s: %v\n", f.Track.Arch, f.Track.OS, f.Error)
			fmt.Printf("    see ~/.ferry/logs/bundle-%s-%s.log\n", f.Track.Arch, f.Track.OS)
		}
		return fmt.Errorf("%d track(s) failed", len(failures))
	}

	fmt.Printf("\n  run ferry bootstrap <target> to deploy\n")
	return nil
}

// progressPrinter re-renders a fixed block of lines in place using ANSI cursor-up.
type progressPrinter struct {
	lineCount int
}

func (pp *progressPrinter) render(states []bundle.BuildState) {
	// Move cursor up to overwrite the previously printed lines.
	if pp.lineCount > 0 {
		fmt.Printf("\033[%dA", pp.lineCount)
	}
	for _, s := range states {
		track := fmt.Sprintf("%s-%s", s.Track.Arch, s.Track.OS)
		bar := progressBar(s.Status)
		dur := ""
		if s.Duration > 0 {
			dur = s.Duration.Round(time.Second).String()
		}
		// \033[2K clears the current line before printing.
		fmt.Printf("\033[2K  %-20s %s %-12s %s\n", track, bar, s.Status, dur)
	}
}

func progressBar(status string) string {
	const width = 20
	switch status {
	case "complete":
		return strings.Repeat("█", width)
	case "building":
		return strings.Repeat("█", width/2) + strings.Repeat("░", width/2)
	default:
		return strings.Repeat("░", width)
	}
}

func formatMB(b int64) string {
	return fmt.Sprintf("%.0fMB", float64(b)/1024/1024)
}
