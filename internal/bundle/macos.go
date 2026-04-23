package bundle

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// buildMacOSTrack builds a bundle for a darwin track by directly downloading
// pre-built binaries rather than using Docker.
func buildMacOSTrack(track BuildTrack, opts BuildOptions, langs []registry.ResolvedLanguage, lockHash string) BuildResult {
	r := BuildResult{Track: track}
	s, err := store.NewStore()
	if err != nil {
		r.Error = err
		return r
	}

	var components []store.Component

	// 1. Download neovim binary for macOS (only when nvim is enabled for this profile).
	prof := opts.Lock.Profiles[opts.Profile]
	if prof.NvimEnabled() {
		nvimComps, err := buildNvimMacOS(track, opts.Lock, s)
		if err != nil {
			r.Error = fmt.Errorf("nvim macOS: %w", err)
			return r
		}
		components = append(components, nvimComps...)
	}

	// 2. Download language runtimes.
	for _, rl := range langs {
		if !rl.Language.MacOSSupported {
			continue
		}
		rt := rl.Runtime
		if rt == nil || len(rt.MacOSDownloads) == 0 || len(rt.ContainerPaths) == 0 {
			continue
		}
		comps, err := BuildMacOSComponent(rl, track, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s runtime: %v\n", rl.Language.Name, err)
			continue
		}
		components = append(components, comps...)
	}

	// 3. Download CLI tools.
	for _, name := range flattenCLI(opts.Lock, opts.Profile) {
		comp, err := downloadCLIMacOS(name, track, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping CLI tool %s: %v\n", name, err)
			continue
		}
		if comp != nil {
			components = append(components, *comp)
		}
	}

	// 4. Download custom tools (formatters, linters, LSPs from ferry-tools.yaml).
	if opts.Tools != nil {
		allCustom := make(map[string]config.CustomTool)
		for name, ct := range opts.Tools.Formatters {
			allCustom[name] = ct
		}
		for name, ct := range opts.Tools.Linters {
			allCustom[name] = ct
		}
		for name, ct := range opts.Tools.LSPs {
			allCustom[name] = ct
		}

		// Only download custom tools that are actually referenced by the profile's languages.
		needed := make(map[string]bool)
		for _, rl := range langs {
			for _, name := range rl.EffectiveFormatters {
				if _, ok := opts.Tools.Formatters[name]; ok {
					needed[name] = true
				}
			}
			for _, name := range rl.EffectiveLinters {
				if _, ok := opts.Tools.Linters[name]; ok {
					needed[name] = true
				}
			}
			if _, ok := opts.Tools.LSPs[rl.EffectiveLSP]; ok {
				needed[rl.EffectiveLSP] = true
			}
		}

		for name := range needed {
			ct := allCustom[name]
			comp, err := downloadCustomToolMacOS(name, ct, track, s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping custom tool %s: %v\n", name, err)
				continue
			}
			if comp != nil {
				components = append(components, *comp)
			}
		}
	}

	// 5. Shell components — copied from local filesystem.
	if sp := prof.Shell; sp != nil {
		type shellSpec struct {
			id          string
			localPath   string
			installPath string
			preserve    bool
		}
		var shellSpecs []shellSpec
		if sp.Framework != "" && sp.FrameworkPath != "" {
			shellSpecs = append(shellSpecs, shellSpec{
				id:          "shell/framework",
				localPath:   config.ExpandHome(sp.FrameworkPath),
				installPath: sp.FrameworkPath + "/",
			})
		}
		if sp.RCPath != "" {
			shellSpecs = append(shellSpecs, shellSpec{
				id:          "shell/rc",
				localPath:   config.ExpandHome(sp.RCPath),
				installPath: sp.RCPath,
				preserve:    true,
			})
		}
		if sp.ThemeConfigPath != "" {
			shellSpecs = append(shellSpecs, shellSpec{
				id:          "shell/theme-config",
				localPath:   config.ExpandHome(sp.ThemeConfigPath),
				installPath: sp.ThemeConfigPath,
				preserve:    true,
			})
		}

		for _, ss := range shellSpecs {
			tmpDir := fmt.Sprintf("/tmp/ferry-shell-%s-%s", track.Arch, sanitizeID(ss.id))
			if err := copyLocalToTmp(ss.localPath, tmpDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping shell component %s: %v\n", ss.id, err)
				os.RemoveAll(tmpDir)
				continue
			}
			compressed, err := store.CompressDir(tmpDir, opts.Lock.Bundle.Exclude)
			os.RemoveAll(tmpDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping shell component %s: compress failed: %v\n", ss.id, err)
				continue
			}
			hash := store.HashBytes(compressed)
			if err := s.Put(hash, compressed); err != nil {
				r.Error = fmt.Errorf("storing shell component %s: %w", ss.id, err)
				return r
			}
			components = append(components, store.Component{
				ID:             ss.id,
				Hash:           hash,
				SizeCompressed: int64(len(compressed)),
				InstallPath:    ss.installPath,
				ArchSpecific:   false,
				Preserve:       ss.preserve,
			})
		}
	}

	// 6. Write manifest.
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

// downloadCustomToolMacOS downloads a custom tool binary for macOS using the
// download spec from the ferry-tools.yaml file.
func downloadCustomToolMacOS(name string, ct config.CustomTool, track BuildTrack, s *store.Store) (*store.Component, error) {
	if len(ct.MacOSDownloads) == 0 {
		return nil, nil // no macOS download configured — skip silently
	}

	// Select the best download for this arch.
	var selected *config.CustomMacOSDownload
	for i := range ct.MacOSDownloads {
		dl := &ct.MacOSDownloads[i]
		if dl.Arch == track.Arch {
			selected = dl
			break
		}
		if dl.Arch == "universal" && selected == nil {
			selected = dl
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no macOS download for arch %q", track.Arch)
	}

	tmpFile, err := downloadToTemp(selected.URL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", selected.URL, err)
	}

	tmpDir, err := os.MkdirTemp("", "ferry-custom-macos-*")
	if err != nil {
		os.Remove(tmpFile)
		return nil, err
	}

	extractErr := extractArchive(tmpFile, selected.ArchiveRoot, tmpDir)
	os.Remove(tmpFile)
	if extractErr != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("extracting %s: %w", name, extractErr)
	}

	// Locate the binary — use InnerPath if specified, otherwise the first file.
	var binSrc string
	if selected.InnerPath != "" {
		binSrc = tmpDir + "/" + selected.InnerPath
	} else {
		entries, err := os.ReadDir(tmpDir)
		if err != nil || len(entries) == 0 {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("no file found after downloading %s", name)
		}
		binSrc = tmpDir + "/" + entries[0].Name()
	}

	stageDir, err := os.MkdirTemp("", "ferry-custom-stage-*")
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}
	stageBin := stageDir + "/" + name
	if err := copyFile(binSrc, stageBin); err != nil {
		os.RemoveAll(tmpDir)
		os.RemoveAll(stageDir)
		return nil, fmt.Errorf("staging %s binary: %w", name, err)
	}
	os.Chmod(stageBin, 0755) //nolint:errcheck
	os.RemoveAll(tmpDir)

	compressed, err := store.CompressDir(stageDir, nil)
	os.RemoveAll(stageDir)
	if err != nil {
		return nil, fmt.Errorf("compressing %s: %w", name, err)
	}

	hash := store.HashBytes(compressed)
	if err := s.Put(hash, compressed); err != nil {
		return nil, fmt.Errorf("storing %s: %w", name, err)
	}

	return &store.Component{
		ID:             "custom/" + name,
		Hash:           hash,
		SizeCompressed: int64(len(compressed)),
		InstallPath:    "~/.local/bin/" + name,
		ArchSpecific:   true,
	}, nil
}

