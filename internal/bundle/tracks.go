package bundle

// BuildTrack describes one build target (OS + arch combination).
type BuildTrack struct {
	Arch        string // x86_64 | arm64
	OS          string // linux | darwin
	Libc        string // glibc | "" (darwin has no libc distinction)
	DockerBase  string // "ubuntu:22.04" for linux docker tracks; "" for darwin
	Platform    string // "linux/amd64" | "linux/arm64" | "" for darwin
	BuildMethod string // "docker" | "download"
}

// BuildTracks lists all supported build combinations.
// Linux tracks use Docker. Darwin tracks use direct binary downloads.
var BuildTracks = []BuildTrack{
	{Arch: "x86_64", OS: "linux",  Libc: "glibc", DockerBase: "ubuntu:22.04", Platform: "linux/amd64",  BuildMethod: "docker"},
	{Arch: "arm64",  OS: "linux",  Libc: "glibc", DockerBase: "ubuntu:22.04", Platform: "linux/arm64",  BuildMethod: "docker"},
	{Arch: "x86_64", OS: "darwin", Libc: "",       DockerBase: "",             Platform: "",             BuildMethod: "download"},
	{Arch: "arm64",  OS: "darwin", Libc: "",       DockerBase: "",             Platform: "",             BuildMethod: "download"},
}

// FilterTracks returns tracks matching the given arch and OS filters.
// "all" matches any value for that field.
func FilterTracks(arch, os string) []BuildTrack {
	var out []BuildTrack
	for _, t := range BuildTracks {
		if (arch == "all" || arch == t.Arch) && (os == "all" || os == t.OS) {
			out = append(out, t)
		}
	}
	return out
}
