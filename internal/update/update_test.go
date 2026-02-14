package update

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestCheckAndWarn(t *testing.T) {
	t.Parallel()

	remoteCalls := 0
	localReleasesServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			must.Write(t, w, `{"tag_name": "v0.2.0"}`)

			remoteCalls++
		}),
	)

	w := bytes.NewBuffer(nil)
	tempStateDir := t.TempDir()
	opts := Options{
		CurrentVersion:    "v0.1.0",
		CurrentTime:       time.Now().UTC(),
		StateDir:          tempStateDir,
		ReleaseServerHost: localReleasesServer.URL,
		ReleaseServerPath: "/repos/open-policy-agent/regal/releases/latest",
		CTAURLPrefix:      "https://github.com/open-policy-agent/regal/releases/tag/",
		Debug:             true,
	}

	output := checkAndWarn(t, opts, w)
	assert.Equal(t, 1, remoteCalls, "remote calls")

	expectedOutput := `A new version of Regal is available (v0.2.0). You are running v0.1.0.
See https://github.com/open-policy-agent/regal/releases/tag/v0.2.0 for the latest release.`

	assert.StringContains(t, output, expectedOutput, "output")

	// run the function again and check that the state is loaded from disk
	output = checkAndWarn(t, opts, w)
	assert.Equal(t, 1, remoteCalls, "remote calls")
	assert.StringContains(t, output, expectedOutput, "output") // output expected based on the data on disk

	// update the time to sometime in the future
	opts.CurrentTime = opts.CurrentTime.Add(4 * 24 * time.Hour)

	// run the function again and check that the state is loaded from the remote again
	output = checkAndWarn(t, opts, w)
	assert.Equal(t, 2, remoteCalls, "remote calls")
	assert.StringContains(t, output, expectedOutput, "output") // same output expected again

	// if the version is not a semver, then there should be no update warning
	opts.CurrentVersion = "not-semver"
	output = checkAndWarn(t, opts, w)

	if strings.Contains(output, "A new version of Regal is available") {
		t.Fatalf("expected no update warning for invalid semver, got\n%s", output)
	}

	// contains debug message when debug is enabled
	assert.StringContains(t, output, "Skipping version check: invalid semver", "debug=true, message expected")

	// debug disabled, no output at all
	opts.Debug = false
	must.Equal(t, "", checkAndWarn(t, opts, w), "debug=false, expected no output")

	// if the version is greater than the latest version, then there should be no output
	opts.CurrentVersion = "v0.3.0"
	opts.Debug = false
	must.Equal(t, "", checkAndWarn(t, opts, w), "expected no output")

	// if the version is the same as the latest version, then there should be no output
	opts.CurrentVersion = "v0.2.0"
	must.Equal(t, "", checkAndWarn(t, opts, w), "expected no output")
}

func checkAndWarn(t *testing.T, opts Options, bb *bytes.Buffer) string {
	t.Helper()
	bb.Reset()

	CheckAndWarn(opts, bb)

	return bb.String()
}
