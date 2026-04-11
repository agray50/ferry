package cmd

import (
	"testing"
)

// TestInitCmdExists verifies the init subcommand is registered.
func TestInitCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("init command not registered on rootCmd")
	}
}
