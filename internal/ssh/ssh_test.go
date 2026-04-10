package ssh

import "testing"

func TestParseTarget(t *testing.T) {
	cases := []struct {
		input string
		user  string
		host  string
		port  int
	}{
		{"user@host", "user", "host", 22},
		{"user@host:2222", "user", "host", 2222},
		{"host", "", "host", 22}, // user comes from OS
	}

	for _, tc := range cases {
		pt, err := ParseTarget(tc.input)
		if err != nil {
			t.Errorf("ParseTarget(%q): %v", tc.input, err)
			continue
		}
		if tc.user != "" && pt.User != tc.user {
			t.Errorf("ParseTarget(%q).User = %q, want %q", tc.input, pt.User, tc.user)
		}
		if pt.Host != tc.host {
			t.Errorf("ParseTarget(%q).Host = %q, want %q", tc.input, pt.Host, tc.host)
		}
		if pt.Port != tc.port {
			t.Errorf("ParseTarget(%q).Port = %d, want %d", tc.input, pt.Port, tc.port)
		}
	}
}

func TestDetectLibc(t *testing.T) {
	if detectLibc("ldd (GNU libc) 2.35") != "glibc" {
		t.Error("expected glibc")
	}
	if detectLibc("musl libc (x86_64)") != "musl" {
		t.Error("expected musl")
	}
}

func TestNormaliseArch(t *testing.T) {
	if normaliseArch("x86_64") != "x86_64" {
		t.Error("x86_64")
	}
	if normaliseArch("aarch64") != "arm64" {
		t.Error("aarch64 → arm64")
	}
	if normaliseArch("amd64") != "x86_64" {
		t.Error("amd64 → x86_64")
	}
}

func TestParseZshVersion(t *testing.T) {
	v := parseZshVersion("zsh 5.9 (x86_64-ubuntu-linux-gnu)")
	if v != "5.9" {
		t.Errorf("got %q want 5.9", v)
	}
}
