package bundle

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// BuildMacOSComponent downloads a runtime for a darwin track, extracts it
// to a temp directory, and stores it as a content-addressed component.
func BuildMacOSComponent(lang registry.ResolvedLanguage, track BuildTrack, s *store.Store) ([]store.Component, error) {
	rt := lang.Runtime
	if rt == nil {
		return nil, nil
	}
	version := rt.DefaultVersion

	var components []store.Component
	for i, path := range rt.ContainerPaths {
		_ = path.Container // unused for darwin; MacOSDownloads used instead

		if i >= len(rt.MacOSDownloads) {
			break
		}
		dl, err := selectMacOSDownload(rt.MacOSDownloads, track.Arch)
		if err != nil {
			return nil, fmt.Errorf("no macOS download for %s/%s: %w", lang.Language.Name, track.Arch, err)
		}

		url := substituteDownloadURL(dl.URL, version, track.Arch)
		installPath := substituteDownloadURL(path.InstallPath, version, track.Arch)

		tmpFile, err := downloadToTemp(url)
		if err != nil {
			return nil, fmt.Errorf("downloading %s: %w", url, err)
		}
		defer os.Remove(tmpFile)

		tmpDir, err := os.MkdirTemp("", "ferry-macos-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)

		if err := extractArchive(tmpFile, dl.ArchiveRoot, tmpDir); err != nil {
			return nil, fmt.Errorf("extracting %s: %w", url, err)
		}

		compressed, err := store.CompressDir(tmpDir, nil)
		if err != nil {
			return nil, fmt.Errorf("compressing %s component: %w", lang.Language.Name, err)
		}

		hash := store.HashBytes(compressed)
		if err := s.Put(hash, compressed); err != nil {
			return nil, fmt.Errorf("storing %s component: %w", lang.Language.Name, err)
		}

		components = append(components, store.Component{
			ID:             fmt.Sprintf("runtime/%s", lang.Language.Name),
			Version:        version,
			Hash:           hash,
			SizeCompressed: int64(len(compressed)),
			InstallPath:    installPath,
			ArchSpecific:   true,
		})
	}
	return components, nil
}

// substituteDownloadURL substitutes {VERSION} and {ARCH} in a URL or path template.
func substituteDownloadURL(s, version, arch string) string {
	return substituteVars(s, version, arch)
}

// selectMacOSDownload picks the best MacOSDownload for the given arch.
// Prefers exact arch match; falls back to "universal".
func selectMacOSDownload(downloads []registry.MacOSDownload, arch string) (registry.MacOSDownload, error) {
	var universal *registry.MacOSDownload
	for _, dl := range downloads {
		if dl.Arch == arch {
			return dl, nil
		}
		if dl.Arch == "universal" {
			copy := dl
			universal = &copy
		}
	}
	if universal != nil {
		return *universal, nil
	}
	return registry.MacOSDownload{}, fmt.Errorf("no download found for arch %q", arch)
}

// downloadToTemp downloads url to a temporary file and returns its path.
func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.CreateTemp("", "ferry-dl-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// extractArchive extracts a .tar.gz, .tar.xz, or .zip archive to destDir.
// If archiveRoot is non-empty, only files under that prefix are extracted.
func extractArchive(path, archiveRoot, destDir string) error {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(path, archiveRoot, destDir)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(path, archiveRoot, destDir)
	default:
		base := filepath.Base(path)
		return copyFile(path, filepath.Join(destDir, base))
	}
}

func extractTarGz(src, root, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := hdr.Name
		if root != "" {
			if !strings.HasPrefix(name, root) {
				continue
			}
			name = strings.TrimPrefix(name, root)
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dest, name)
		if hdr.Typeflag == tar.TypeDir {
			os.MkdirAll(target, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	return nil
}

func extractZip(src, root, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := f.Name
		if root != "" {
			if !strings.HasPrefix(name, root) {
				continue
			}
			name = strings.TrimPrefix(name, root)
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dest, name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
