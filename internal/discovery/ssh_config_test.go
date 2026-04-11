package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// overrideSSHConfigPath points ssh_config.go at a temp file for tests.
func overrideSSHConfigPath(t *testing.T, content string) (path string, restore func()) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	old := sshConfigPathOverride
	sshConfigPathOverride = p
	return p, func() { sshConfigPathOverride = old }
}

func TestParseSSHConfig_basicHost(t *testing.T) {
	_, restore := overrideSSHConfigPath(t, `
Host dev-vm
    HostName 10.0.1.5
    User ubuntu
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
`)
	defer restore()

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 {
		t.Fatalf("want 1 host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.Alias != "dev-vm" {
		t.Errorf("alias: want dev-vm, got %s", h.Alias)
	}
	if h.Hostname != "10.0.1.5" {
		t.Errorf("hostname: want 10.0.1.5, got %s", h.Hostname)
	}
	if h.User != "ubuntu" {
		t.Errorf("user: want ubuntu, got %s", h.User)
	}
	if h.Port != 2222 {
		t.Errorf("port: want 2222, got %d", h.Port)
	}
	if h.FerryManaged {
		t.Error("should not be ferry managed")
	}
}

func TestParseSSHConfig_ferryManaged(t *testing.T) {
	_, restore := overrideSSHConfigPath(t, `
Host user-host
    HostName 1.2.3.4
    User alice

# ferry managed
Host ferry-host
    HostName 5.6.7.8
    User bob
`)
	defer restore()

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Fatalf("want 2 hosts, got %d", len(hosts))
	}
	if hosts[0].FerryManaged {
		t.Error("first host should not be ferry managed")
	}
	if !hosts[1].FerryManaged {
		t.Error("second host should be ferry managed")
	}
}

func TestParseSSHConfig_missing(t *testing.T) {
	old := sshConfigPathOverride
	sshConfigPathOverride = "/tmp/nonexistent-ferry-test-ssh-config"
	defer func() { sshConfigPathOverride = old }()

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Errorf("want 0 hosts for missing file, got %d", len(hosts))
	}
}

func TestAppendSSHHost(t *testing.T) {
	path, restore := overrideSSHConfigPath(t, "")
	defer restore()

	h := SSHHost{
		Alias:        "test-vm",
		Hostname:     "192.168.1.10",
		User:         "ubuntu",
		Port:         22,
		IdentityFile: "~/.ssh/id_ed25519",
	}
	if err := AppendSSHHost(h); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.Contains(s, "# ferry managed") {
		t.Error("missing # ferry managed tag")
	}
	if !strings.Contains(s, "Host test-vm") {
		t.Error("missing Host line")
	}
	if !strings.Contains(s, "HostName 192.168.1.10") {
		t.Error("missing HostName line")
	}

	// round-trip: parse should return the appended host as ferry managed
	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || !hosts[0].FerryManaged {
		t.Errorf("round-trip: want 1 ferry-managed host, got %+v", hosts)
	}
}

func TestRemoveSSHHost_ferryManaged(t *testing.T) {
	_, restore := overrideSSHConfigPath(t, `
Host user-host
    HostName 1.2.3.4
    User alice

# ferry managed
Host ferry-host
    HostName 5.6.7.8
    User bob
`)
	defer restore()

	if err := RemoveSSHHost("ferry-host"); err != nil {
		t.Fatal(err)
	}

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0].Alias != "user-host" {
		t.Errorf("want only user-host remaining, got %+v", hosts)
	}
}

func TestRemoveSSHHost_notFerryManaged(t *testing.T) {
	_, restore := overrideSSHConfigPath(t, `
Host user-host
    HostName 1.2.3.4
    User alice
`)
	defer restore()

	err := RemoveSSHHost("user-host")
	if err == nil {
		t.Error("want error when removing non-ferry-managed host, got nil")
	}
}

func TestRemoveSSHHost_missing(t *testing.T) {
	_, restore := overrideSSHConfigPath(t, `
Host user-host
    HostName 1.2.3.4
`)
	defer restore()

	// removing a non-existent alias is a no-op
	if err := RemoveSSHHost("does-not-exist"); err != nil {
		t.Errorf("removing non-existent host should be no-op, got: %v", err)
	}
}
