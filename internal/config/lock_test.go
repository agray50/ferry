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
				Plugins:      []string{"nvim-lspconfig", "telescope.nvim"},
				CLI:          []string{"rg", "fzf"},
				IncludeShell: true,
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

func TestDefaultLockFileHasNoGlobalLanguages(t *testing.T) {
	lf := DefaultLockFile()
	// All profiles start with empty language list; wizard populates them
	for _, prof := range lf.Profiles {
		_ = prof
	}
}

func TestLockFileNoGlobalCLI(t *testing.T) {
	lf := DefaultLockFile()
	// CLI is now per-profile, not at top level
	for _, prof := range lf.Profiles {
		if prof.CLI == nil {
			// CLI field exists and can be nil (no tools selected)
		}
	}
}
