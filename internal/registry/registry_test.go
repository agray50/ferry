package registry

import (
	"testing"

	"github.com/anthropics/ferry/internal/config"
)

func TestGet(t *testing.T) {
	l, err := Get("python")
	if err != nil {
		t.Fatalf("Get(python): %v", err)
	}
	if l.Name != "python" {
		t.Errorf("name: got %q", l.Name)
	}
	if l.LSP != "pyright" {
		t.Errorf("lsp: got %q want pyright", l.LSP)
	}
	if l.Runtime == nil || l.Runtime.Manager != "pyenv" {
		t.Error("expected pyenv runtime")
	}
}

func TestGetUnknown(t *testing.T) {
	_, err := Get("unknown")
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
}

func TestIsValidLSP(t *testing.T) {
	if !IsValidLSP("python", "pyright") {
		t.Error("pyright should be valid for python")
	}
	if !IsValidLSP("python", "pylsp") {
		t.Error("pylsp should be valid alternate for python")
	}
	if IsValidLSP("python", "rust-analyzer") {
		t.Error("rust-analyzer should not be valid for python")
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("expected non-empty names")
	}
	// verify sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("names not sorted: %v", names)
			break
		}
	}
}

func TestResolveLanguages(t *testing.T) {
	enabled := []string{"python", "go"}
	overrides := map[string]config.LanguageOverride{
		"python": {LSP: "pylsp", RuntimeVersion: "3.11", ExtraPackages: []string{"numpy", "pandas"}},
	}

	langs, err := ResolveLanguages(enabled, overrides)
	if err != nil {
		t.Fatalf("ResolveLanguages: %v", err)
	}
	if len(langs) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(langs))
	}
	if langs[0].LSP != "pylsp" {
		t.Errorf("expected pylsp override, got %s", langs[0].LSP)
	}
	if langs[0].Runtime.DefaultVersion != "3.11" {
		t.Errorf("expected 3.11 runtime, got %s", langs[0].Runtime.DefaultVersion)
	}
	if len(langs[0].Runtime.ExtraPackages) != 2 {
		t.Errorf("expected 2 extra packages, got %d", len(langs[0].Runtime.ExtraPackages))
	}
}

func TestResolveLanguagesUnknown(t *testing.T) {
	_, err := ResolveLanguages([]string{"cobol"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
}

func TestResolveLanguagesInvalidLSP(t *testing.T) {
	_, err := ResolveLanguages([]string{"python"},
		map[string]config.LanguageOverride{
			"python": {LSP: "rust-analyzer"},
		})
	if err == nil {
		t.Fatal("expected error for invalid LSP override")
	}
}
