package bootstrap

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/ssh"
	"github.com/anthropics/ferry/internal/store"
)

// ValidationResult holds the outcome of post-install validation.
type ValidationResult struct {
	Items    []ValidationItem
	Warnings int
	Errors   int
}

// ValidationItem is a single validated component.
type ValidationItem struct {
	Name    string
	Status  string // "ok" | "warning" | "error"
	Version string
	Message string
}

// Validate checks that all manifest components are correctly installed.
func Validate(c *ssh.Client, m *store.Manifest) ValidationResult {
	var result ValidationResult

	for _, comp := range m.Components {
		item := validateComponent(c, comp)
		result.Items = append(result.Items, item)
		switch item.Status {
		case "warning":
			result.Warnings++
		case "error":
			result.Errors++
		}
	}
	return result
}

func validateComponent(c *ssh.Client, comp store.Component) ValidationItem {
	item := ValidationItem{Name: comp.ID}

	switch {
	case comp.ID == "nvim-binary":
		stdout, _, code, err := c.Run("nvim --version 2>&1 | head -1")
		if err != nil || code != 0 {
			item.Status = "error"
			item.Message = "nvim not found after install"
			return item
		}
		item.Status = "ok"
		item.Version = strings.TrimSpace(strings.TrimPrefix(stdout, "NVIM "))

	case strings.HasPrefix(comp.ID, "lsp/"):
		binary := strings.TrimPrefix(comp.ID, "lsp/")
		stdout, _, code, err := c.Run(fmt.Sprintf("which %s && %s --version 2>&1 | head -1", binary, binary))
		if err != nil || code != 0 {
			item.Status = "error"
			item.Message = fmt.Sprintf("%s not found", binary)
			return item
		}
		item.Status = "ok"
		item.Version = strings.TrimSpace(stdout)

	case strings.HasPrefix(comp.ID, "cli/"):
		binary := strings.TrimPrefix(comp.ID, "cli/")
		stdout, _, code, err := c.Run(fmt.Sprintf("which %s && %s --version 2>&1 | head -1", binary, binary))
		if err != nil || code != 0 {
			item.Status = "warning"
			item.Message = fmt.Sprintf("%s not found (system requirement)", binary)
			return item
		}
		item.Status = "ok"
		item.Version = strings.TrimSpace(stdout)

	case strings.HasPrefix(comp.ID, "runtime/python"):
		stdout, _, code, err := c.Run("$HOME/.pyenv/bin/python --version 2>&1")
		if err != nil || code != 0 {
			item.Status = "error"
			item.Message = "python runtime not found"
			return item
		}
		item.Status = "ok"
		item.Version = strings.TrimSpace(stdout) + " via pyenv"

	case strings.HasPrefix(comp.ID, "runtime/node"):
		stdout, _, code, err := c.Run(". $HOME/.nvm/nvm.sh && node --version 2>&1")
		if err != nil || code != 0 {
			item.Status = "error"
			item.Message = "node runtime not found"
			return item
		}
		item.Status = "ok"
		item.Version = strings.TrimSpace(stdout) + " via nvm"

	case strings.HasPrefix(comp.ID, "lazy/"):
		pluginName := strings.TrimPrefix(comp.ID, "lazy/")
		exists, err := c.FileExists(fmt.Sprintf("$HOME/.local/share/nvim/lazy/%s", pluginName))
		if err != nil || !exists {
			item.Status = "error"
			item.Message = "plugin directory missing"
			return item
		}
		item.Status = "ok"
		item.Version = "installed"

	default:
		exists, err := c.FileExists(comp.InstallPath)
		if err != nil || !exists {
			item.Status = "warning"
			item.Message = "install path missing"
			return item
		}
		item.Status = "ok"
	}

	return item
}

// PrintValidation prints a formatted validation table.
func PrintValidation(result ValidationResult) {
	fmt.Printf("── VALIDATION ───────────────────────────────────────\n\n")
	for _, item := range result.Items {
		var icon string
		switch item.Status {
		case "ok":
			icon = "✓"
		case "warning":
			icon = "✗"
		case "error":
			icon = "✗"
		}
		ver := item.Version
		if ver == "" {
			ver = item.Message
		}
		fmt.Printf("  %-35s %s  %s\n", item.Name, icon, ver)
	}

	if result.Warnings > 0 {
		fmt.Printf("\n  %d warning(s) (system requirements — install manually if needed)\n", result.Warnings)
	}
	if result.Errors > 0 {
		fmt.Printf("\n  %d error(s)\n", result.Errors)
	}
}
