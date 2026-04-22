package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/ferry/internal/config"
)

// Manifest describes a complete bundle for a given profile/arch/libc.
type Manifest struct {
	Profile      string      `json:"profile"`
	Arch         string      `json:"arch"`
	Libc         string      `json:"libc"`
	LockfileHash string      `json:"lockfile_hash"`
	BuiltAt      string      `json:"built_at"`
	Components   []Component `json:"components"`
}

// Component is a single installable unit within a manifest.
type Component struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Hash           string `json:"hash"`
	SizeCompressed int64  `json:"size_compressed"`
	InstallPath    string `json:"install_path"`
	BinSymlink     string `json:"bin_symlink,omitempty"`
	ArchSpecific   bool   `json:"arch_specific"`
	Preserve       bool   `json:"preserve,omitempty"`
}

// ComponentMeta holds metadata for a stored blob.
type ComponentMeta struct {
	Hash           string
	SizeCompressed int64
	CreatedAt      time.Time
}

// Store is a content-addressed local component store.
type Store struct {
	dir string
}

// NewStore opens the component store at ~/.ferry/store/.
func NewStore() (*Store, error) {
	if err := config.EnsureFerryDir(); err != nil {
		return nil, err
	}
	return &Store{dir: config.StoreDir()}, nil
}

func (s *Store) path(hash string) string {
	return filepath.Join(s.dir, hash+".tar.zst")
}

// Put stores a component tarball by hash. Idempotent.
func (s *Store) Put(hash string, data []byte) error {
	p := s.path(hash)
	if _, err := os.Stat(p); err == nil {
		return nil // already exists
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("store.Put %s: %w", hash, err)
	}
	return nil
}

// Get retrieves a component tarball by hash.
func (s *Store) Get(hash string) ([]byte, error) {
	data, err := os.ReadFile(s.path(hash))
	if err != nil {
		return nil, fmt.Errorf("store.Get %s: %w", hash, err)
	}
	return data, nil
}

// Exists returns true if the hash is present in the store.
func (s *Store) Exists(hash string) bool {
	_, err := os.Stat(s.path(hash))
	return err == nil
}

// List returns metadata for all components in the store.
func (s *Store) List() ([]ComponentMeta, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("store.List: %w", err)
	}
	var out []ComponentMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		// strip .tar.zst suffix
		if len(name) < 8 || name[len(name)-8:] != ".tar.zst" {
			continue
		}
		out = append(out, ComponentMeta{
			Hash:           name[:len(name)-8],
			SizeCompressed: info.Size(),
			CreatedAt:      info.ModTime(),
		})
	}
	return out, nil
}

// Delete removes a specific component by hash.
func (s *Store) Delete(hash string) error {
	return os.Remove(s.path(hash))
}

// Prune removes components not referenced by any manifest in ~/.ferry/bundles/.
// Returns count of removed components and bytes freed.
func (s *Store) Prune() (int, int64, error) {
	manifests, err := ListManifests()
	if err != nil {
		return 0, 0, err
	}

	// collect all referenced hashes
	referenced := make(map[string]bool)
	for _, m := range manifests {
		for _, c := range m.Components {
			referenced[c.Hash] = true
		}
	}

	all, err := s.List()
	if err != nil {
		return 0, 0, err
	}

	var count int
	var freed int64
	for _, meta := range all {
		if !referenced[meta.Hash] {
			if err := s.Delete(meta.Hash); err == nil {
				count++
				freed += meta.SizeCompressed
			}
		}
	}
	return count, freed, nil
}
