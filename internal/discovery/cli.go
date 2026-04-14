package discovery

import "os/exec"

// CLITool holds information about a discovered CLI tool.
type CLITool struct {
	Name    string
	Path    string
	Version string
	Static  bool
}

// KnownCLITools is the list of CLI tools ferry can bundle.
var KnownCLITools = []string{
	// search & navigation
	"rg",       // ripgrep — fast grep
	"fd",       // find alternative
	"fzf",      // fuzzy finder
	"bat",      // cat with syntax highlighting
	"eza",      // ls alternative (exa successor)
	"exa",      // ls alternative
	"lsd",      // ls deluxe
	"zoxide",   // smarter cd
	"broot",    // interactive tree navigator
	"navi",     // interactive cheatsheet

	// data & text processing
	"jq",       // JSON processor
	"yq",       // YAML/JSON processor
	"gron",     // flatten JSON to greppable form
	"fx",       // interactive JSON viewer
	"xsv",      // CSV toolkit
	"miller",   // mlr — CSV/JSON/TSV processor

	// shell helpers
	"shfmt",      // shell formatter
	"shellcheck", // shell linter
	"direnv",     // per-directory env vars
	"zellij",     // terminal multiplexer

	// git tools
	"gh",         // GitHub CLI
	"delta",      // better git diff
	"lazygit",    // TUI git client
	"gitui",      // TUI git client
	"tig",        // ncurses git browser

	// docker & containers
	"hadolint",   // Dockerfile linter
	"dive",       // Docker image explorer
	"ctop",       // container metrics TUI

	// system monitoring
	"htop",       // process viewer
	"btop",       // resource monitor
	"dust",       // disk usage
	"duf",        // df alternative
	"procs",      // ps alternative
	"bottom",     // btm — system monitor

	// file management
	"lf",         // terminal file manager
	"ranger",     // terminal file manager
	"nnn",        // terminal file manager
	"yazi",       // terminal file manager

	// network
	"httpie",     // http — HTTP client
	"xh",         // HTTPie-compatible HTTP client
	"curlie",     // curl + httpie
	"dog",        // DNS client

	// build & task runners
	"just",       // command runner (Justfile)
	"make",       // classic build tool
	"task",       // go-task — modern make alternative

	// editor helpers
	"stylua",     // Lua formatter
	"prettier",   // JS/TS/CSS/HTML formatter
}

// DiscoverCLITools scans PATH for known CLI tools.
func DiscoverCLITools() []CLITool {
	var tools []CLITool
	seen := make(map[string]bool)
	for _, name := range KnownCLITools {
		if seen[name] {
			continue
		}
		seen[name] = true
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		ti := probeTool(name, path, "direct")
		tools = append(tools, CLITool{
			Name:    name,
			Path:    path,
			Version: ti.Version,
			Static:  ti.Static,
		})
	}
	return tools
}
