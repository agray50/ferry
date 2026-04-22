package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverShellPure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pureDir := filepath.Join(home, ".zsh", "pure")
	os.MkdirAll(pureDir, 0755)
	os.WriteFile(filepath.Join(pureDir, "pure.zsh"), []byte("# pure"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "pure" {
		t.Errorf("ThemeDetected = %q, want pure", d.ThemeDetected)
	}
	if d.ThemeConfigPath != "" {
		t.Errorf("pure has no config file, ThemeConfigPath should be empty, got %q", d.ThemeConfigPath)
	}
}

func TestDiscoverShellP10k(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	os.WriteFile(filepath.Join(home, ".p10k.zsh"), []byte("# p10k"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "p10k" {
		t.Errorf("ThemeDetected = %q, want p10k", d.ThemeDetected)
	}
	if d.ThemeConfigPath != filepath.Join(home, ".p10k.zsh") {
		t.Errorf("ThemeConfigPath = %q, want ~/.p10k.zsh", d.ThemeConfigPath)
	}
}

func TestDiscoverShellStarship(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	starshipDir := filepath.Join(home, ".config")
	os.MkdirAll(starshipDir, 0755)
	os.WriteFile(filepath.Join(starshipDir, "starship.toml"), []byte("# starship"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "starship" {
		t.Errorf("ThemeDetected = %q, want starship", d.ThemeDetected)
	}
}
