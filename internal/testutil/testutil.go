package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

func Must[T any](x T, err error) func(testing.TB) T {
	return func(tb testing.TB) T {
		tb.Helper()

		if err != nil {
			tb.Fatal(err)
		}

		return x
	}
}

func TempDirectoryOf(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()

	for file, contents := range files {
		path := filepath.Join(tmpDir, file)

		MustMkdirAll(t, filepath.Dir(path))
		MustWriteFile(t, path, []byte(contents))
	}

	return tmpDir
}

func MustMkdirAll(t *testing.T, path ...string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(path...), 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func MustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}

	return contents
}

func MustWriteFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func MustRemove(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Fatalf("failed to remove file %s: %v", path, err)
	}
}

func MustRemoveAll(t *testing.T, path ...string) {
	t.Helper()

	if err := os.RemoveAll(filepath.Join(path...)); err != nil {
		t.Fatalf("failed to remove path %s: %v", path, err)
	}
}

func AssertNumViolations(tb testing.TB, num int, rep report.Report) {
	tb.Helper()

	if rep.Summary.NumViolations != num {
		tb.Errorf("expected %d violations, got %d", num, rep.Summary.NumViolations)
	}
}

func ViolationTitles(rep report.Report) *util.Set[string] {
	titles := make([]string, len(rep.Violations))
	for i := range rep.Violations {
		titles[i] = rep.Violations[i].Title
	}

	return util.NewSet(titles...)
}

func AssertOnlyViolations(t *testing.T, rep report.Report, expected ...string) {
	t.Helper()

	violationNames := ViolationTitles(rep)

	if violationNames.Size() != len(expected) {
		t.Errorf("expected %d violations, got %d: %v", len(expected), violationNames.Size(), violationNames.Items())
	}

	for _, name := range expected {
		if !violationNames.Contains(name) {
			t.Errorf("expected violation for rule %q, but it was not found", name)
		}
	}
}

func AssertContainsViolations(t *testing.T, rep report.Report, expected ...string) {
	t.Helper()

	violationNames := ViolationTitles(rep)

	for _, name := range expected {
		if !violationNames.Contains(name) {
			t.Errorf("expected violation for rule %q, but it was not found", name)
		}
	}
}

func AssertNotContainsViolations(t *testing.T, rep report.Report, unexpected ...string) {
	t.Helper()

	violationNames := ViolationTitles(rep)
	if violationNames.Contains(unexpected...) {
		t.Errorf("expected no violations for rules %v, but found: %v", unexpected, violationNames.Items())
	}
}

func RemoveIgnoreErr(paths ...string) func() {
	return func() {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}
}

func MustUnmarshalYAML[T any](t *testing.T, data []byte) T {
	t.Helper()

	var result T
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	return result
}

func ToJSONRawMessage(tb testing.TB, msg any) *json.RawMessage {
	tb.Helper()

	data, err := encoding.JSON().Marshal(msg)
	if err != nil {
		tb.Fatalf("failed to marshal message: %v", err)
	}

	jraw := json.RawMessage(data)

	return &jraw
}
