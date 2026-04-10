package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client wraps an SSH connection.
type Client struct {
	host        string
	user        string
	conn        *ssh.Client
	agentClient agent.ExtendedAgent
	done        chan struct{}
}

func buildAuthMethods() ([]ssh.AuthMethod, agent.ExtendedAgent) {
	var methods []ssh.AuthMethod
	var agentClient agent.ExtendedAgent

	// SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			agentClient = agent.NewClient(conn)
			methods = append(methods, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}

	// Key files — fallback when agent is unavailable
	home, _ := os.UserHomeDir()
	for _, name := range []string{"id_ed25519", "id_rsa"} {
		path := filepath.Join(home, ".ssh", name)
		if data, err := os.ReadFile(path); err == nil {
			if signer, err := ssh.ParsePrivateKey(data); err == nil {
				methods = append(methods, ssh.PublicKeys(signer))
			}
		}
	}

	return methods, agentClient
}

func buildHostKeyCallback(host string) (ssh.HostKeyCallback, error) {
	home, _ := os.UserHomeDir()
	khPath := filepath.Join(home, ".ssh", "known_hosts")

	cb, err := knownhosts.New(khPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"~/.ssh/known_hosts not found\n  add host key first:\n  ssh-keyscan -H %s >> ~/.ssh/known_hosts",
				host,
			)
		}
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := cb(hostname, remote, key)
		if err != nil {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
				return fmt.Errorf(
					"host %s not in known_hosts\n  add it with:\n  ssh-keyscan -H %s >> ~/.ssh/known_hosts",
					host, host,
				)
			}
		}
		return err
	}, nil
}

func dial(target string, agentForward bool) (*Client, error) {
	pt, err := ParseTarget(target)
	if err != nil {
		return nil, err
	}

	methods, agentClient := buildAuthMethods()

	cb, err := buildHostKeyCallback(pt.Host)
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            pt.User,
		Auth:            methods,
		HostKeyCallback: cb,
		Timeout:         15 * time.Second,
	}

	conn, err := ssh.Dial("tcp", pt.Addr(), cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", pt.Addr(), err)
	}

	c := &Client{
		host: pt.String(),
		user: pt.User,
		conn: conn,
		done: make(chan struct{}),
	}

	if agentForward && agentClient != nil {
		c.agentClient = agentClient
		// Register connection-level agent forwarding handler.
		// Sessions that want forwarding must also call agent.RequestAgentForwarding.
		agent.ForwardToAgent(conn, agentClient)
	}

	// Keepalive — stopped when Close() is called.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				conn.SendRequest("keepalive@openssh.com", true, nil)
			case <-c.done:
				return
			}
		}
	}()

	return c, nil
}

// Connect opens an SSH connection to user@host.
// Uses SSH agent if available (SSH_AUTH_SOCK), falls back to ~/.ssh/id_ed25519 then id_rsa.
func Connect(target string) (*Client, error) {
	return dial(target, false)
}

// ConnectWithAgent opens an SSH connection with agent forwarding enabled.
func ConnectWithAgent(target string) (*Client, error) {
	return dial(target, true)
}

// Close closes the SSH connection and stops background goroutines.
func (c *Client) Close() error {
	close(c.done)
	return c.conn.Close()
}

// newSession opens a new SSH session, enabling agent forwarding if configured.
func (c *Client) newSession() (*ssh.Session, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	if c.agentClient != nil {
		// Best-effort — server may not support it.
		_ = agent.RequestAgentForwarding(sess)
	}
	return sess, nil
}

// Run executes a command on the remote host.
// Returns stdout, stderr, exit code.
func (c *Client) Run(cmd string) (stdout string, stderr string, exitCode int, err error) {
	sess, err := c.newSession()
	if err != nil {
		return "", "", -1, err
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	runErr := sess.Run(cmd)
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			return stdout, stderr, exitErr.ExitStatus(), nil
		}
		return stdout, stderr, -1, runErr
	}
	return stdout, stderr, 0, nil
}

// RunWithEnv executes a command with additional environment variables set via
// SSH Setenv requests. Returns an error if the server rejects any variable —
// add AcceptEnv directives to the remote sshd_config, or use RunWithStdin
// to deliver sensitive values securely.
func (c *Client) RunWithEnv(cmd string, env map[string]string) (string, string, int, error) {
	sess, err := c.newSession()
	if err != nil {
		return "", "", -1, err
	}
	defer sess.Close()

	for k, v := range env {
		if err := sess.Setenv(k, v); err != nil {
			return "", "", -1, fmt.Errorf(
				"server rejected env var %q — add 'AcceptEnv %s' to remote sshd_config, or use RunWithStdin for sensitive values",
				k, k,
			)
		}
	}

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	runErr := sess.Run(cmd)
	stdout := outBuf.String()
	stderr := errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			return stdout, stderr, exitErr.ExitStatus(), nil
		}
		return stdout, stderr, -1, runErr
	}
	return stdout, stderr, 0, nil
}

