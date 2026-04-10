package discovery

import "os/exec"

// CLITool holds information about a discovered static CLI tool.
type CLITool struct {
	Name    string
	Path    string
	Version string
	Static  bool
}

// KnownCLITools is the list of static CLI tools ferry manages.
var KnownCLITools = []string{
	"rg", "fzf", "zoxide", "jq", "shfmt",
	"shellcheck", "stylua", "hadolint",
}

// DiscoverCLITools scans PATH for known static CLI tools.
func DiscoverCLITools() []CLITool {
	var tools []CLITool
	for _, name := range KnownCLITools {
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
