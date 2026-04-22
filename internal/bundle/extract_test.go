package bundle

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/config"
)

func TestSubstituteContainerPath(t *testing.T) {
	path := "/root/.pyenv/versions/{VERSION}/"
	got := substituteVars(path, "3.12.4", "x86_64")
	want := "/root/.pyenv/versions/3.12.4/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteInstallPathFull(t *testing.T) {
	path := "~/.ferry/runtimes/python-{VERSION}/"
	got := substituteVars(path, "3.12.4", "x86_64")
	want := "~/.ferry/runtimes/python-3.12.4/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteRustArch(t *testing.T) {
	path := "/root/.rustup/toolchains/{VERSION}-{ARCH}-unknown-linux-gnu/"
	got := substituteVars(path, "stable", "x86_64")
	want := "/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildComponentSpecsNvimDisabled(t *testing.T) {
	falseVal := false
	lock := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {IncludeNvim: &falseVal},
		},
	}
	track := BuildTrack{Arch: "x86_64", OS: "linux"}
	specs := buildComponentSpecs(track, lock, "shell-only", nil)
	for _, s := range specs {
		if s.id == "nvim-binary" || strings.HasPrefix(s.id, "treesitter") || strings.HasPrefix(s.id, "lazy/") {
			t.Errorf("nvim-disabled profile should not produce spec %q", s.id)
		}
	}
}

func TestBuildComponentSpecsShellComponents(t *testing.T) {
	lock := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {
				Shell: &config.ShellProfile{
					Type:            "zsh",
					Framework:       "oh-my-zsh",
					FrameworkPath:   "~/.oh-my-zsh",
					RCPath:          "~/.zshrc",
					Theme:           "p10k",
					ThemeConfigPath: "~/.p10k.zsh",
				},
			},
		},
	}
	track := BuildTrack{Arch: "x86_64", OS: "linux"}
	specs := buildComponentSpecs(track, lock, "shell-only", nil)

	ids := make(map[string]componentSpec)
	for _, s := range specs {
		ids[s.id] = s
	}

	if _, ok := ids["shell/framework"]; !ok {
		t.Error("expected shell/framework component spec")
	}
	if _, ok := ids["shell/rc"]; !ok {
		t.Error("expected shell/rc component spec")
	}
	if rc := ids["shell/rc"]; !rc.preserve {
		t.Error("shell/rc should have preserve=true")
	}
	if _, ok := ids["shell/theme-config"]; !ok {
		t.Error("expected shell/theme-config component spec")
	}
	if tc := ids["shell/theme-config"]; !tc.preserve {
		t.Error("shell/theme-config should have preserve=true")
	}
}