// buildNvimMacOS downloads the neovim release tarball for the given darwin track.
func buildNvimMacOS(track BuildTrack, lock *config.LockFile, s *store.Store) ([]store.Component, error) {
	nvimVersion := lock.Nvim.Version
	if nvimVersion == "" {
		nvimVersion = "0.10.4"
	}

	// Neovim macOS release naming: nvim-macos-{arch}.tar.gz where arch is
	// "x86_64" or "arm64" — the same names ferry uses.
	url := fmt.Sprintf("https://github.com/neovim/neovim/releases/download/v%s/nvim-macos-%s.tar.gz", nvimVersion, track.Arch)
	archiveRoot := fmt.Sprintf("nvim-macos-%s/", track.Arch)

	tmpFile, err := downloadToTemp(url)
	if err != nil {
		return nil, fmt.Errorf("downloading nvim: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ferry-nvim-macos-*")
	if err != nil {
		os.Remove(tmpFile)
		return nil, err
	}

	extractErr := extractArchive(tmpFile, archiveRoot, tmpDir)
	os.Remove(tmpFile)
	if extractErr != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("extracting nvim: %w", extractErr)
	}

	compressed, err := store.CompressDir(tmpDir, nil)
	os.RemoveAll(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("compressing nvim: %w", err)
	}

	hash := store.HashBytes(compressed)
	if err := s.Put(hash, compressed); err != nil {
		return nil, fmt.Errorf("storing nvim: %w", err)
	}

	return []store.Component{
		{
			ID:             "nvim-binary",
			Version:        nvimVersion,
			Hash:           hash,
			SizeCompressed: int64(len(compressed)),
			InstallPath:    "~/.local/share/nvim-dist/",
			BinSymlink:     "~/.local/bin/nvim",
			ArchSpecific:   true,
		},
	}, nil
}

