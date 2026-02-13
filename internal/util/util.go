package util

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"iter"
	"math"
	"net"
	"path/filepath"
	"slices"
	"strings"
)

// NilSliceToEmpty returns empty slice if provided slice is nil.
func NilSliceToEmpty[T any](a []T) []T {
	if a == nil {
		return []T{}
	}

	return a
}

// SearchMap searches map for value at provided path.
func SearchMap(object map[string]any, path ...string) (any, error) {
	current := object

	for i, p := range path {
		var ok bool
		if i == len(path)-1 {
			if value, ok := current[p]; ok {
				return value, nil
			}

			return nil, fmt.Errorf("no '%v' attribute at path '%v'", p, strings.Join(path[:i], "."))
		}

		if current, ok = current[p].(map[string]any); !ok {
			return nil, fmt.Errorf("no '%v' attribute at path '%v'", p, strings.Join(path[:i], "."))
		}
	}

	return current, nil
}

// Must takes a value and an error (as commonly returned by Go functions) and panics if the error is not nil.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// Map applies a function to each element of a slice and returns a new slice with the results.
func Map[T, U any](a []T, f func(T) U) []U {
	b := make([]U, len(a))
	for i := range a {
		b[i] = f(a[i])
	}

	return b
}

// MapKeys applies a function to each key of a map and returns a new slice with the results.
func MapKeys[K comparable, V any, U any](m map[K]V, f func(K) U) []U {
	keys := make([]U, 0, len(m))
	for k := range m {
		keys = append(keys, f(k))
	}

	return keys
}

// MapValues applies the function f to each value in the map m and returns a new map with the same keys.
func MapValues[K comparable, V, R any](m map[K]V, f func(V) R) map[K]R {
	mapped := make(map[K]R, len(m))
	for k, v := range m {
		mapped[k] = f(v)
	}

	return mapped
}

// Filter returns a new slice containing only the elements of s that
// satisfy the predicate f. This function runs each element of s through
// f twice in order to allocate exactly what is needed. This is commonly
// *much* more efficient than appending blindly, but do not use this if
// the predicate function is expensive to compute.
func Filter[T any](s []T, f func(T) bool) []T {
	n := 0

	for i := range s {
		if f(s[i]) {
			n++
		}
	}

	if n == 0 {
		return nil
	}

	r := make([]T, 0, n)

	for i := range s {
		if f(s[i]) {
			r = append(r, s[i])
		}
	}

	return r
}

// FindClosestMatchingRoot finds the closest matching root for a given path.
// If no matching root is found, an empty string is returned.
func FindClosestMatchingRoot(path string, roots []string) string {
	currentLongestPrefix, longestPrefixIndex := 0, -1

	for i, root := range roots {
		if root == path {
			return path
		}

		if !strings.HasPrefix(path, root) {
			continue
		}

		if len(root) > currentLongestPrefix {
			currentLongestPrefix = len(root)
			longestPrefixIndex = i
		}
	}

	if longestPrefixIndex == -1 {
		return ""
	}

	return roots[longestPrefixIndex]
}

// FilepathJoiner returns a function that joins provided path with base path.
func FilepathJoiner(base string) func(string) string {
	return func(path string) string {
		return filepath.Join(base, path)
	}
}

// SafeUintToInt will convert a uint to an int, clamping the result to
// math.MaxInt.
func SafeUintToInt(u uint) int {
	if u > math.MaxInt {
		return math.MaxInt // Clamp to prevent overflow
	}

	return int(u)
}

// EnsureSuffix ensures that the given string ends with the specified suffix.
// If the string already ends with suf, it is returned unchanged.
// Note that an empty string s is returned unchanged — *not* turned into "/".
func EnsureSuffix(s, suf string) string {
	if s != "" && !strings.HasSuffix(s, suf) {
		s += suf
	}

	return s
}

// IsAnyError checks if the provided error "Is" any of the provided errors.
func IsAnyError(err error, errs ...error) bool {
	if err != nil {
		return slices.ContainsFunc(errs, Partial2(errors.Is, err))
	}

	return false
}

