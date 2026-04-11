package bootstrap

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

func makeTestManifest() *store.Manifest {
	return &store.Manifest{
		Profile: "backend",
		Components: []store.Component{
			{ID: "nvim-binary", InstallPath: "~/.local/share/nvim-dist/", BinSymlink: "~/.local/bin/nvim"},
			{ID: "runtime/python", InstallPath: "~/.ferry/runtimes/python-3.12/", Version: "3.12"},
			{ID: "runtime/go", InstallPath: "~/.ferry/runtimes/go-1.22.5/", Version: "1.22.5"},
		},
	}
}

func makeTestLangs() []registry.ResolvedLanguage {
	pyRuntime := &registry.Runtime{
		ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/python-3.12/bin:$PATH"`},
	}
	goRuntime := &registry.Runtime{
		ShellInit: []string{
			`export PATH="$HOME/.ferry/runtimes/go-1.22.5/bin:$HOME/.ferry/runtimes/go-tools/bin:$PATH"`,
			`export GOPATH="$HOME/.ferry/runtimes/go-tools"`,
		},
	}
	return []registry.ResolvedLanguage{
		{Language: registry.Language{Name: "python"}, Runtime: pyRuntime},
		{Language: registry.Language{Name: "go"}, Runtime: goRuntime},
	}
}

func TestGenerateInstallScriptContainsFerryRuntimePath(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, ".ferry/runtimes/python-3.12") {
		t.Error("install script should reference python runtime PATH")
	}
	if !strings.Contains(script, ".ferry/runtimes/go-1.22.5") {
		t.Error("install script should reference go runtime PATH")
	}
}

func TestGenerateInstallScriptNoVersionManager(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// Version managers must not appear in the install script
	for _, mgr := range []string{"pyenv", "nvm", "rbenv", "rustup", "sdkman"} {
		if strings.Contains(script, mgr) {
			t.Errorf("install script should not reference version manager %q", mgr)
		}
	}
}

func TestGenerateInstallScriptIdempotentPathCheck(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// Each PATH export should be guarded by a grep check to prevent duplication
	if !strings.Contains(script, "grep -q") {
		t.Error("install script should use grep -q to check before appending PATH entries")
	}
}

func TestGenerateInstallScriptNvimSymlink(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "ln -sf") {
		t.Error("install script should create nvim symlink")
	}
}
