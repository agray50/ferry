package crypto

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/anthropics/ferry/internal/config"
)

// GenerateKeypair generates a new age keypair.
// Writes private key to ~/.ferry/key.txt (0600) and public key to ~/.ferry/key.pub (0644).
// Returns error if files already exist.
func GenerateKeypair() error {
	privPath := config.KeyFile()
	pubPath := config.PubKeyFile()

	if _, err := os.Stat(privPath); err == nil {
		return fmt.Errorf("age key already exists at %s", privPath)
	}

	if err := config.EnsureFerryDir(); err != nil {
		return err
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("generating age keypair: %w", err)
	}

	if err := os.WriteFile(privPath, []byte(identity.String()+"\n"), 0600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	if err := os.WriteFile(pubPath, []byte(identity.Recipient().String()+"\n"), 0644); err != nil {
		os.Remove(privPath)
		return fmt.Errorf("writing public key: %w", err)
	}

	return nil
}

// LoadPrivateKey reads the age private key from ~/.ferry/key.txt.
func LoadPrivateKey() (age.Identity, error) {
	data, err := os.ReadFile(config.KeyFile())
	if err != nil {
		return nil, fmt.Errorf("reading age private key: %w", err)
	}
	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing age identity: %w", err)
	}
	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found in key file")
	}
	return identities[0], nil
}

// LoadPublicKey reads the age public key from ~/.ferry/key.pub.
func LoadPublicKey() (string, error) {
	data, err := os.ReadFile(config.PubKeyFile())
	if err != nil {
		return "", fmt.Errorf("reading age public key: %w", err)
	}
	return string(bytes.TrimSpace(data)), nil
}

// KeyExists returns true if ~/.ferry/key.txt exists.
func KeyExists() bool {
	_, err := os.Stat(config.KeyFile())
	return err == nil
}

// EncryptFile encrypts a file at srcPath using the public key.
// Writes encrypted output to dstPath.
func EncryptFile(srcPath string, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcPath, err)
	}
	encrypted, err := EncryptBytes(data)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, encrypted, 0600)
}

// EncryptBytes encrypts raw bytes using the public key.
func EncryptBytes(data []byte) ([]byte, error) {
	pubKey, err := LoadPublicKey()
	if err != nil {
		return nil, err
	}
	recipient, err := age.ParseX25519Recipient(pubKey)
	if err != nil {
		return nil, fmt.Errorf("parsing age recipient: %w", err)
	}

	var buf bytes.Buffer
	aw := armor.NewWriter(&buf)
	w, err := age.Encrypt(aw, recipient)
	if err != nil {
		return nil, fmt.Errorf("age encrypt: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if err := aw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecryptBytes decrypts age-encrypted bytes using the private key.
func DecryptBytes(encrypted []byte, identity age.Identity) ([]byte, error) {
	ar := armor.NewReader(bytes.NewReader(encrypted))
	r, err := age.Decrypt(ar, identity)
	if err != nil {
		return nil, fmt.Errorf("age decrypt: %w", err)
	}
	return io.ReadAll(r)
}
