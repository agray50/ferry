package bundle

import "testing"

func TestSubstituteContainerPath(t *testing.T) {
	path := "/root/.pyenv/versions/{VERSION}/"
	got := substituteVars(path, "3.12.4", "x86_64")
	want := "/root/.pyenv/versions/3.12.4/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteInstallPathFull(t *testing.T) {
	path := "~/.ferry/runtimes/python-{VERSION}/"
	got := substituteVars(path, "3.12.4", "x86_64")
	want := "~/.ferry/runtimes/python-3.12.4/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteRustArch(t *testing.T) {
	path := "/root/.rustup/toolchains/{VERSION}-{ARCH}-unknown-linux-gnu/"
	got := substituteVars(path, "stable", "x86_64")
	want := "/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
