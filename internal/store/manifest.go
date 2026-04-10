package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/ferry/internal/config"
)

func manifestFilename(profile, arch, libc, hash string) string {
	return fmt.Sprintf("%s-%s-%s-%s.json", profile, arch, libc, hash)
}

// WriteManifest writes a manifest JSON to ~/.ferry/bundles/.
func WriteManifest(m *Manifest) error {
	if err := config.EnsureFerryDir(); err != nil {
		return err
	}
	name := manifestFilename(m.Profile, m.Arch, m.Libc, m.LockfileHash)
	path := filepath.Join(config.BundlesDir(), name)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ReadManifest reads a manifest by profile/arch/libc/hash.
func ReadManifest(profile, arch, libc, hash string) (*Manifest, error) {
	name := manifestFilename(profile, arch, libc, hash)
	path := filepath.Join(config.BundlesDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadManifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListManifests returns all manifests in ~/.ferry/bundles/.
func ListManifests() ([]*Manifest, error) {
	dir := config.BundlesDir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*Manifest
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		out = append(out, &m)
	}
	return out, nil
}

// FindManifest returns the latest manifest for profile + arch + libc.
func FindManifest(profile, arch, libc string) (*Manifest, error) {
	manifests, err := ListManifests()
	if err != nil {
		return nil, err
	}
	var found *Manifest
	for _, m := range manifests {
		if m.Profile == profile && m.Arch == arch && m.Libc == libc {
			if found == nil || m.BuiltAt > found.BuiltAt {
				found = m
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no manifest found for profile=%s arch=%s libc=%s\n  run: ferry bundle --arch %s --libc %s",
			profile, arch, libc, arch, libc)
	}
	return found, nil
}
