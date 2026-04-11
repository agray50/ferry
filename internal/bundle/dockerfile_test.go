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
	})
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, langs)
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
	})
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, langs)
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
	})
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, langs)
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
	})
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, langs)
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
	})
	lf := &config.LockFile{}
	df, err := GenerateDockerfile(track, lf, langs)
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
	_, err := GenerateDockerfile(darwinTrack, &config.LockFile{}, nil)
	if err == nil {
		t.Error("GenerateDockerfile should return error for darwin track")
	}
}
