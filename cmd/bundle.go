package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/bundle"
	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/format"
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

	// Pre-flight: ferry.lock
	lock, err := config.ReadLockFile()
	if err != nil {
		return fmt.Errorf("ferry.lock not found — run: ferry init\n  %w", err)
	}

	tools, err := config.LoadToolsFile()
	if err != nil {
		return fmt.Errorf("loading tools file: %w", err)
	}

	tracks := bundle.FilterTracks(arch, osFlag)

	// Pre-flight: docker — only required when building linux tracks
	needsDocker := false
	for _, t := range tracks {
		if t.BuildMethod == "docker" {
			needsDocker = true
			break
		}
	}
	if needsDocker {
		if err := bundle.DockerAvailable(); err != nil {
			fmt.Printf("✗ Docker not found. Docker is required to build Linux bundles.\n")
			fmt.Printf("  Install Docker Desktop: https://docs.docker.com/get-docker/\n")
			return err
		}
	}
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
		Tools:   tools,
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
			fmt.Printf("  %-20s %-10s %s\n", track, "FAILED", r.Duration.Round(time.Second))
			failures = append(failures, r)
		} else if r.Manifest != nil {
			var size int64
			for _, c := range r.Manifest.Components {
				size += c.SizeCompressed
			}
			label := format.Bytes(size)
			if r.Cached {
				label += " (cached)"
			}
			fmt.Printf("  %-20s %-16s %s\n", track, label, r.Duration.Round(time.Second))
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

	fmt.Printf("\n  to deploy, run:\n")
	fmt.Printf("    ferry bootstrap <user@host>   # first time on a new target\n")
	fmt.Printf("    ferry update <target>         # update an already-bootstrapped target\n")
	return nil
}

// progressPrinter re-renders a fixed block of lines in place using ANSI cursor-up.
type progressPrinter struct {
	lineCount int
}

var dockerStepRe = regexp.MustCompile(`\[(\d+)/(\d+)\]`)

// dockerProgress reads the tail of the build log and returns (currentStep, totalSteps, label).
// It reads only the last 4KB to avoid scanning the entire log on every tick.
func dockerProgress(logPath string) (cur, total int, label string) {
	f, err := os.Open(logPath)
	if err != nil {
		return 0, 0, ""
	}
	defer f.Close()

	const tailSize = 4096
	if fi, err := f.Stat(); err == nil && fi.Size() > tailSize {
		f.Seek(-tailSize, io.SeekEnd)
	}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if m := dockerStepRe.FindStringSubmatch(line); m != nil {
			fmt.Sscanf(m[1], "%d", &cur)
			fmt.Sscanf(m[2], "%d", &total)
		}
	}
	if total > 0 {
		label = fmt.Sprintf("step %d/%d", cur, total)
	}
	return
}

func (pp *progressPrinter) render(states []bundle.BuildState) {
	// Move cursor up to overwrite the previously printed lines.
	if pp.lineCount > 0 {
		fmt.Printf("\033[%dA", pp.lineCount)
	}
	for _, s := range states {
		track := fmt.Sprintf("%s-%s", s.Track.Arch, s.Track.OS)
		dur := ""
		if s.Duration > 0 {
			dur = s.Duration.Round(time.Second).String()
		}
		label := s.Status
		cur, total := 0, 0
		if s.Status == "building" && s.LogPath != "" {
			var step string
			cur, total, step = dockerProgress(s.LogPath)
			if step != "" {
				label = step
			}
		}
		bar := progressBar(s.Status, cur, total)
		// \033[2K clears the current line before printing.
		fmt.Printf("\033[2K  %-20s %s %-16s %s\n", track, bar, label, dur)
	}
}

func progressBar(status string, cur, total int) string {
	const width = 20
	switch status {
	case "complete", "cached":
		return strings.Repeat("█", width)
	case "building":
		if total > 0 {
			filled := cur * width / total
			if filled < 1 {
				filled = 1
			}
			return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		}
		return strings.Repeat("█", width/2) + strings.Repeat("░", width/2)
	default:
		return strings.Repeat("░", width)
	}
}

