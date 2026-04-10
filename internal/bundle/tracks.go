package bundle

// BuildTrack describes a target architecture and libc combination.
type BuildTrack struct {
	Arch       string
	Libc       string
	DockerBase string
	Platform   string
}

// BuildTracks is all supported build combinations.
var BuildTracks = []BuildTrack{
	{"x86_64", "glibc", "ubuntu:22.04", "linux/amd64"},
	{"arm64", "glibc", "ubuntu:22.04", "linux/arm64"},
	{"x86_64", "musl", "alpine:3.19", "linux/amd64"},
	{"arm64", "musl", "alpine:3.19", "linux/arm64"},
}

// FilterTracks returns tracks matching the given arch and libc filters.
// "all" matches any value.
func FilterTracks(arch, libc string) []BuildTrack {
	var out []BuildTrack
	for _, t := range BuildTracks {
		if (arch == "all" || arch == t.Arch) && (libc == "all" || libc == t.Libc) {
			out = append(out, t)
		}
	}
	return out
}