// HasAnySuffix checks if the string s has any of the provided suffixes.
func HasAnySuffix(s string, suffixes ...string) bool {
	return slices.ContainsFunc(suffixes, Partial2(strings.HasSuffix, s))
}

// Partial2 is a helper function that partially applies the first argument
// of a two-argument function, returning a new function taking the second argument.
func Partial2[T, U, R any](f func(a T, b U) R, a T) func(U) R {
	return func(b U) R {
		return f(a, b)
	}
}

// EqualsAny checks if the provided value is equal to any of the values in the slice.
func EqualsAny[T comparable](a ...T) func(T) bool {
	return Partial2(slices.Contains, a)
}

// SafeIntToUint will convert an int to a uint, clamping negative values to 0.
func SafeIntToUint(i int) uint {
	if i < 0 {
		return 0 // Clamp negative values to 0
	}

	return uint(i)
}

// FreePort returns a free port to listen on, if none of the preferred ports
// are available on the localhost interface, then a random free port is returned.
func FreePort(preferred ...int) (port int, err error) {
	listen := func(p int) (int, error) {
		l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: p})
		if err != nil {
			return 0, fmt.Errorf("failed to listen on port %d: %w", p, err)
		}
		defer l.Close()

		if addr, ok := l.Addr().(*net.TCPAddr); ok {
			return addr.Port, nil
		}

		return 0, errors.New("failed to get port from listener")
	}

	for _, p := range preferred {
		if p != 0 {
			if port, err = listen(p); err == nil {
				return port, nil
			}
		}
	}

	// If no preferred port is available, find a random free port using :0
	if port, err = listen(0); err == nil {
		return port, nil
	}

	return 0, fmt.Errorf("failed to find free port: %w", err)
}

// Wrap wraps a value and an error into a function that returns the value and error.
func Wrap[T any](v T, err error) func(string) (T, error) {
	if err != nil {
		return func(msg string) (T, error) {
			return v, fmt.Errorf("%s: %w", msg, err)
		}
	}

	return func(string) (T, error) {
		return v, nil
	}
}

// WrapErr wraps an error with a message if the error is not nil.
func WrapErr(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// SendToAll sends the provided value to all provided channels.
func SendToAll[T any](val T, ch ...chan T) {
	for _, c := range ch {
		c <- val
	}
}

// GetMapValue extracts a typed value from a map[string]any, returning the value if the type matched,
// or the zero values of correct type.
func GetMapValue[T any](m map[string]any, key string) T {
	if val, ok := m[key]; ok {
		if typed, ok := val.(T); ok {
			return typed
		}
	}

	var zero T

	return zero
}

// AnySliceTo converts a slice of any to a slice of T, returning an error if any element cannot be casted.
func AnySliceTo[T any](in []any) ([]T, error) {
	out := make([]T, 0, len(in))

	for _, item := range in {
		casted, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("expected %T, got %T", casted, item)
		}

		out = append(out, casted)
	}

	return out, nil
}

// Sorted sorts s in place using slices.Sort and returns it
// Can be convenient for use in return values, map definitions, chaining, etc.
func Sorted[T cmp.Ordered](s []T) []T {
	slices.Sort(s)

	return s
}

// Reversed reverses s in place using slices.Reverse and returns it.
func Reversed[T any](s []T) []T {
	slices.Reverse(s)

	return s
}

// LineContents returns the contents on line lineNum (0-indexed) from document.
// This function assumes the lineNum is known to be contained within the document,.
func LineContents(document []byte, lineNum uint) []byte {
	for i, line := range Lines(document) {
		if i == lineNum {
			return bytes.TrimSuffix(line, []byte{'\n'})
		}
	}

	return nil
}

// Lines works like [bytes.Lines] but yields both the line number (0-indexed) and the line contents.
func Lines(s []byte) iter.Seq2[uint, []byte] {
	return func(yield func(uint, []byte) bool) {
		var lineNum uint
		for line := range bytes.Lines(s) {
			if !yield(lineNum, line) {
				return
			}

			lineNum++
		}
	}
}
