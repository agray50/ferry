package store

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

func TestStorePutGet(t *testing.T) {
	withTempHome(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	data := []byte("test component data")
	hash := HashBytes(data)

	if err := s.Put(hash, data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// idempotent
	if err := s.Put(hash, data); err != nil {
		t.Fatalf("Put (2nd): %v", err)
	}

	got, err := s.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(data) {
		t.Error("Get returned wrong data")
	}
}

func TestStoreExists(t *testing.T) {
	withTempHome(t)
	s, _ := NewStore()

	if s.Exists("nonexistent") {
		t.Error("Exists should return false for unknown hash")
	}

	data := []byte("hello")
	hash := HashBytes(data)
	s.Put(hash, data)

	if !s.Exists(hash) {
		t.Error("Exists should return true after Put")
	}
}

func TestHashDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("beta"), 0644)

	h1, err := HashDir(dir, nil)
	if err != nil {
		t.Fatalf("HashDir: %v", err)
	}

	// same content = same hash
	h2, _ := HashDir(dir, nil)
	if h1 != h2 {
		t.Error("HashDir should be stable")
	}

	// change content = different hash
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0644)
	h3, _ := HashDir(dir, nil)
	if h1 == h3 {
		t.Error("HashDir should change when file content changes")
	}
}

func TestCompressExtractRoundtrip(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0644)
	sub := filepath.Join(src, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("nested"), 0644)

	compressed, err := CompressDir(src, nil)
	if err != nil {
		t.Fatalf("CompressDir: %v", err)
	}

	dest := t.TempDir()
	if err := ExtractTo(compressed, dest); err != nil {
		t.Fatalf("ExtractTo: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "file.txt"))
	if err != nil || string(data) != "content" {
		t.Errorf("file.txt: got %q want content", data)
	}
	data, err = os.ReadFile(filepath.Join(dest, "sub", "nested.txt"))
	if err != nil || string(data) != "nested" {
		t.Errorf("sub/nested.txt: got %q want nested", data)
	}
}

func TestDiffManifests(t *testing.T) {
	local := &Manifest{
		Components: []Component{
			{ID: "nvim", Hash: "aaa"},
			{ID: "rg", Hash: "bbb"},
			{ID: "new-tool", Hash: "ccc", SizeCompressed: 100},
		},
	}
	remote := &Manifest{
		Components: []Component{
			{ID: "nvim", Hash: "aaa"},
			{ID: "rg", Hash: "old"},
			{ID: "old-tool", Hash: "ddd"},
		},
	}

	diff := DiffManifests(local, remote)

	if len(diff.Same) != 1 || diff.Same[0].ID != "nvim" {
		t.Errorf("Same: expected [nvim], got %v", diff.Same)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].ID != "rg" {
		t.Errorf("Changed: expected [rg], got %v", diff.Changed)
	}
	if len(diff.New) != 1 || diff.New[0].ID != "new-tool" {
		t.Errorf("New: expected [new-tool], got %v", diff.New)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].ID != "old-tool" {
		t.Errorf("Removed: expected [old-tool], got %v", diff.Removed)
	}
	if diff.DiffSize() != 100 {
		t.Errorf("DiffSize: got %d want 100", diff.DiffSize())
	}
}
