package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	cp "github.com/otiai10/copy"

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
	Tools   *config.ToolsFile
}

// BuildAll builds all selected tracks sequentially, reporting progress after each.
func BuildAll(opts BuildOptions, progress func(state []BuildState)) ([]BuildResult, error) {
	tracks := FilterTracks(opts.Arch, opts.OS)
	if len(tracks) == 0 {
		return nil, fmt.Errorf("no tracks match arch=%s os=%s", opts.Arch, opts.OS)
	}

	lockHash, err := config.LockFileHash()
	if err != nil {
		return nil, err
	}

	var profileLangs []config.LanguageConfig
	if prof, ok := opts.Lock.Profiles[opts.Profile]; ok {
		profileLangs = prof.Languages
	}
	langs, err := registry.ResolveFromProfile(profileLangs, opts.Tools)
	if err != nil {
		return nil, err
	}

	states := make([]BuildState, len(tracks))
	for i, t := range tracks {
		states[i] = BuildState{Track: t, Status: "queued"}
	}

	results := make([]BuildResult, len(tracks))
	for i, track := range tracks {
		logPath := filepath.Join(config.FerryDir(), "logs",
			fmt.Sprintf("bundle-%s-%s.log", track.Arch, track.OS))
		states[i].Status = "building"
		states[i].LogPath = logPath
		if progress != nil {
			progress(states)
		}

		// Ticker: re-render every 2s so step progress stays live.
		done := make(chan struct{})
		start := time.Now()
		if progress != nil {
			go func(idx int, t0 time.Time) {
				tick := time.NewTicker(2 * time.Second)
				defer tick.Stop()
				for {
					select {
					case <-tick.C:
						states[idx].Duration = time.Since(t0)
						progress(states)
					case <-done:
						return
					}
				}
			}(i, start)
		}

		results[i] = buildTrack(track, opts, langs, lockHash)
		close(done)
		results[i].Duration = time.Since(start)
		if results[i].Error != nil {
			states[i].Status = "failed"
		} else if results[i].Cached {
			states[i].Status = "cached"
		} else {
			states[i].Status = "complete"
		}
		states[i].Duration = results[i].Duration
		if progress != nil {
			progress(states)
		}
	}
	return results, nil
}

func buildTrack(track BuildTrack, opts BuildOptions, langs []registry.ResolvedLanguage, lockHash string) BuildResult {
	r := BuildResult{Track: track}

	// check if already built
	if !opts.Force {
		if existing, err := store.FindManifest(opts.Profile, track.Arch, track.Libc); err == nil {
			if existing.LockfileHash == lockHash {
				r.Manifest = existing
				r.Cached = true
				return r
			}
		}
	}

	if track.BuildMethod == "download" {
		return buildMacOSTrack(track, opts, langs, lockHash)
	}

	// create temp build dir
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("ferry-build-%s-%s-*", track.Arch, track.Libc))
	if err != nil {
		r.Error = err
		return r
	}
	defer os.RemoveAll(tmpDir)

	// generate Dockerfile
	dockerfile, err := GenerateDockerfile(track, opts.Lock, opts.Profile, langs, opts.Tools)
	if err != nil {
		r.Error = err
		return r
	}

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		r.Error = err
		return r
	}

	// copy nvim config into build context — resolve symlinks first so a
	// broken or indirect ~/.config/nvim symlink doesn't silently skip config.
	// Missing config is non-fatal but means no lazy plugins will be bundled.
	home, _ := os.UserHomeDir()
	nvimSrc := filepath.Join(home, ".config", "nvim")
	if resolved, err := filepath.EvalSymlinks(nvimSrc); err == nil {
		nvimSrc = resolved
	}
	nvimDst := filepath.Join(tmpDir, "nvim-config")
	if err := copyDir(nvimSrc, nvimDst); err != nil {
		os.MkdirAll(nvimDst, 0755)
		fmt.Fprintf(os.Stderr, "warning: ~/.config/nvim not found — lazy plugins will not be bundled\n")
	}

	// set up log file
	if err := config.EnsureFerryDir(); err != nil {
		r.Error = err
		return r
	}
	logPath := filepath.Join(config.FerryDir(), "logs",
		fmt.Sprintf("bundle-%s-%s.log", track.Arch, track.OS))
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
	cmd.Stdout = logFile
	cmd.Stderr = logFile

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
	components, err := ExtractComponents(containerID, track, opts.Lock, opts.Profile, langs)
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
	return cp.Copy(src, dst)
}
