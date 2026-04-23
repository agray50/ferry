package bundle

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

func TestGenerateDockerfileContainsBaseFrom(t *testing.T) {
	track := BuildTracks[0] // linux/x86_64
	langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
		{Name: "python", Tier: "full"},
	}, nil)
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, "", langs, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if !strings.HasPrefix(df, "FROM ubuntu:22.04") {
		t.Errorf("Dockerfile should start with FROM ubuntu:22.04, got:\n%s", df[:100])
	}
}

func TestGenerateDockerfileContainsBuildSteps(t *testing.T) {
	track := BuildTracks[0]
	langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
		{Name: "python", Tier: "full"},
	}, nil)
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, "", langs, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if !strings.Contains(df, "pyenv.run") {
		t.Error("Dockerfile should contain pyenv install step")
	}
	if !strings.Contains(df, "ENV PATH") {
		t.Error("Dockerfile should contain ENV PATH for pyenv")
	}
}

func TestGenerateDockerfileSubstitutesVersion(t *testing.T) {
	track := BuildTracks[0]
	langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
		{Name: "python", Tier: "full", RuntimeVersion: "3.11"},
	}, nil)
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, "", langs, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if strings.Contains(df, "{VERSION}") {
		t.Error("Dockerfile should not contain unsubstituted {VERSION}")
	}
	if !strings.Contains(df, "3.11") {
		t.Error("Dockerfile should contain substituted version 3.11")
	}
}

func TestGenerateDockerfileSubstitutesArch(t *testing.T) {
	track := BuildTracks[0] // x86_64
	langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
		{Name: "go", Tier: "full"},
	}, nil)
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, "", langs, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if strings.Contains(df, "{ARCH}") {
		t.Error("Dockerfile should not contain unsubstituted {ARCH}")
	}
	if !strings.Contains(df, "amd64") {
		t.Error("Dockerfile should contain amd64 for x86_64 track")
	}
}

func TestGenerateDockerfileDeduplicatesSharedRuntime(t *testing.T) {
	track := BuildTracks[0]
	// JS and TS share nvm — nvm install should appear exactly once
	langs, _ := registry.ResolveFromProfile([]config.LanguageConfig{
		{Name: "javascript", Tier: "full"},
		{Name: "typescript", Tier: "full"},
	}, nil)
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, "", langs, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	count := strings.Count(df, "nvm-sh/nvm")
	if count != 1 {
		t.Errorf("nvm install should appear exactly once, appeared %d times", count)
	}
}

func TestGenerateDockerfileErrorsForDarwinTrack(t *testing.T) {
	darwinTrack := BuildTrack{OS: "darwin", BuildMethod: "download"}
	_, err := GenerateDockerfile(darwinTrack, &config.LockFile{}, "", nil, nil)
	if err == nil {
		t.Error("GenerateDockerfile should return error for darwin track")
	}
}

func TestGenerateDockerfileNvimDisabled(t *testing.T) {
	falseVal := false
	track := BuildTracks[0] // linux/x86_64
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {IncludeNvim: &falseVal},
		},
	}
	df, err := GenerateDockerfile(track, lf, "shell-only", nil, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if strings.Contains(df, "neovim") || strings.Contains(df, "nvim") {
		t.Error("Dockerfile for nvim-disabled profile should not contain nvim")
	}
	if strings.Contains(df, "Lazy!") {
		t.Error("Dockerfile for nvim-disabled profile should not contain lazy.nvim bootstrap")
	}
	if strings.Contains(df, "TSInstall") {
		t.Error("Dockerfile for nvim-disabled profile should not contain TSInstall")
	}
}

func TestGenerateDockerfileNvimEnabled(t *testing.T) {
	trueVal := true
	track := BuildTracks[0]
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"default": {IncludeNvim: &trueVal},
		},
	}
	df, err := GenerateDockerfile(track, lf, "default", nil, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if !strings.Contains(df, "nvim") {
		t.Error("nvim-enabled Dockerfile should contain nvim")
	}
}
