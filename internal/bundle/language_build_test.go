package bundle

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

// TestAllLanguageDockerfilesGenerate verifies every language with a runtime
// can produce a Dockerfile with no unsubstituted {VERSION} or {ARCH} placeholders.
func TestAllLanguageDockerfilesGenerate(t *testing.T) {
	track := BuildTracks[0] // linux/x86_64
	lf := config.DefaultLockFile()

	for _, lang := range registry.All() {
		if lang.Runtime == nil {
			continue
		}
		t.Run(lang.Name, func(t *testing.T) {
			langs, err := registry.ResolveFromProfile([]config.LanguageConfig{
				{Name: lang.Name, Tier: "full"},
			}, nil)
			if err != nil {
				t.Fatalf("ResolveFromProfile: %v", err)
			}
			df, err := GenerateDockerfile(track, lf, "", langs, nil)
			if err != nil {
				t.Fatalf("GenerateDockerfile: %v", err)
			}
			if strings.Contains(df, "{VERSION}") {
				t.Error("unsubstituted {VERSION} in dockerfile")
			}
			if strings.Contains(df, "{ARCH}") {
				t.Error("unsubstituted {ARCH} in dockerfile")
			}
		})
	}
}

// TestAllLanguageDownloadURLsReachable fetches HEAD on every download URL
// embedded in language BuildSteps and reports any that return non-2xx/3xx.
func TestAllLanguageDownloadURLsReachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping URL reachability check in short mode")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow — 3xx is fine
		},
	}

	urlRe := regexp.MustCompile(`https?://[^\s"\\]+`)
	track := BuildTracks[0]
	lf := config.DefaultLockFile()

	checked := map[string]bool{}

	for _, lang := range registry.All() {
		if lang.Runtime == nil {
			continue
		}
		langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
			{Name: lang.Name, Tier: "full"},
		}, nil)
		df, err := GenerateDockerfile(track, lf, "", langs, nil)
		if err != nil {
			t.Errorf("%s: GenerateDockerfile failed: %v", lang.Name, err)
			continue
		}

		urls := urlRe.FindAllString(df, -1)
		for _, u := range urls {
			// Strip trailing punctuation that may have been captured
			u = strings.TrimRight(u, `"'\`)
			// Skip installer scripts (they redirect or stream — not simple HEAD)
			if strings.Contains(u, "pyenv.run") ||
				strings.Contains(u, "rustup.rs") ||
				strings.Contains(u, "sdkman.io") ||
				strings.Contains(u, "asdf-vm") ||
				strings.Contains(u, "get.sdkman") ||
				strings.Contains(u, "dotnet-install") ||
				strings.Contains(u, "nvm-sh") {
				continue
			}
			if checked[u] {
				continue
			}
			checked[u] = true

			t.Run(fmt.Sprintf("%s/%s", lang.Name, urlShort(u)), func(t *testing.T) {
				resp, err := client.Head(u)
				if err != nil {
					t.Errorf("HEAD %s: %v", u, err)
					return
				}
				resp.Body.Close()
				if resp.StatusCode >= 400 {
					t.Errorf("HEAD %s → %d", u, resp.StatusCode)
				}
			})
		}
	}
}

func urlShort(u string) string {
	if len(u) > 60 {
		return u[:60] + "..."
	}
	return u
}
