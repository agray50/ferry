package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// ExtractComponents copies component directories from a running container,
// hashes them, compresses them, and stores them in the component store.
func ExtractComponents(containerID string, track BuildTrack, lock *config.LockFile, profile string, langs []registry.ResolvedLanguage) ([]store.Component, error) {
	s, err := store.NewStore()
	if err != nil {
		return nil, err
	}

	type componentSpec struct {
		id            string
		containerPath string
		installPath   string
		binSymlink    string
		version       string
	}

	var specs []componentSpec

	// nvim binary (always present)
	specs = append(specs, componentSpec{
		id:            "nvim-binary",
		containerPath: "/opt/nvim/",
		installPath:   "~/.local/share/nvim-dist/",
		binSymlink:    "~/.local/bin/nvim",
	})

	// lazy plugins for the named profile
	if prof, ok := lock.Profiles[profile]; ok {
		for _, plugin := range prof.Plugins {
			specs = append(specs, componentSpec{
				id:            "lazy/" + plugin,
				containerPath: "/root/.local/share/nvim/lazy/" + plugin + "/",
				installPath:   "~/.local/share/nvim/lazy/" + plugin + "/",
			})
		}
	}

	// treesitter parsers
	specs = append(specs, componentSpec{
		id:            "treesitter/parsers",
		containerPath: "/root/.local/share/nvim/lazy/nvim-treesitter/parser/",
		installPath:   "~/.local/share/nvim/lazy/nvim-treesitter/parser/",
	})

	// Language runtimes — from registry ContainerPaths
	for _, rl := range langs {
		rt := rl.Runtime
		if rt == nil {
			continue
		}
		version := rt.DefaultVersion
		for _, cp := range rt.ContainerPaths {
			containerPath := substituteVars(cp.Container, version, track.Arch)
			installPath := substituteVars(cp.InstallPath, version, track.Arch)
			specs = append(specs, componentSpec{
				id:            "runtime/" + rl.Language.Name + "/" + containerPath[strings.LastIndex(containerPath, "/")+1:],
				containerPath: containerPath,
				installPath:   installPath,
				version:       version,
			})
		}
	}

	// CLI tools (union across all profiles)
	for _, name := range flattenCLI(lock) {
		specs = append(specs, componentSpec{
			id:            "cli/" + name,
			containerPath: "/usr/local/bin/" + name,
			installPath:   "~/.local/bin/" + name,
		})
	}

	var components []store.Component
	for _, spec := range specs {
		tmpDir := fmt.Sprintf("/tmp/ferry-extract-%s-%s", containerID[:8], sanitizeID(spec.id))

		cpCmd := exec.Command("docker", "cp", containerID+":"+spec.containerPath, tmpDir)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			_ = out
			continue
		}
		defer os.RemoveAll(tmpDir)

		compressed, err := store.CompressDir(tmpDir, lock.Bundle.Exclude)
		if err != nil {
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

func sanitizeID(id string) string {
	return strings.NewReplacer("/", "-", " ", "-").Replace(id)
}

// BuildState holds the progress state of a single build track.
type BuildState struct {
	Track    BuildTrack
	Status   string // queued | building | complete | failed
	Duration time.Duration
	Step     string
	Error    error
}

// BuildResult is the outcome of building a single track.
type BuildResult struct {
	Track    BuildTrack
	Manifest *store.Manifest
	Duration time.Duration
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

// BuildxSupported returns nil if docker buildx is available.
func BuildxSupported() error {
	_, err := exec.Command("docker", "buildx", "ls").Output()
	if err != nil {
		return fmt.Errorf("docker buildx not available")
	}
	return nil
}

// ImageManifest describes a built docker image.
type ImageManifest struct {
	ID       string
	RepoTags []string
	Size     int64
}

// LastBuiltImage returns the ID of the most recently built image with the given tag.
func LastBuiltImage(tag string) (string, error) {
	out, err := exec.Command("docker", "images", tag, "--format", "{{json .}}").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("image %q not found", tag)
	}
	var img ImageManifest
	if err := json.Unmarshal([]byte(lines[0]), &img); err != nil {
		return "", err
	}
	return img.ID, nil
}
