package util

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// NullToEmpty returns empty slice if provided slice is nil.
func NullToEmpty[T any](a []T) []T {
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
	currentLongestSuffix, longestSuffixIndex := 0, -1

	for i, root := range roots {
		if root == path {
			return path
		}

		if !strings.HasPrefix(path, root) {
			continue
		}

		suffix := strings.TrimPrefix(root, path)
		if len(suffix) > currentLongestSuffix {
			currentLongestSuffix = len(suffix)
			longestSuffixIndex = i
		}
	}

	if longestSuffixIndex == -1 {
		return ""
	}

	return roots[longestSuffixIndex]
}

// FilepathJoiner returns a function that joins provided path with base path.
func FilepathJoiner(base string) func(string) string {
	return func(path string) string {
		return filepath.Join(base, path)
	}
}

// DeleteEmptyDirs will delete empty directories up to the root for a given
// directory.
func DeleteEmptyDirs(dir string) error {
	for {
		// os.Remove will only delete empty directories
		if err := os.Remove(dir); err != nil {
			if os.IsExist(err) {
				break
			} else if !os.IsPermission(err) {
				return fmt.Errorf("failed to clean directory %s: %w", dir, err)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return nil
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
		for _, e := range errs {
			if errors.Is(err, e) {
				return true
			}
		}
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

		addr, ok := l.Addr().(*net.TCPAddr)
		if !ok {
			return 0, errors.New("failed to get port from listener")
		}

		return addr.Port, nil
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

func Pointer[T any](v T) *T {
	return &v
}

func Wrap[T any](v T, err error) func(string) (T, error) {
	if err != nil {
		return func(msg string) (T, error) {
			return v, fmt.Errorf("%s: %w", msg, err)
		}
	}

	return func(_ string) (T, error) {
		return v, nil
	}
}

func WrapErr(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func WithOpen[T any](path string, f func(*os.File) (T, error)) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T

		return zero, fmt.Errorf("failed to open file '%s': %w", path, err)
	}
	defer file.Close()

	return f(file)
}
