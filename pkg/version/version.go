package version

import (
	"cmp"
	"runtime"
	"strings"
)

const platform = runtime.GOOS + "/" + runtime.GOARCH

// Values injected at build time using -ldflags.
var (
	Version   = ""
	Commit    = ""
	Timestamp = ""
	Hostname  = ""

	// The version of Go Regal was built with.
	goVersion = runtime.Version()
)

// Info wraps the various version metadata values and provides a means of marshalling as JSON or pretty string.
type Info struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Commit    string `json:"commit"`
	Timestamp string `json:"timestamp"`
	Hostname  string `json:"hostname"`
}

func (vi Info) String() string {
	return strings.Join(
		[]string{
			"Version:    " + vi.Version,
			"Go Version: " + vi.GoVersion,
			"Platform:   " + vi.Platform,
			"Commit:     " + vi.Commit,
			"Timestamp:  " + vi.Timestamp,
			"Hostname:   " + vi.Hostname,
		},
		"\n",
	) + "\n"
}

func New() Info {
	return Info{
		Version:   cmp.Or(Version, "unknown"),
		GoVersion: goVersion,
		Platform:  platform,
		Commit:    cmp.Or(Commit, "unknown"),
		Timestamp: cmp.Or(Timestamp, "unknown"),
		Hostname:  cmp.Or(Hostname, "unknown"),
	}
}
