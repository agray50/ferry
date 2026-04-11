package registry

import (
	"strings"
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
	langs, err := ResolveLanguages([]config.LanguageConfig{
		{Name: "python", Tier: "full", LSP: "pylsp", RuntimeVersion: "3.11", ExtraPackages: []string{"numpy", "pandas"}},
		{Name: "go", Tier: "full"},
	})
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
	_, err := ResolveLanguages([]config.LanguageConfig{{Name: "cobol", Tier: "full"}})
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
}

func TestResolveLanguagesInvalidLSP(t *testing.T) {
	_, err := ResolveLanguages([]config.LanguageConfig{
		{Name: "python", Tier: "full", LSP: "rust-analyzer"},
	})
	if err == nil {
		t.Fatal("expected error for invalid LSP override")
	}
}

func TestBuildStepKinds(t *testing.T) {
	run := BuildStep{Kind: "run", Value: "echo hello"}
	env := BuildStep{Kind: "env", Value: "PATH=/usr/bin:$PATH"}
	if run.Kind != "run" {
		t.Errorf("run.Kind = %q, want run", run.Kind)
	}
	if env.Kind != "env" {
		t.Errorf("env.Kind = %q, want env", env.Kind)
	}
}

func TestContainerPathFields(t *testing.T) {
	cp := ContainerPath{
		Container:   "/root/.pyenv/versions/{VERSION}/",
		InstallPath: "~/.ferry/runtimes/python-{VERSION}/",
	}
	if !strings.Contains(cp.Container, "{VERSION}") {
		t.Error("ContainerPath.Container should contain {VERSION} placeholder")
	}
	if !strings.Contains(cp.InstallPath, "{VERSION}") {
		t.Error("ContainerPath.InstallPath should contain {VERSION} placeholder")
	}
}

func TestPythonBuildSteps(t *testing.T) {
	l, err := Get("python")
	if err != nil {
		t.Fatalf("Get(python): %v", err)
	}
	if l.Runtime == nil {
		t.Fatal("python Runtime should not be nil")
	}
	if len(l.Runtime.BuildSteps) == 0 {
		t.Error("python Runtime.BuildSteps should not be empty")
	}
	hasRun, hasEnv := false, false
	for _, s := range l.Runtime.BuildSteps {
		if s.Kind == "run" { hasRun = true }
		if s.Kind == "env" { hasEnv = true }
	}
	if !hasRun {
		t.Error("python BuildSteps should contain at least one 'run' step")
	}
	if !hasEnv {
		t.Error("python BuildSteps should contain at least one 'env' step")
	}
}

func TestPythonContainerPaths(t *testing.T) {
	l, _ := Get("python")
	if len(l.Runtime.ContainerPaths) == 0 {
		t.Fatal("python Runtime.ContainerPaths should not be empty")
	}
	cp := l.Runtime.ContainerPaths[0]
	if !strings.Contains(cp.Container, "{VERSION}") {
		t.Errorf("Container path %q should contain {VERSION}", cp.Container)
	}
	if !strings.Contains(cp.InstallPath, "~/.ferry/runtimes/") {
		t.Errorf("InstallPath %q should be under ~/.ferry/runtimes/", cp.InstallPath)
	}
}

func TestPythonShellInit(t *testing.T) {
	l, _ := Get("python")
	if len(l.Runtime.ShellInit) == 0 {
		t.Fatal("python Runtime.ShellInit should not be empty")
	}
	if !strings.Contains(l.Runtime.ShellInit[0], ".ferry/runtimes/") {
		t.Errorf("ShellInit %q should reference ~/.ferry/runtimes/", l.Runtime.ShellInit[0])
	}
}

func TestPythonApproxSize(t *testing.T) {
	l, _ := Get("python")
	if l.ApproxSizeMB <= 0 {
		t.Error("python ApproxSizeMB should be > 0")
	}
	if l.ApproxLSPOnlyMB <= 0 {
		t.Error("python ApproxLSPOnlyMB should be > 0")
	}
	if l.ApproxLSPOnlyMB >= l.ApproxSizeMB {
		t.Error("python LSP-only tier should be smaller than full runtime")
	}
}

func TestMacOSSupported(t *testing.T) {
	python, _ := Get("python")
	if !python.MacOSSupported {
		t.Error("python should be MacOSSupported")
	}
}

func TestJSTSSharedRuntime(t *testing.T) {
	js, _ := Get("javascript")
	ts, _ := Get("typescript")
	if js.Runtime == nil {
		t.Fatal("javascript must have a Runtime")
	}
	if js.Runtime.Manager != "nvm" {
		t.Errorf("javascript runtime manager should be nvm, got %s", js.Runtime.Manager)
	}
	// typescript shares the nvm runtime — its Runtime field must also be non-nil
	if ts.Runtime == nil {
		t.Fatal("typescript must have a Runtime (shares nvm with javascript)")
	}
	if ts.Runtime.Manager != js.Runtime.Manager {
		t.Error("javascript and typescript should share the same Manager (nvm)")
	}
}

