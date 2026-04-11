package bundle

import "testing"

func TestBuildTracksNoMusl(t *testing.T) {
	for _, tr := range BuildTracks {
		if tr.OS == "linux" && tr.Libc == "musl" {
			t.Errorf("musl track found: %+v — musl is not supported", tr)
		}
	}
}

func TestBuildTracksHasDarwin(t *testing.T) {
	var darwinCount int
	for _, tr := range BuildTracks {
		if tr.OS == "darwin" {
			darwinCount++
		}
	}
	if darwinCount != 2 {
		t.Errorf("expected 2 darwin tracks (x86_64 + arm64), got %d", darwinCount)
	}
}

func TestBuildTracksHasLinux(t *testing.T) {
	var linuxCount int
	for _, tr := range BuildTracks {
		if tr.OS == "linux" {
			linuxCount++
		}
	}
	if linuxCount != 2 {
		t.Errorf("expected 2 linux tracks, got %d", linuxCount)
	}
}

func TestBuildMethodSet(t *testing.T) {
	for _, tr := range BuildTracks {
		if tr.BuildMethod == "" {
			t.Errorf("track %s/%s has empty BuildMethod", tr.OS, tr.Arch)
		}
		if tr.OS == "linux" && tr.BuildMethod != "docker" {
			t.Errorf("linux track should use docker, got %q", tr.BuildMethod)
		}
		if tr.OS == "darwin" && tr.BuildMethod != "download" {
			t.Errorf("darwin track should use download, got %q", tr.BuildMethod)
		}
	}
}

func TestFilterTracksByOS(t *testing.T) {
	linux := FilterTracks("all", "linux")
	if len(linux) != 2 {
		t.Errorf("FilterTracks(all, linux) = %d, want 2", len(linux))
	}
	darwin := FilterTracks("all", "darwin")
	if len(darwin) != 2 {
		t.Errorf("FilterTracks(all, darwin) = %d, want 2", len(darwin))
	}
}
