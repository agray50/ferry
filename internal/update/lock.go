package update

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anthropics/ferry/internal/ssh"
)

const lockPath = "$HOME/.ferry/.update.lock"

// AcquireLock atomically writes the remote PID to the update lock file.
// Uses POSIX set -C (noclobber) so that the create-and-write is atomic —
// two concurrent ferry processes cannot both acquire the lock.
func AcquireLock(c *ssh.Client) error {
	// set -C causes > to fail if the file already exists (POSIX noclobber).
	// Run in a subshell so it does not affect the calling shell's options.
	_, _, code, err := c.Run(`( set -C; echo $$ > ` + lockPath + ` ) 2>/dev/null`)
	if err != nil {
		return fmt.Errorf("acquiring update lock: %w", err)
	}
	if code != 0 {
		pid, _, _ := LockInfo(c)
		return fmt.Errorf("update lock exists (pid %d)\n  if stale: ssh target \"rm ~/.ferry/.update.lock\"", pid)
	}
	return nil
}

// ReleaseLock removes ~/.ferry/.update.lock on target.
func ReleaseLock(c *ssh.Client) {
	c.Run("rm -f " + lockPath)
}

// LockInfo reads the lock file and returns the PID if present.
func LockInfo(c *ssh.Client) (pid int, exists bool, err error) {
	data, err := c.DownloadBytes(lockPath)
	if err != nil {
		return 0, false, nil
	}
	pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
	return pid, true, err
}
