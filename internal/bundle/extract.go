package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cp "github.com/otiai10/copy"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// Note: component extraction failures are non-fatal — a missing component is
// skipped and the bundle is built without it. Failures are logged to stderr.

// componentSpec describes a single extractable component from a build container.
type componentSpec struct {
	id            string
	containerPath string
	installPath   string
	binSymlink    string
	version       string
	localPath     string // if set, copy from local filesystem instead of docker cp
	preserve      bool   // if true, install script skips restore when dest already exists
}

// buildComponentSpecs assembles the full list of components to extract for a build.
func buildComponentSpecs(track BuildTrack, lock *config.LockFile, profile string, langs []registry.ResolvedLanguage) []componentSpec {
	prof := lock.Profiles[profile]
	var specs []componentSpec

	// Nvim — gated on IncludeNvim (nil = unset = true).
	if prof.NvimEnabled() {
		specs = append(specs,
			componentSpec{id: "nvim-binary", containerPath: "/opt/nvim/", installPath: "~/.local/share/nvim-dist/", binSymlink: "~/.local/bin/nvim"},
			componentSpec{id: "treesitter/parsers", containerPath: "/root/.local/share/nvim/lazy/nvim-treesitter/parser/", installPath: "~/.local/share/nvim/lazy/nvim-treesitter/parser/"},
		)
		for _, plugin := range prof.Plugins {
			specs = append(specs, componentSpec{
				id:            "lazy/" + plugin,
				containerPath: "/root/.local/share/nvim/lazy/" + plugin + "/",
				installPath:   "~/.local/share/nvim/lazy/" + plugin + "/",
			})
		}
	}

	// Shell — framework dir + dotfiles, always from local filesystem.
	if sp := prof.Shell; sp != nil {
		if sp.Framework != "" && sp.FrameworkPath != "" {
			fp := config.ExpandHome(sp.FrameworkPath)
			specs = append(specs, componentSpec{
				id:          "shell/framework",
				installPath: sp.FrameworkPath + "/",
				localPath:   fp,
			})
		}
		if sp.RCPath != "" {
			rc := config.ExpandHome(sp.RCPath)
			specs = append(specs, componentSpec{
				id:          "shell/rc",
				installPath: sp.RCPath,
				localPath:   rc,
				preserve:    true,
			})
		}
		if sp.ThemeConfigPath != "" {
			tc := config.ExpandHome(sp.ThemeConfigPath)
			specs = append(specs, componentSpec{
				id:          "shell/theme-config",
				installPath: sp.ThemeConfigPath,
				localPath:   tc,
				preserve:    true,
			})
		}
	}

	// Language runtimes.
	for _, rl := range langs {
		if rl.Runtime == nil {
			continue
		}
		version := rl.Runtime.DefaultVersion
		for _, cp := range rl.Runtime.ContainerPaths {
			containerPath := substituteVars(cp.Container, version, track.Arch)
			installPath := substituteVars(cp.InstallPath, version, track.Arch)
			// Derive a stable component ID from the path, trimming trailing slash
			// before taking the base name so /opt/jdtls/ → "jdtls", not "".
			trimmed := strings.TrimRight(containerPath, "/")
			base := trimmed[strings.LastIndex(trimmed, "/")+1:]
			specs = append(specs, componentSpec{
				id:            "runtime/" + rl.Language.Name + "/" + base,
				containerPath: containerPath,
				installPath:   installPath,
				version:       version,
			})
		}
	}

	// CLI tools.
	for _, name := range flattenCLI(lock, profile) {
		specs = append(specs, componentSpec{
			id:            "cli/" + name,
			containerPath: "/usr/local/bin/" + name,
			installPath:   "~/.local/bin/" + name,
		})
	}
	return specs
}

// ExtractComponents copies component directories from a running container,
// hashes them, compresses them, and stores them in the component store.
func ExtractComponents(containerID string, track BuildTrack, lock *config.LockFile, profile string, langs []registry.ResolvedLanguage) ([]store.Component, error) {
	s, err := store.NewStore()
	if err != nil {
		return nil, err
	}

	var components []store.Component
	for _, spec := range buildComponentSpecs(track, lock, profile, langs) {
		tmpDir := fmt.Sprintf("/tmp/ferry-extract-%s-%s", containerID[:8], sanitizeID(spec.id))

		if spec.localPath != "" {
			// Copy from local filesystem (shell dotfiles/framework, not from container).
			if err := copyLocalToTmp(spec.localPath, tmpDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping component %s: local copy failed: %v\n", spec.id, err)
				os.RemoveAll(tmpDir)
				continue
			}
		} else {
			// Copy from running container.
			isSingleFile := !strings.HasSuffix(spec.containerPath, "/")
			if isSingleFile {
				os.MkdirAll(tmpDir, 0755)
			}
			cpDest := tmpDir
			if isSingleFile {
				cpDest = tmpDir + "/"
			}
			cpCmd := exec.Command("docker", "cp", containerID+":"+spec.containerPath, cpDest)
			if out, err := cpCmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping component %s: docker cp failed: %s\n", spec.id, strings.TrimSpace(string(out)))
				os.RemoveAll(tmpDir)
				continue
			}
		}

		compressed, err := store.CompressDir(tmpDir, lock.Bundle.Exclude)
		os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping component %s: compress failed: %v\n", spec.id, err)
			continue
		}

		hash := store.HashBytes(compressed)
		if err := s.Put(hash, compressed); err != nil {
			return nil, fmt.Errorf("storing component %s: %w", spec.id, err)
		}

		components = append(components, store.Component{
			ID:             spec.id,
			Version:        spec.version,
			Hash:           hash,
			SizeCompressed: int64(len(compressed)),
			InstallPath:    spec.installPath,
			BinSymlink:     spec.binSymlink,
			ArchSpecific:   true,
			Preserve:       spec.preserve,
		})
	}
	return components, nil
}

// CreateContainer creates a stopped container from an image and returns its ID.
func CreateContainer(imageID string) (string, error) {
	out, err := exec.Command("docker", "create", imageID).Output()
	if err != nil {
		return "", fmt.Errorf("docker create: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RemoveContainer removes a container by ID.
func RemoveContainer(containerID string) {
	exec.Command("docker", "rm", containerID).Run()
}

// copyLocalToTmp copies a local file or directory into tmpDir.
func copyLocalToTmp(src, tmpDir string) error {
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	dst := filepath.Join(tmpDir, filepath.Base(src))
	return cp.Copy(src, dst)
}

func sanitizeID(id string) string {
	return strings.NewReplacer("/", "-", " ", "-").Replace(id)
}

// BuildState holds the progress state of a single build track.
type BuildState struct {
	Track    BuildTrack
	Status   string // queued | building | complete | failed
	Duration time.Duration
	LogPath  string
	Error    error
}

// BuildResult is the outcome of building a single track.
type BuildResult struct {
	Track    BuildTrack
	Manifest *store.Manifest
	Duration time.Duration
	Cached   bool
	Error    error
}

// DockerAvailable returns nil if Docker is accessible and running.
func DockerAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH")
	}
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker not running: %s", out)
	}
	return nil
}

