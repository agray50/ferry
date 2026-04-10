package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
}

func TestGenerateAndLoadKeypair(t *testing.T) {
	withTempHome(t)

	if err := GenerateKeypair(); err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	// second call should error
	if err := GenerateKeypair(); err == nil {
		t.Fatal("expected error on second GenerateKeypair")
	}

	if !KeyExists() {
		t.Fatal("KeyExists should return true")
	}

	_, err := LoadPrivateKey()
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	pub, err := LoadPublicKey()
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}
	if pub == "" {
		t.Fatal("public key should not be empty")
	}

	// check permissions
	info, _ := os.Stat(os.Getenv("HOME") + "/.ferry/key.txt")
	if info.Mode().Perm() != 0600 {
		t.Errorf("key.txt permissions: got %v want 0600", info.Mode().Perm())
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	withTempHome(t)
	GenerateKeypair()

	plaintext := []byte("super secret data 12345")

	encrypted, err := EncryptBytes(plaintext)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}

	identity, err := LoadPrivateKey()
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	decrypted, err := DecryptBytes(encrypted, identity)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("roundtrip failed: got %q want %q", decrypted, plaintext)
	}
}

func TestScanFile(t *testing.T) {
	f, _ := os.CreateTemp("", "secrets-*.txt")
	defer os.Remove(f.Name())

	f.WriteString("export GITHUB_TOKEN=ghp_abc123\n")
	f.WriteString("# export SKIP_TOKEN=should_be_ignored\n")
	f.WriteString("password = hunter2\n")
	f.WriteString("normal line\n")
	f.WriteString("export MY_API_KEY=somekey\n")
	f.Close()

	matches, err := ScanFile(f.Name())
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d: %v", len(matches), matches)
	}
	// comment should not match
	for _, m := range matches {
		if m.Line == 2 {
			t.Error("commented line should not match")
		}
	}
}

func TestRedactPreview(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"export GITHUB_TOKEN=ghp_abc123", "export GITHUB_TOKEN=***"},
		{"password = hunter2", "password =***"},
		{"no equals here", "no equals here"},
	}
	for _, tc := range cases {
		got := RedactPreview(tc.input)
		if got != tc.want {
			t.Errorf("RedactPreview(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEncryptFile(t *testing.T) {
	withTempHome(t)
	GenerateKeypair()

	src := filepath.Join(t.TempDir(), "secret.txt")
	dst := src + ".age"
	os.WriteFile(src, []byte("secret content"), 0644)

	if err := EncryptFile(src, dst); err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if len(data) == 0 {
		t.Fatal("encrypted file should not be empty")
	}
	if string(data) == "secret content" {
		t.Fatal("encrypted file should not be plaintext")
	}
}