// downloadCLIMacOS downloads a single CLI tool binary for macOS, wraps it in
// a content-addressed component, and returns a Component ready for the manifest.
func downloadCLIMacOS(name string, track BuildTrack, s *store.Store) (*store.Component, error) {
	dl, ok := CLIMacOSDownloads[name]
	if !ok {
		return nil, fmt.Errorf("no macOS download spec for %q", name)
	}

	archStr, ok := dl.Arches[track.Arch]
	if !ok {
		return nil, fmt.Errorf("no macOS arch %q entry for %q", track.Arch, name)
	}

	replacer := strings.NewReplacer("{VERSION}", dl.Version, "{ARCH}", archStr)
	url := replacer.Replace(dl.URLTemplate)
	innerPath := replacer.Replace(dl.ArchiveInnerPath)

	tmpFile, err := downloadToTemp(url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}

	tmpDir, err := os.MkdirTemp("", "ferry-cli-macos-*")
	if err != nil {
		os.Remove(tmpFile)
		return nil, err
	}

	extractErr := extractArchive(tmpFile, "", tmpDir)
	os.Remove(tmpFile)
	if extractErr != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("extracting %s: %w", name, extractErr)
	}

	// For archives, locate the binary at innerPath; for direct downloads the
	// file lands as the archive name — just grab the first (only) file.
	var binSrc string
	if innerPath != "" {
		binSrc = tmpDir + "/" + innerPath
	} else {
		// Direct binary download: tmpDir contains exactly one file.
		entries, err := os.ReadDir(tmpDir)
		if err != nil || len(entries) == 0 {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("no file found after downloading %s", name)
		}
		binSrc = tmpDir + "/" + entries[0].Name()
	}

	// Create a clean staging dir containing just the binary named after the tool.
	stageDir, err := os.MkdirTemp("", "ferry-cli-stage-*")
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}
	stageBin := stageDir + "/" + name
	if err := copyFile(binSrc, stageBin); err != nil {
		os.RemoveAll(tmpDir)
		os.RemoveAll(stageDir)
		return nil, fmt.Errorf("staging %s binary: %w", name, err)
	}
	os.Chmod(stageBin, 0755) //nolint:errcheck
	os.RemoveAll(tmpDir)

	compressed, err := store.CompressDir(stageDir, nil)
	os.RemoveAll(stageDir)
	if err != nil {
		return nil, fmt.Errorf("compressing %s: %w", name, err)
	}

	hash := store.HashBytes(compressed)
	if err := s.Put(hash, compressed); err != nil {
		return nil, fmt.Errorf("storing %s: %w", name, err)
	}

	return &store.Component{
		ID:             "cli/" + name,
		Version:        dl.Version,
		Hash:           hash,
		SizeCompressed: int64(len(compressed)),
		InstallPath:    "~/.local/bin/" + name,
		ArchSpecific:   true,
	}, nil
}
