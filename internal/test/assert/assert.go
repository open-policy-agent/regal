// Package assert provides helper functions for test assertions. These function can be used in both tests and
// benchmarks, but should **not** be used inside benchmark Loop() bodies, as they come with some overhead that
// you don't want to measure.
package assert

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func Equal[T comparable](tb testing.TB, exp, got T, s ...any) {
	tb.Helper()

	if exp != got {
		tb.Error(FormatMsg(exp, got, s...))
	}
}

func SlicesEqual[T comparable](tb testing.TB, exp, got []T, s ...any) {
	tb.Helper()

	True(tb, slices.Equal(exp, got), s...)
}

func MapsEqual[K comparable, V comparable](tb testing.TB, exp, got map[K]V, s ...any) {
	tb.Helper()

	True(tb, maps.Equal(exp, got), s...)
}

func DeepEqual(tb testing.TB, exp, got any, s ...any) {
	tb.Helper()

	if !reflect.DeepEqual(exp, got) {
		tb.Error(FormatMsg(exp, got, s...))
	}
}

func DereferenceEqual[T comparable](tb testing.TB, exp T, got *T, s ...any) {
	tb.Helper()

	NotNil(tb, got, s...)
	Equal(tb, exp, *got, s...)
}

func StringContains[T ~string](tb testing.TB, str, substr T, s ...any) {
	tb.Helper()

	if !strings.Contains(string(str), string(substr)) {
		tb.Error(FormatMsg(str, substr, s...))
	}
}

func KeyMissing[K comparable, V any](tb testing.TB, m map[K]V, key K) {
	tb.Helper()

	if _, ok := m[key]; ok {
		tb.Errorf("did not expect key %v in map", key)
	}
}

func True(tb testing.TB, got bool, s ...any) {
	tb.Helper()

	Equal(tb, true, got, s...)
}

func False(tb testing.TB, got bool, s ...any) {
	tb.Helper()

	Equal(tb, false, got, s...)
}

func NotNil(tb testing.TB, got any, s ...any) {
	tb.Helper()

	if got == nil {
		tb.Error(FormatMsg("non-nil value", got, s...))
	}
}

func FormatMsg[T any](exp, got T, s ...any) string {
	if l := len(s); l == 0 {
		// fallthrough to default message
	} else if l == 1 {
		return fmt.Sprintf("%s: expected %v, got %v", s[0], exp, got)
	} else if msg, ok := s[0].(string); ok {
		return fmt.Sprintf("%s: expected %v, got %v", fmt.Sprintf(msg, s[1:]...), exp, got)
	}

	return fmt.Sprintf("expected %v, got %v", exp, got)
}
