package update

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestCheckAndWarn(t *testing.T) {
	t.Parallel()

	remoteCalls := 0
	localReleasesServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			testutil.MustWrite(t, w, `{"tag_name": "v0.2.0"}`)

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
	if remoteCalls != 1 {
		t.Errorf("expected 1 remote call, got %d", remoteCalls)
	}

	expectedOutput := `A new version of Regal is available (v0.2.0). You are running v0.1.0.
See https://github.com/open-policy-agent/regal/releases/tag/v0.2.0 for the latest release.`

	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("expected output to contain\n%s,\ngot\n%s", expectedOutput, output)
	}

	// run the function again and check that the state is loaded from disk
	output = checkAndWarn(t, opts, w)
	if remoteCalls != 1 {
		t.Errorf("expected remote to only be called once, got %d", remoteCalls)
	}

	// the same output is expected based on the data on disk
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("expected output to contain\n%s,\ngot\n%s", expectedOutput, output)
	}

	// update the time to sometime in the future
	opts.CurrentTime = opts.CurrentTime.Add(4 * 24 * time.Hour)

	// run the function again and check that the state is loaded from the remote again
	output = checkAndWarn(t, opts, w)
	if remoteCalls != 2 {
		t.Errorf("expected remote to be called again, got %d", remoteCalls)
	}

	// the same output is expected again
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("expected output to contain\n%s,\ngot\n%s", expectedOutput, output)
	}

	// if the version is not a semver, then there should be no update warning
	opts.CurrentVersion = "not-semver"
	output = checkAndWarn(t, opts, w)

	if strings.Contains(output, "A new version of Regal is available") {
		t.Fatalf("expected no update warning for invalid semver, got\n%s", output)
	}

	// contains debug message when debug is enabled
	if !strings.Contains(output, "Skipping version check: invalid semver") {
		t.Fatalf("expected debug message for invalid semver when debug=true, got\n%s", output)
	}

	// debug disabled, no output at all
	opts.Debug = false
	if output = checkAndWarn(t, opts, w); output != "" {
		t.Fatalf("expected no output when debug=false and invalid semver, got\n%s", output)
	}

	opts.Debug = true

	// if the version is greater than the latest version, then there should be no output
	opts.CurrentVersion = "v0.3.0"

	opts.Debug = false
	if output = checkAndWarn(t, opts, w); output != "" {
		t.Fatalf("expected no output, got\n%s", output)
	}

	// if the version is the same as the latest version, then there should be no output
	opts.CurrentVersion = "v0.2.0"
	if output = checkAndWarn(t, opts, w); output != "" {
		t.Fatalf("expected no output, got\n%s", output)
	}
}

func checkAndWarn(t *testing.T, opts Options, bb *bytes.Buffer) string {
	t.Helper()
	bb.Reset()

	CheckAndWarn(opts, bb)

	return bb.String()
}