func TestAllLanguagesHaveRequiredFields(t *testing.T) {
	languages := []string{
		"python", "javascript", "typescript", "go", "rust",
		"ruby", "java", "c", "cpp", "csharp", "php",
		"kotlin", "scala", "swift", "zig", "elixir", "dart", "r",
		"lua", "bash", "sh",
		"yaml", "json", "markdown", "dockerfile",
	}
	for _, name := range languages {
		t.Run(name, func(t *testing.T) {
			l, err := Get(name)
			if err != nil {
				t.Fatalf("Get(%q): %v", name, err)
			}
			if l.LSP == "" {
				t.Errorf("%s: LSP should not be empty", name)
			}
			// Languages with a runtime must have BuildSteps and ContainerPaths
			if l.Runtime != nil {
				if len(l.Runtime.BuildSteps) == 0 {
					t.Errorf("%s: Runtime.BuildSteps should not be empty", name)
				}
				if len(l.Runtime.ContainerPaths) == 0 {
					t.Errorf("%s: Runtime.ContainerPaths should not be empty", name)
				}
				if len(l.Runtime.ShellInit) == 0 {
					t.Errorf("%s: Runtime.ShellInit should not be empty", name)
				}
			}
		})
	}
}

func TestAllLanguagesHaveSizeEstimates(t *testing.T) {
	for _, l := range All() {
		if l.Runtime != nil && l.ApproxSizeMB == 0 {
			t.Errorf("%s has Runtime but zero ApproxSizeMB", l.Name)
		}
	}
}

func TestResolveFromProfile(t *testing.T) {
	langs := []config.LanguageConfig{
		{Name: "python", Tier: "full", RuntimeVersion: "3.11", LSP: "pylsp"},
		{Name: "go", Tier: "lsp-only"},
	}

	resolved, err := ResolveFromProfile(langs)
	if err != nil {
		t.Fatalf("ResolveFromProfile: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved languages, got %d", len(resolved))
	}

	// Python: full tier, pylsp override
	py := resolved[0]
	if py.Language.Name != "python" {
		t.Errorf("resolved[0].Name = %q, want python", py.Language.Name)
	}
	if py.Runtime == nil {
		t.Fatal("python full tier: Runtime should not be nil")
	}
	if py.Runtime.DefaultVersion != "3.11" {
		t.Errorf("python runtime version = %q, want 3.11", py.Runtime.DefaultVersion)
	}
	if py.EffectiveLSP != "pylsp" {
		t.Errorf("python EffectiveLSP = %q, want pylsp", py.EffectiveLSP)
	}

	// Go: lsp-only tier
	go_ := resolved[1]
	if go_.Runtime != go_.Language.LSPOnlyRuntime {
		t.Error("go lsp-only: Runtime should be LSPOnlyRuntime")
	}
}

func TestResolveFromProfileUnknownLanguage(t *testing.T) {
	_, err := ResolveFromProfile([]config.LanguageConfig{{Name: "cobol", Tier: "full"}})
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
}

func TestResolveFromProfileDeduplicatesNVM(t *testing.T) {
	langs := []config.LanguageConfig{
		{Name: "javascript", Tier: "full"},
		{Name: "typescript", Tier: "full"},
	}
	resolved, err := ResolveFromProfile(langs)
	if err != nil {
		t.Fatalf("ResolveFromProfile: %v", err)
	}
	// Both resolved but typescript has nil Runtime (shares nvm with javascript)
	var runtimeCount int
	for _, r := range resolved {
		if r.Runtime != nil {
			runtimeCount++
		}
	}
	if runtimeCount != 1 {
		t.Errorf("js+ts should produce 1 runtime component, got %d", runtimeCount)
	}
}

func TestPresetsNonEmpty(t *testing.T) {
	presets := Presets()
	if len(presets) == 0 {
		t.Fatal("Presets() should return at least one preset")
	}
	for _, p := range presets {
		if p.Name == "" {
			t.Error("preset Name should not be empty")
		}
		if p.Description == "" {
			t.Errorf("preset %q: Description should not be empty", p.Name)
		}
		if p.EstimatedMB <= 0 {
			t.Errorf("preset %q: EstimatedMB should be > 0", p.Name)
		}
	}
}

func TestPresetsLanguagesAreValid(t *testing.T) {
	for _, p := range Presets() {
		for _, lc := range p.Profile.Languages {
			if _, err := Get(lc.Name); err != nil {
				t.Errorf("preset %q: unknown language %q", p.Name, lc.Name)
			}
		}
	}
}
