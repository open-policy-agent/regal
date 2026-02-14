package must

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/regal/internal/test/assert"
)

func Return[T any](x T, err error) func(testing.TB) T {
	return func(tb testing.TB) T {
		tb.Helper()

		if err != nil {
			tb.Fatal(err)
		}

		return x
	}
}

func Equal[T comparable](tb testing.TB, exp, got T, s ...any) {
	tb.Helper()

	if exp != got {
		tb.Fatal(assert.FormatMsg(exp, got, s...))
	}
}

func NotEqual[T comparable](tb testing.TB, exp, got T, s ...any) {
	tb.Helper()

	if exp == got {
		tb.Fatal(assert.FormatMsg(exp, got, s...))
	}
}

func Be[T any](tb testing.TB, v any) T {
	tb.Helper()

	r, ok := v.(T)
	if !ok {
		tb.Fatalf("failed to convert %T to %T", v, r)
	}

	return r
}

func Unmarshal[T any](tb testing.TB, data []byte) (v T) {
	tb.Helper()

	if err := jsoniter.ConfigFastest.Unmarshal(data, &v); err != nil {
		tb.Fatalf("failed to unmarshal: %v", err)
	}

	return v
}

func Write(tb testing.TB, w io.Writer, contents string) {
	tb.Helper()

	if _, err := w.Write([]byte(contents)); err != nil {
		tb.Fatalf("failed to write to writer: %v", err)
	}
}

func ReadFile(tb testing.TB, path string) string {
	tb.Helper()

	contents, err := os.ReadFile(path)
	Equal(tb, nil, err, "failed to read file %s", path)

	return string(contents)
}

func WriteFile(tb testing.TB, path string, contents []byte) {
	tb.Helper()

	if err := os.WriteFile(path, contents, 0o600); err != nil {
		tb.Fatalf("failed to write file %s: %v", path, err)
	}
}

func Remove(tb testing.TB, path string) {
	tb.Helper()

	if err := os.Remove(path); err != nil {
		tb.Fatalf("failed to remove file %s: %v", path, err)
	}
}

func RemoveAll(tb testing.TB, path ...string) {
	tb.Helper()

	if err := os.RemoveAll(filepath.Join(path...)); err != nil {
		tb.Fatalf("failed to remove path %s: %v", path, err)
	}
}

func MkdirAll(tb testing.TB, path ...string) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Join(path...), 0o755); err != nil {
		tb.Fatalf("failed to create directory %s: %v", path, err)
	}
}
