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
// Hash is computed over all file paths and their contents, sorted by path.
// Files are streamed one at a time rather than loaded all into memory.
// Excludes patterns matching any entry in excludePatterns (glob).
func HashDir(path string, excludePatterns []string) (string, error) {
	globs := compileGlobs(excludePatterns)

	// Collect relative paths first so we can sort before hashing.
	var paths []string
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		if d.IsDir() || matchesAny(rel, globs) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(paths)

	h := sha256.New()
	for _, rel := range paths {
		fmt.Fprintf(h, "%s\x00", rel)
		f, err := os.Open(filepath.Join(path, rel))
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(h, f)
		f.Close()
		if copyErr != nil {
			return "", copyErr
		}
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
