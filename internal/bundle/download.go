package bundle

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/ulikunitz/xz"

	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// httpClient is a shared retryable HTTP client with a 3-retry policy and
// no logger output. Reused across all downloads in a build session.
var httpClient = func() *retryablehttp.Client {
	c := retryablehttp.NewClient()
	c.RetryMax = 3
	c.Logger = nil // suppress default INFO log lines
	return c
}()

// BuildMacOSComponent downloads a runtime for a darwin track, extracts it
// to a temp directory, and stores it as a content-addressed component.
func BuildMacOSComponent(lang registry.ResolvedLanguage, track BuildTrack, s *store.Store) ([]store.Component, error) {
	rt := lang.Runtime
	if rt == nil {
		return nil, nil
	}
	version := rt.DefaultVersion

	// MacOSDownloads provides arch-specific variants for the primary runtime
	// (ContainerPaths[0]). Multi-path runtimes (e.g. Go runtime + tools) only
	// bundle the primary path; supplementary paths require separate downloads.
	if len(rt.ContainerPaths) == 0 {
		return nil, nil
	}
	cp := rt.ContainerPaths[0]

	var components []store.Component
	{
		dl, err := selectMacOSDownload(rt.MacOSDownloads, track.Arch)
		if err != nil {
			return nil, fmt.Errorf("no macOS download for %s/%s: %w", lang.Language.Name, track.Arch, err)
		}

		url := substituteVars(dl.URL, version, track.Arch)
		installPath := substituteVars(cp.InstallPath, version, track.Arch)

		tmpFile, err := downloadToTemp(url)
		if err != nil {
			return nil, fmt.Errorf("downloading %s: %w", url, err)
		}

		tmpDir, err := os.MkdirTemp("", "ferry-macos-*")
		if err != nil {
			os.Remove(tmpFile)
			return nil, err
		}

		extractErr := extractArchive(tmpFile, dl.ArchiveRoot, tmpDir)
		os.Remove(tmpFile)
		if extractErr != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("extracting %s: %w", url, extractErr)
		}

		compressed, err := store.CompressDir(tmpDir, nil)
		os.RemoveAll(tmpDir)
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
// The temp file preserves the URL's archive extension so extractArchive can
// dispatch on it (e.g. *.tar.gz, *.tar.xz, *.zip).
func downloadToTemp(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.CreateTemp("", "ferry-dl-*"+urlArchiveExt(url))
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

// urlArchiveExt returns the archive extension from a URL path (e.g. ".tar.gz"),
// stripping any query string first. Returns "" for bare binary downloads.
func urlArchiveExt(url string) string {
	base := filepath.Base(strings.SplitN(url, "?", 2)[0])
	for _, ext := range []string{".tar.gz", ".tar.xz", ".tar.zst", ".tgz", ".zip"} {
		if strings.HasSuffix(base, ext) {
			return ext
		}
	}
	return ""
}

// extractArchive extracts a .tar.gz, .tar.xz, .tar.zst, or .zip archive to destDir.
// If archiveRoot is non-empty, only files under that prefix are extracted.
// Direct binary downloads (no recognised extension) are copied as-is.
func extractArchive(path, archiveRoot, destDir string) error {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTar(path, archiveRoot, destDir, "gz")
	case strings.HasSuffix(lower, ".tar.xz"):
		return extractTar(path, archiveRoot, destDir, "xz")
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(path, archiveRoot, destDir)
	default:
		base := filepath.Base(path)
		return copyFile(path, filepath.Join(destDir, base))
	}
}

// safeJoin joins dest and name and returns an error if the result escapes dest.
// Prevents zip-slip / tar-slip path traversal attacks.
func safeJoin(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("archive entry %q would escape destination", name)
	}
	return target, nil
}

// extractTar extracts a tar archive compressed with "gz" or "xz" to dest.
func extractTar(src, root, dest, compression string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader
	switch compression {
	case "gz":
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		r = gr
	case "xz":
		xr, err := xz.NewReader(f)
		if err != nil {
			return err
		}
		r = xr
	default:
		return fmt.Errorf("unsupported compression %q", compression)
	}

	tr := tar.NewReader(r)
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
		target, err := safeJoin(dest, name)
		if err != nil {
			continue // skip malicious entry
		}
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
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
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
		target, err := safeJoin(dest, name)
		if err != nil {
			continue // skip malicious entry
		}
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
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
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
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
