package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/ssh"
)

var connectCmd = &cobra.Command{
	Use:   "connect <target>",
	Short: "SSH into a target and attach to a tmux session",
	Args:  cobra.ExactArgs(1),
	RunE:  runConnect,
}

func init() {
	connectCmd.Flags().String("session", "", "tmux session name to attach to")
}

func runConnect(cmd *cobra.Command, args []string) error {
	target := args[0]
	sessionFlag, _ := cmd.Flags().GetString("session")

	// verify target is bootstrapped
	if _, err := config.GetTarget(target); err != nil {
		return err
	}

	// build session name
	sessionName := sessionFlag
	if sessionName == "" {
		sessionName = "ferry-" + sanitizeTarget(target)
	}

	// check for existing tmux session
	c, err := ssh.Connect(target)
	if err != nil {
		return err
	}

	_, _, code, _ := c.Run(fmt.Sprintf("tmux has-session -t %s 2>/dev/null", sessionName))
	c.Close()

	var tmuxCmd string
	if code == 0 {
		tmuxCmd = fmt.Sprintf("tmux attach-session -t %s", sessionName)
	} else {
		tmuxCmd = fmt.Sprintf("tmux new-session -s %s", sessionName)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH")
	}

	sshArgs := []string{"ssh", "-t", target, tmuxCmd}
	return syscall.Exec(sshBin, sshArgs, os.Environ())
}

func sanitizeTarget(target string) string {
	return strings.NewReplacer("@", "-", ".", "-").Replace(target)
}
