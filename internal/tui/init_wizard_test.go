package tui

import (
	"testing"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

func TestBuildSizeSummary_empty(t *testing.T) {
	prof := config.ProfileConfig{}
	summary := buildSizeSummary(prof)
	if summary.TotalMB != 0 {
		t.Errorf("empty profile: want 0MB, got %d", summary.TotalMB)
	}
}

func TestBuildSizeSummary_languages(t *testing.T) {
	// temporarily inject test registry entries
	oldAll := registryAllFunc
	registryAllFunc = func() []registry.Language {
		return []registry.Language{
			{Name: "go", ApproxSizeMB: 130, ApproxLSPOnlyMB: 20},
			{Name: "python", ApproxSizeMB: 150, ApproxLSPOnlyMB: 25},
		}
	}
	defer func() { registryAllFunc = oldAll }()

	prof := config.ProfileConfig{
		Languages: []config.LanguageConfig{
			{Name: "go", Tier: "full"},
			{Name: "python", Tier: "lsp-only"},
		},
	}
	summary := buildSizeSummary(prof)
	// go=130 + python lsp-only=25 = 155
	if summary.TotalMB != 155 {
		t.Errorf("want 155MB, got %d", summary.TotalMB)
	}
	if len(summary.Lines) != 2 {
		t.Errorf("want 2 lines, got %d", len(summary.Lines))
	}
}

func TestBuildSizeSummary_transferTime(t *testing.T) {
	s := SizeSummary{TotalMB: 400}
	at10 := transferTime(s.TotalMB, 10)
	at100 := transferTime(s.TotalMB, 100)
	if at10 <= at100 {
		t.Errorf("10Mbps should be slower than 100Mbps: %s vs %s", at10, at100)
	}
}

func TestTransferTime_edgeCases(t *testing.T) {
	tests := []struct {
		totalMB int
		mbps    int
		want    string
	}{
		{0, 10, "n/a"},
		{400, 0, "n/a"},
		{1, 100, "< 1s"},   // integer truncation: 8/100 = 0 seconds
		{400, 10, "~5m 20s"},
		{60, 10, "~48s"},
	}
	for _, tt := range tests {
		got := transferTime(tt.totalMB, tt.mbps)
		if got != tt.want {
			t.Errorf("transferTime(%d, %d) = %q, want %q", tt.totalMB, tt.mbps, got, tt.want)
		}
	}
}