// RunWithStdin executes a command with stdinData piped to stdin.
// Use this to pass sensitive values (e.g. age private keys) without
// exposing them in process listings.
func (c *Client) RunWithStdin(cmd string, stdinData []byte) (string, string, int, error) {
	sess, err := c.newSession()
	if err != nil {
		return "", "", -1, err
	}
	defer sess.Close()

	sess.Stdin = bytes.NewReader(stdinData)
	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	runErr := sess.Run(cmd)
	stdout := outBuf.String()
	stderr := errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			return stdout, stderr, exitErr.ExitStatus(), nil
		}
		return stdout, stderr, -1, runErr
	}
	return stdout, stderr, 0, nil
}

// Upload copies a local file to a remote path, preserving permissions.
func (c *Client) Upload(localPath string, remotePath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("Upload: %q is a directory", localPath)
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return c.UploadBytes(data, remotePath, info.Mode())
}

// UploadBytes writes bytes to a remote path using the SCP protocol.
// Creates parent directories if needed. Preserves the given file mode.
func (c *Client) UploadBytes(data []byte, remotePath string, mode os.FileMode) error {
	if err := c.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return err
	}

	sess, err := c.newSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	filename := filepath.Base(remotePath)
	dir := filepath.Dir(remotePath)

	if err := sess.Start(fmt.Sprintf("scp -qt %s", shellQuote(dir))); err != nil {
		return fmt.Errorf("UploadBytes scp start: %w", err)
	}

	// readACK reads and validates the one-byte SCP acknowledgement.
	readACK := func() error {
		buf := make([]byte, 1)
		if _, err := io.ReadFull(stdoutPipe, buf); err != nil {
			return fmt.Errorf("scp: reading ack: %w", err)
		}
		if buf[0] != 0 {
			return fmt.Errorf("scp: server error response (%d)", buf[0])
		}
		return nil
	}

	// 1. Initial ACK from server (server is ready)
	if err := readACK(); err != nil {
		return err
	}
	// 2. Send file header
	if _, err := fmt.Fprintf(stdinPipe, "C%04o %d %s\n", mode, len(data), filename); err != nil {
		return err
	}
	// 3. ACK for header
	if err := readACK(); err != nil {
		return err
	}
	// 4. Send file data
	if _, err := stdinPipe.Write(data); err != nil {
		return err
	}
	// 5. End-of-file marker
	if _, err := stdinPipe.Write([]byte{0}); err != nil {
		return err
	}
	// 6. Final ACK
	if err := readACK(); err != nil {
		return err
	}

	stdinPipe.Close()
	return sess.Wait()
}

// Download copies a remote file to a local path.
func (c *Client) Download(remotePath string, localPath string) error {
	data, err := c.DownloadBytes(remotePath)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}

// DownloadBytes reads a remote file and returns its contents.
// Returns error if the file does not exist.
func (c *Client) DownloadBytes(remotePath string) ([]byte, error) {
	stdout, _, code, err := c.Run(fmt.Sprintf("cat %s", shellQuote(remotePath)))
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("remote file not found: %s", remotePath)
	}
	return []byte(stdout), nil
}

// FileExists returns true if a path exists on the remote host.
func (c *Client) FileExists(remotePath string) (bool, error) {
	_, _, code, err := c.Run(fmt.Sprintf("test -e %s", shellQuote(remotePath)))
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

// MkdirAll creates a directory and all parents on the remote host.
func (c *Client) MkdirAll(remotePath string) error {
	_, _, _, err := c.Run(fmt.Sprintf("mkdir -p %s", shellQuote(remotePath)))
	return err
}

// StreamUpload streams data to a remote path over a single SSH channel.
// Calls onProgress with cumulative bytes written after each chunk.
func (c *Client) StreamUpload(data []byte, remotePath string, onProgress func(written int64)) error {
	if err := c.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return err
	}

	sess, err := c.newSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	if err := sess.Start(fmt.Sprintf("cat > %s", shellQuote(remotePath))); err != nil {
		return err
	}

	const chunkSize = 32 * 1024
	var written int64
	r := bytes.NewReader(data)
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, werr := stdinPipe.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	stdinPipe.Close()
	return sess.Wait()
}

// Ping attempts a connection and runs `echo ok`.
// Returns error if the host is unreachable.
func Ping(target string) error {
	c, err := Connect(target)
	if err != nil {
		return err
	}
	defer c.Close()
	_, _, code, err := c.Run("echo ok")
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("ping failed with exit code %d", code)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
