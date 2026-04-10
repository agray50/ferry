package store

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/gobwas/glob"
)

// HashBytes returns the SHA256 hex string of the given bytes.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

// HashFile returns the SHA256 hex string of a file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashDir returns the SHA256 hex string of a directory's contents.
// Hash is computed over all file paths and their contents, sorted.
// Excludes patterns matching any entry in excludePatterns (glob).
func HashDir(path string, excludePatterns []string) (string, error) {
	globs := compileGlobs(excludePatterns)

	type entry struct {
		rel  string
		data []byte
	}
	var entries []entry

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		if d.IsDir() {
			return nil
		}
		if matchesAny(rel, globs) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		entries = append(entries, entry{rel: rel, data: data})
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })

	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%s\x00", e.rel)
		h.Write(e.data)
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func compileGlobs(patterns []string) []glob.Glob {
	var out []glob.Glob
	for _, p := range patterns {
		if g, err := glob.Compile(p, '/'); err == nil {
			out = append(out, g)
		}
	}
	return out
}

func matchesAny(path string, globs []glob.Glob) bool {
	for _, g := range globs {
		if g.Match(path) {
			return true
		}
	}
	return false
}
