package bundle

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// BuildOptions controls ferry bundle behaviour.
type BuildOptions struct {
	Arch    string
	OS      string
	Profile string
	Force   bool
	Lock    *config.LockFile
}

// BuildAll builds all selected tracks in parallel.
func BuildAll(opts BuildOptions, progress func(state []BuildState)) ([]BuildResult, error) {
	tracks := FilterTracks(opts.Arch, opts.OS)
	if len(tracks) == 0 {
		return nil, fmt.Errorf("no tracks match arch=%s os=%s", opts.Arch, opts.OS)
	}

	lockHash, err := config.LockFileHash()
	if err != nil {
		return nil, err
	}

	// TODO: rewrite in Phase 3 to resolve per-profile languages
	var profileLangs []config.LanguageConfig
	if prof, ok := opts.Lock.Profiles[opts.Profile]; ok {
		profileLangs = prof.Languages
	}
	langs, err := registry.ResolveLanguages(profileLangs)
	if err != nil {
		return nil, err
	}

	states := make([]BuildState, len(tracks))
	for i, t := range tracks {
		states[i] = BuildState{Track: t, Status: "queued"}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]BuildResult, len(tracks))

	for i, track := range tracks {
		wg.Add(1)
		go func(idx int, t BuildTrack) {
			defer wg.Done()

			mu.Lock()
			states[idx].Status = "building"
			if progress != nil {
				progress(states)
			}
			mu.Unlock()

			start := time.Now()
			result := buildTrack(t, opts, langs, lockHash)
			result.Duration = time.Since(start)

			mu.Lock()
			results[idx] = result
			if result.Error != nil {
				states[idx].Status = "failed"
			} else {
				states[idx].Status = "complete"
			}
			states[idx].Duration = result.Duration
			if progress != nil {
				progress(states)
			}
			mu.Unlock()
		}(i, track)
	}

	wg.Wait()
	return results, nil
}

func buildTrack(track BuildTrack, opts BuildOptions, langs []registry.Language, lockHash string) BuildResult {
	r := BuildResult{Track: track}

	// check if already built
	if !opts.Force {
		if existing, err := store.FindManifest(opts.Profile, track.Arch, track.Libc); err == nil {
			if existing.LockfileHash == lockHash {
				r.Manifest = existing
				return r
			}
		}
	}

	// create temp build dir
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("ferry-build-%s-%s-*", track.Arch, track.Libc))
	if err != nil {
		r.Error = err
		return r
	}
	defer os.RemoveAll(tmpDir)

	// generate Dockerfile
	dockerfile, err := GenerateDockerfile(track, opts.Lock, langs)
	if err != nil {
		r.Error = err
		return r
	}

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		r.Error = err
		return r
	}

	// copy nvim config into build context
	home, _ := os.UserHomeDir()
	nvimSrc := filepath.Join(home, ".config", "nvim")
	nvimDst := filepath.Join(tmpDir, "nvim-config")
	if err := copyDir(nvimSrc, nvimDst); err != nil {
		// not fatal — nvim config may not exist
		os.MkdirAll(nvimDst, 0755)
	}

	// set up log file
	if err := config.EnsureFerryDir(); err != nil {
		r.Error = err
		return r
	}
	logPath := filepath.Join(config.FerryDir(), "logs",
		fmt.Sprintf("bundle-%s-%s.log", track.Arch, track.Libc))
	logFile, err := os.Create(logPath)
	if err != nil {
		r.Error = err
		return r
	}
	defer logFile.Close()

	imageTag := fmt.Sprintf("ferry-bundle-%s-%s:latest", track.Arch, track.Libc)

	// docker buildx build
	cmd := exec.Command("docker", "buildx", "build",
		"--platform", track.Platform,
		"--tag", imageTag,
		"--load",
		"--file", dockerfilePath,
		tmpDir,
	)
	cmd.Stdout = io.MultiWriter(logFile)
	cmd.Stderr = io.MultiWriter(logFile)

	if err := cmd.Run(); err != nil {
		r.Error = fmt.Errorf("docker build failed — see %s", logPath)
		return r
	}

	// create container for extraction
	containerID, err := CreateContainer(imageTag)
	if err != nil {
		r.Error = err
		return r
	}
	defer RemoveContainer(containerID)

	// extract components
	components, err := ExtractComponents(containerID, track, opts.Lock, opts.Profile)
	if err != nil {
		r.Error = err
		return r
	}

	// write manifest
	m := &store.Manifest{
		Profile:      opts.Profile,
		Arch:         track.Arch,
		Libc:         track.Libc,
		LockfileHash: lockHash,
		BuiltAt:      time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Components:   components,
	}

	if err := store.WriteManifest(m); err != nil {
		r.Error = err
		return r
	}

	r.Manifest = m
	return r
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
