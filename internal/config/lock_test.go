package config

import (
	"os"
	"testing"
)

func TestLanguageConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	lf := &LockFile{
		Profiles: map[string]ProfileConfig{
			"backend": {
				Description: "backend profile",
				Languages: []LanguageConfig{
					{Name: "go", Tier: "full", RuntimeVersion: "1.22.5"},
					{Name: "python", Tier: "lsp-only", LSP: "pylsp"},
				},
				Plugins: []string{"nvim-lspconfig", "telescope.nvim"},
				CLI:     []string{"rg", "fzf"},
			},
		},
	}

	if err := WriteLockFile(lf); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	got, err := ReadLockFile()
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}

	prof := got.Profiles["backend"]
	if len(prof.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(prof.Languages))
	}
	if prof.Languages[0].Name != "go" {
		t.Errorf("Languages[0].Name = %q, want go", prof.Languages[0].Name)
	}
	if prof.Languages[0].Tier != "full" {
		t.Errorf("Languages[0].Tier = %q, want full", prof.Languages[0].Tier)
	}
	if prof.Languages[1].LSP != "pylsp" {
		t.Errorf("Languages[1].LSP = %q, want pylsp", prof.Languages[1].LSP)
	}
	if len(prof.CLI) != 2 {
		t.Errorf("CLI len = %d, want 2", len(prof.CLI))
	}
}

func TestNvimEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	// nil pointer = unset = enabled (backward compat)
	p := ProfileConfig{}
	if !p.NvimEnabled() {
		t.Error("nil IncludeNvim should mean nvim is enabled")
	}

	// explicit true
	p.IncludeNvim = &trueVal
	if !p.NvimEnabled() {
		t.Error("IncludeNvim=true should mean nvim is enabled")
	}

	// explicit false
	p.IncludeNvim = &falseVal
	if p.NvimEnabled() {
		t.Error("IncludeNvim=false should mean nvim is disabled")
	}
}

func TestShellProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	trueVal := true
	lf := &LockFile{
		Profiles: map[string]ProfileConfig{
			"shell-only": {
				Description: "shell only profile",
				IncludeNvim: &trueVal,
				Shell: &ShellProfile{
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

	if err := WriteLockFile(lf); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}
	got, err := ReadLockFile()
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	prof := got.Profiles["shell-only"]
	if prof.Shell == nil {
		t.Fatal("Shell should not be nil after round-trip")
	}
	if prof.Shell.Type != "zsh" {
		t.Errorf("Shell.Type = %q, want zsh", prof.Shell.Type)
	}
	if prof.Shell.Framework != "oh-my-zsh" {
		t.Errorf("Shell.Framework = %q, want oh-my-zsh", prof.Shell.Framework)
	}
	if prof.Shell.FrameworkPath != "~/.oh-my-zsh" {
		t.Errorf("Shell.FrameworkPath = %q, want ~/.oh-my-zsh", prof.Shell.FrameworkPath)
	}
	if prof.Shell.RCPath != "~/.zshrc" {
		t.Errorf("Shell.RCPath = %q, want ~/.zshrc", prof.Shell.RCPath)
	}
	if prof.Shell.Theme != "p10k" {
		t.Errorf("Shell.Theme = %q, want p10k", prof.Shell.Theme)
	}
	if prof.Shell.ThemeConfigPath != "~/.p10k.zsh" {
		t.Errorf("Shell.ThemeConfigPath = %q, want ~/.p10k.zsh", prof.Shell.ThemeConfigPath)
	}
	if prof.IncludeNvim == nil || !*prof.IncludeNvim {
		t.Error("IncludeNvim should be true after round-trip")
	}
}

func TestDefaultLockFileNvimEnabled(t *testing.T) {
	lf := DefaultLockFile()
	for name, prof := range lf.Profiles {
		if !prof.NvimEnabled() {
			t.Errorf("profile %q: DefaultLockFile should have nvim enabled", name)
		}
	}
}

func TestDefaultLockFileProfiles(t *testing.T) {
	lf := DefaultLockFile()
	if len(lf.Profiles) == 0 {
		t.Fatal("DefaultLockFile should have at least one profile")
	}
	def, ok := lf.Profiles["default"]
	if !ok {
		t.Fatal("DefaultLockFile should have a 'default' profile")
	}
	if def.Languages == nil {
		t.Error("default profile Languages should be non-nil (empty slice, not nil)")
	}
}
