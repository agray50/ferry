package bundle

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/registry"
)

func TestSubstituteVarsURL(t *testing.T) {
	url := substituteVars("https://example.com/{VERSION}/tool-{ARCH}.tar.gz", "1.2.3", "x86_64")
	if strings.Contains(url, "{VERSION}") {
		t.Error("url should not contain {VERSION}")
	}
	if strings.Contains(url, "{ARCH}") {
		t.Error("url should not contain {ARCH}")
	}
	if !strings.Contains(url, "1.2.3") {
		t.Error("url should contain version 1.2.3")
	}
	if !strings.Contains(url, "x86_64") {
		t.Error("url should contain arch x86_64")
	}
}

func TestSubstituteVarsInstallPath(t *testing.T) {
	path := substituteVars("~/.ferry/runtimes/python-{VERSION}/", "3.12.4", "arm64")
	if path != "~/.ferry/runtimes/python-3.12.4/" {
		t.Errorf("got %q, want ~/.ferry/runtimes/python-3.12.4/", path)
	}
}

func TestSelectMacOSDownloadForArch(t *testing.T) {
	downloads := []registry.MacOSDownload{
		{URL: "https://example.com/x86_64.tar.gz", Arch: "x86_64"},
		{URL: "https://example.com/arm64.tar.gz", Arch: "arm64"},
	}
	got, err := selectMacOSDownload(downloads, "arm64")
	if err != nil {
		t.Fatalf("selectMacOSDownload: %v", err)
	}
	if got.Arch != "arm64" {
		t.Errorf("got arch %q, want arm64", got.Arch)
	}
}

func TestSelectMacOSDownloadUniversalFallback(t *testing.T) {
	downloads := []registry.MacOSDownload{
		{URL: "https://example.com/universal.tar.gz", Arch: "universal"},
	}
	got, err := selectMacOSDownload(downloads, "arm64")
	if err != nil {
		t.Fatalf("selectMacOSDownload: %v", err)
	}
	if got.Arch != "universal" {
		t.Errorf("universal should match any arch, got %q", got.Arch)
	}
}

func TestSelectMacOSDownloadNoMatch(t *testing.T) {
	downloads := []registry.MacOSDownload{
		{URL: "https://example.com/x86_64.tar.gz", Arch: "x86_64"},
	}
	_, err := selectMacOSDownload(downloads, "arm64")
	if err == nil {
		t.Error("expected error when no matching download found")
	}
}
