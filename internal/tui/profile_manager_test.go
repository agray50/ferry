package tui

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/discovery"
)

func TestBuildProfileRows_empty(t *testing.T) {
	lf := &config.LockFile{Profiles: map[string]config.ProfileConfig{}}
	rows := buildProfileRows(lf, config.TargetsFile{}, nil)
	if len(rows) != 0 {
		t.Errorf("want 0 rows, got %d", len(rows))
	}
}

func TestBuildProfileRows_languageSummary(t *testing.T) {
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"backend": {
				Languages: []config.LanguageConfig{
					{Name: "go", Tier: "full"},
					{Name: "python", Tier: "full"},
					{Name: "rust", Tier: "lsp-only"},
				},
			},
		},
	}
	rows := buildProfileRows(lf, config.TargetsFile{}, nil)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	summary := rows[0].LanguageSummary
	if !strings.Contains(summary, "go") {
		t.Errorf("summary missing 'go': %s", summary)
	}
	if !strings.Contains(summary, "python") {
		t.Errorf("summary missing 'python': %s", summary)
	}
	if !strings.Contains(summary, "rust") {
		t.Errorf("summary missing 'rust': %s", summary)
	}
	if !strings.Contains(summary, "(LSP)") {
		t.Errorf("summary missing '(LSP)' for lsp-only lang: %s", summary)
	}
}

func TestBuildProfileRows_targetStatus(t *testing.T) {
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"backend": {},
		},
	}
	targets := config.TargetsFile{
		"prod-vm": {Profile: "backend", BundleHash: "abc123"},
		"dev-vm":  {Profile: "backend", BundleHash: "def456"},
	}
	localHash := "abc123"

	rows := buildProfileRows(lf, targets, &localHash)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	var prodStatus, devStatus string
	for _, tr := range rows[0].Targets {
		switch tr.Host {
		case "prod-vm":
			prodStatus = tr.Status
		case "dev-vm":
			devStatus = tr.Status
		}
	}
	if prodStatus != "up-to-date" {
		t.Errorf("prod-vm: want up-to-date, got %s", prodStatus)
	}
	if devStatus != "stale" {
		t.Errorf("dev-vm: want stale, got %s", devStatus)
	}
}

func TestBuildProfileRows_noTargets(t *testing.T) {
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"frontend": {},
		},
	}
	rows := buildProfileRows(lf, config.TargetsFile{}, nil)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if len(rows[0].Targets) != 0 {
		t.Errorf("want 0 targets, got %d", len(rows[0].Targets))
	}
}

func TestProfileManagerView_containsProfiles(t *testing.T) {
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"backend":  {Languages: []config.LanguageConfig{{Name: "go", Tier: "full"}}},
			"frontend": {Languages: []config.LanguageConfig{{Name: "typescript", Tier: "full"}}},
		},
	}
	m := newProfileManagerModel(lf, config.TargetsFile{}, nil, []discovery.SSHHost{}, false)
	view := m.View()
	if !strings.Contains(view, "backend") {
		t.Errorf("view missing 'backend':\n%s", view)
	}
	if !strings.Contains(view, "frontend") {
		t.Errorf("view missing 'frontend':\n%s", view)
	}
}

func TestProfileManagerView_readonlyHidesEditHints(t *testing.T) {
	lf := &config.LockFile{Profiles: map[string]config.ProfileConfig{"p": {}}}
	m := newProfileManagerModel(lf, config.TargetsFile{}, nil, nil, true)
	view := m.View()
	if strings.Contains(view, "e: edit") {
		t.Error("read-only view should not show 'e: edit'")
	}
}
