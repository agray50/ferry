package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteLockFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	lf := DefaultLockFile()
	lf.Nvim.Version = "0.10.0"
	// Languages are now per-profile; set on the default profile
	if prof, ok := lf.Profiles["default"]; ok {
		prof.Languages = []LanguageConfig{
			{Name: "go", Tier: "full"},
			{Name: "python", Tier: "full"},
		}
		lf.Profiles["default"] = prof
	}

	if err := WriteLockFile(lf); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	got, err := ReadLockFile()
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if got.Nvim.Version != "0.10.0" {
		t.Errorf("version: got %q want %q", got.Nvim.Version, "0.10.0")
	}
	if len(got.Profiles["default"].Languages) != 2 {
		t.Errorf("languages: got %d want 2", len(got.Profiles["default"].Languages))
	}
}

func TestReadLockFileMissing(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := ReadLockFile()
	if err == nil {
		t.Fatal("expected error for missing ferry.lock")
	}
}

func TestLockFileHash(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	lf := DefaultLockFile()
	WriteLockFile(lf)

	h1, err := LockFileHash()
	if err != nil {
		t.Fatalf("LockFileHash: %v", err)
	}
	if h1 == "" {
		t.Fatal("expected non-empty hash")
	}

	// same content → same hash
	h2, _ := LockFileHash()
	if h1 != h2 {
		t.Error("hash should be stable for same content")
	}

	// change content → different hash
	lf.Nvim.Version = "changed"
	WriteLockFile(lf)
	h3, _ := LockFileHash()
	if h1 == h3 {
		t.Error("hash should differ after content change")
	}
}

func TestTargetsRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tf, err := ReadTargets()
	if err != nil {
		t.Fatalf("ReadTargets (empty): %v", err)
	}
	if len(tf) != 0 {
		t.Fatal("expected empty TargetsFile for missing file")
	}

	target := Target{Arch: "x86_64", Libc: "glibc", Profile: "default"}
	if err := UpsertTarget("user@myvm", target); err != nil {
		t.Fatalf("UpsertTarget: %v", err)
	}

	got, err := GetTarget("user@myvm")
	if err != nil {
		t.Fatalf("GetTarget: %v", err)
	}
	if got.Arch != "x86_64" {
		t.Errorf("arch: got %q want x86_64", got.Arch)
	}

	// file permissions
	info, _ := os.Stat(filepath.Join(home, ".ferry", "targets.json"))
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %v", info.Mode().Perm())
	}
}

func TestRemoveTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	UpsertTarget("user@host", Target{Arch: "x86_64"})
	RemoveTarget("user@host")

	_, err := GetTarget("user@host")
	if err == nil {
		t.Fatal("expected error after RemoveTarget")
	}

	// no-op on missing
	if err := RemoveTarget("nonexistent"); err != nil {
		t.Errorf("RemoveTarget on missing should not error: %v", err)
	}
}
