package version

import (
	"cmp"
	"runtime"

	"github.com/open-policy-agent/opa/v1/util"
	opaversion "github.com/open-policy-agent/opa/v1/version"
)

// Info wraps the various version metadata values and provides a means of marshalling as JSON or pretty string.
type Info struct {
	Version    string `json:"version"`
	GoVersion  string `json:"go_version"`
	OPAVersion string `json:"opa_version"`
	Platform   string `json:"platform"`
	Commit     string `json:"commit"`
	Timestamp  string `json:"timestamp"`
	Hostname   string `json:"hostname"`
}

const platform = runtime.GOOS + "/" + runtime.GOARCH

var (
	// Values injected at build time using -ldflags.
	Version, Commit, Timestamp, Hostname = "", "", "", ""
	// The version of Go Regal was built with.
	goVersion = runtime.Version()
)

func New() Info {
	return Info{
		Version:    cmp.Or(Version, "unknown"),
		GoVersion:  goVersion,
		OPAVersion: opaversion.Version,
		Platform:   platform,
		Commit:     cmp.Or(Commit, "unknown"),
		Timestamp:  cmp.Or(Timestamp, "unknown"),
		Hostname:   cmp.Or(Hostname, "unknown"),
	}
}

func (vi Info) Append(buf []byte) []byte {
	buf = append(append(append(buf, "Version:       "...), vi.Version...), '\n')
	buf = append(append(append(buf, "Go Version:    "...), vi.GoVersion...), '\n')
	buf = append(append(append(buf, "OPA Version:   "...), vi.OPAVersion...), '\n')
	buf = append(append(append(buf, "Platform:      "...), vi.Platform...), '\n')
	buf = append(append(append(buf, "Commit:        "...), vi.Commit...), '\n')
	buf = append(append(append(buf, "Timestamp:     "...), vi.Timestamp...), '\n')
	buf = append(append(append(buf, "Installed via: "...), vi.Hostname...), '\n')

	return buf
}

func (vi Info) String() string {
	return util.ByteSliceToString(vi.Append(nil))
}
