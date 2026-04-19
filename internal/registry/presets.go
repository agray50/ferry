package registry

import "github.com/anthropics/ferry/internal/config"

// Preset is a named profile template shown in the first-run TUI wizard.
type Preset struct {
	Name        string
	Description string
	EstimatedMB int
	Profile     config.ProfileConfig
}

// Presets returns the built-in profile presets for the first-run wizard.
// These are starting points — all fields can be modified in the profile editor.
func Presets() []Preset {
	return []Preset{
		{
			Name:        "backend",
			Description: "Go · Python · Rust (LSP only)",
			EstimatedMB: 380,
			Profile: config.ProfileConfig{
				Description:  "backend development",
				IncludeShell: true,
				CLI:          []string{"rg", "fzf", "jq", "zoxide"},
				Languages: []config.LanguageConfig{
					{Name: "go", Tier: "full"},
					{Name: "python", Tier: "full"},
					{Name: "rust", Tier: "lsp-only"},
				},
			},
		},
		{
			Name:        "frontend",
			Description: "TypeScript · JavaScript",
			EstimatedMB: 200,
			Profile: config.ProfileConfig{
				Description:  "frontend development",
				IncludeShell: true,
				CLI:          []string{"rg", "fzf", "jq"},
				Languages: []config.LanguageConfig{
					{Name: "typescript", Tier: "full"},
					{Name: "javascript", Tier: "full"},
				},
			},
		},
		{
			Name:        "fullstack",
			Description: "Go · TypeScript · Python",
			EstimatedMB: 360,
			Profile: config.ProfileConfig{
				Description:  "full-stack development",
				IncludeShell: true,
				CLI:          []string{"rg", "fzf", "jq", "zoxide"},
				Languages: []config.LanguageConfig{
					{Name: "go", Tier: "full"},
					{Name: "typescript", Tier: "full"},
					{Name: "python", Tier: "full"},
				},
			},
		},
		{
			Name:        "data",
			Description: "Python · Go for data pipelines",
			EstimatedMB: 280,
			Profile: config.ProfileConfig{
				Description:  "data science / pipelines",
				IncludeShell: true,
				CLI:          []string{"rg", "jq", "yq"},
				Languages: []config.LanguageConfig{
					{Name: "python", Tier: "full"},
					{Name: "go", Tier: "full"},
				},
			},
		},
		{
			Name:        "minimal",
			Description: "nvim + shell only, no languages",
			EstimatedMB: 80,
			Profile: config.ProfileConfig{
				Description:  "minimal nvim environment",
				IncludeShell: true,
				CLI:          []string{"rg", "fzf"},
			},
		},
	}
}
