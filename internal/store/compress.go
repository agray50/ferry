package store

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// CompressDir compresses a directory to zstd bytes.
// excludePatterns are glob patterns relative to dir root.
func CompressDir(dir string, excludePatterns []string) ([]byte, error) {
	globs := compileGlobs(excludePatterns)

	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(enc)

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		if matchesAny(rel, globs) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Use Lstat so we get info about the symlink itself, not its target.
		// d.Info() follows symlinks via WalkDir and would never report ModeSymlink.
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		// Resolve symlink target for proper tar header construction.
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		hdr.Name = rel
		if d.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		// Symlinks and directories have no data payload.
		if info.Mode()&os.ModeSymlink != 0 || d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("CompressDir: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CompressFile compresses a single file to zstd bytes.
func CompressFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := enc.Write(data); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExtractTo extracts zstd-compressed bytes to a destination directory.
// Creates destination if it does not exist.
// Preserves file permissions from archive.
func ExtractTo(data []byte, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer dec.Close()

	tr := tar.NewReader(dec)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractTo: %w", err)
		}

		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			continue // skip path-traversal attempt
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, fs.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Validate that the resolved symlink target stays within dest.
			resolved := filepath.Join(filepath.Dir(target), hdr.Linkname)
			if rel, err := filepath.Rel(dest, resolved); err != nil || strings.HasPrefix(rel, "..") {
				continue // symlink would escape dest
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			// Remove existing file/symlink at target before creating new symlink.
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

// safeJoin joins dest and name and returns an error if the result escapes dest.
// Prevents tar-slip path traversal attacks.
func safeJoin(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("archive entry %q would escape destination", name)
	}
	return target, nil
}
