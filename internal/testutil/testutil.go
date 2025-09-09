package testutil

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func MustBeOK[T any](x T, ok bool) func(testing.TB) T {
	return func(tb testing.TB) T {
		tb.Helper()

		if !ok {
			tb.Fatal("expected ok to be true, got false")
		}

		return x
	}
}

func MustBe[T any](tb testing.TB, v any) T {
	tb.Helper()

	r, ok := v.(T)
	if !ok {
		tb.Fatalf("failed to convert %T to %T", v, r)
	}

	return r
}

func NoErr(err error) func(testing.TB) {
	return func(tb testing.TB) {
		tb.Helper()

		if err != nil {
			tb.Fatal(err)
		}
	}
}

func ErrMustContain(err error, substr string) func(testing.TB) {
	return func(tb testing.TB) {
		tb.Helper()

		if err == nil {
			tb.Fatal("expected error got nil")
		} else if !strings.Contains(err.Error(), substr) {
			tb.Fatalf("expected error to contain %q, got %q", substr, err.Error())
		}
	}
}

func TempDirectoryOf(tb testing.TB, files map[string]string) string {
	tb.Helper()

	tmpDir := tb.TempDir()

	for file, contents := range files {
		path := filepath.Join(tmpDir, file)

		MustMkdirAll(tb, filepath.Dir(path))
		MustWriteFile(tb, path, []byte(contents))
	}

	return tmpDir
}

func MustMkdirAll(tb testing.TB, path ...string) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Join(path...), 0o755); err != nil {
		tb.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func MustReadFile(tb testing.TB, path string) string {
	tb.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("failed to read file %s: %v", path, err)
	}

	return string(contents)
}

func MustWrite(tb testing.TB, w io.Writer, contents string) {
	tb.Helper()

	if _, err := w.Write([]byte(contents)); err != nil {
		tb.Fatalf("failed to write to writer: %v", err)
	}
}

func MustWriteFile(tb testing.TB, path string, contents []byte) {
	tb.Helper()

	if err := os.WriteFile(path, contents, 0o600); err != nil {
		tb.Fatalf("failed to write file %s: %v", path, err)
	}
}

func MustRemove(tb testing.TB, path string) {
	tb.Helper()

	if err := os.Remove(path); err != nil {
		tb.Fatalf("failed to remove file %s: %v", path, err)
	}
}

func MustRemoveAll(tb testing.TB, path ...string) {
	tb.Helper()

	if err := os.RemoveAll(filepath.Join(path...)); err != nil {
		tb.Fatalf("failed to remove path %s: %v", path, err)
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

func AssertOnlyViolations(tb testing.TB, rep report.Report, expected ...string) {
	tb.Helper()

	violationNames := ViolationTitles(rep)
	if violationNames.Size() != len(expected) {
		tb.Errorf("expected %d violations, got %d: %v", len(expected), violationNames.Size(), violationNames.Items())
	}

	for _, name := range expected {
		if !violationNames.Contains(name) {
			tb.Errorf("expected violation for rule %q, but it was not found", name)
		}
	}
}

func AssertContainsViolations(tb testing.TB, rep report.Report, expected ...string) {
	tb.Helper()

	violationNames := ViolationTitles(rep)
	for _, name := range expected {
		if !violationNames.Contains(name) {
			tb.Errorf("expected violation for rule %q, but it was not found", name)
		}
	}
}

func AssertNotContainsViolations(tb testing.TB, rep report.Report, unexpected ...string) {
	tb.Helper()

	violationNames := ViolationTitles(rep)
	if violationNames.Contains(unexpected...) {
		tb.Errorf("expected no violations for rules %v, but found: %v", unexpected, violationNames.Items())
	}
}

func RemoveIgnoreErr(paths ...string) func() {
	return func() {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}
}

func MustUnmarshalYAML[T any](tb testing.TB, data []byte) T {
	tb.Helper()

	var result T
	if err := yaml.Unmarshal(data, &result); err != nil {
		tb.Fatalf("failed to unmarshal YAML: %v", err)
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
