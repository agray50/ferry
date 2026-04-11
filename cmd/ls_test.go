package cmd

import (
	"testing"
)

func TestLsCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ls command not registered on rootCmd")
	}
}
