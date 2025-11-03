package semver

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// reMetaIdentifier matches pre-release and metadata identifiers against the spec requirements.
var reMetaIdentifier = regexp.MustCompile(`^[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*$`)

// Version represents a parsed SemVer.
type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string `json:"PreRelease,omitempty"`
	Metadata   string `json:"Metadata,omitempty"`
}

// Parse constructs new semver Version from version string.
func Parse(version string) (v Version, err error) {
	version = strings.TrimPrefix(version, "v")

	version, v.Metadata = cut(version, '+')
	if v.Metadata != "" && !reMetaIdentifier.MatchString(v.Metadata) {
		return v, fmt.Errorf("invalid metadata identifier: %s", v.Metadata)
	}

	version, v.PreRelease = cut(version, '-')
	if v.PreRelease != "" && !reMetaIdentifier.MatchString(v.PreRelease) {
		return v, fmt.Errorf("invalid pre-release identifier: %s", v.PreRelease)
	}

	if strings.Count(version, ".") != 2 {
		return v, fmt.Errorf("%s should contain major, minor, and patch versions", version)
	}

	major, after := cut(version, '.')
	if v.Major, err = strconv.ParseInt(major, 10, 64); err != nil {
		return v, err
	}

	minor, after := cut(after, '.')
	if v.Minor, err = strconv.ParseInt(minor, 10, 64); err != nil {
		return v, err
	}

	if v.Patch, err = strconv.ParseInt(after, 10, 64); err != nil {
		return v, err
	}

	return v, nil
}

// MustParse is like Parse but panics if the version string is invalid instead of returning an error.
func MustParse(version string) Version {
	v, err := Parse(version)
	if err != nil {
		panic(err)
	}

	return v
}

// AppendText appends the textual representation of the version to b and returns the extended buffer.
// This method conforms to the encoding.TextAppender interface, and is useful for serializing the Version
// without allocating, provided the caller has pre-allocated sufficient space in b.
func (v Version) AppendText(b []byte) ([]byte, error) {
	if b == nil {
		b = make([]byte, 0, length(v))
	}

	b = append(strconv.AppendInt(b, v.Major, 10), '.')
	b = append(strconv.AppendInt(b, v.Minor, 10), '.')
	b = strconv.AppendInt(b, v.Patch, 10)

	if v.PreRelease != "" {
		b = append(append(b, '-'), v.PreRelease...)
	}

	if v.Metadata != "" {
		b = append(append(b, '+'), v.Metadata...)
	}

	return b, nil
}

// String returns the string representation of the version.
func (v Version) String() string {
	bs := make([]byte, 0, length(v))
	bs, _ = v.AppendText(bs)

	return string(bs)
}

// Compare tests if v is less than, equal to, or greater than other, returning -1, 0, or +1 respectively.
// Comparison is based on the SemVer specification (https://semver.org/#spec-item-11).
func (v Version) Compare(other Version) int {
	if v.Major > other.Major {
		return 1
	} else if v.Major < other.Major {
		return -1
	}

	if v.Minor > other.Minor {
		return 1
	} else if v.Minor < other.Minor {
		return -1
	}

	if v.Patch > other.Patch {
		return 1
	} else if v.Patch < other.Patch {
		return -1
	}

	if v.PreRelease == other.PreRelease {
		return 0
	}

	// if two versions are otherwise equal it is the one without a pre-release that is greater
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}

	if other.PreRelease == "" && v.PreRelease != "" {
		return -1
	}

	a, afterA := cut(v.PreRelease, '.')
	b, afterB := cut(other.PreRelease, '.')

	for {
		if a == "" && b != "" {
			return -1
		}

		if a != "" && b == "" {
			return 1
		}

		aIsInt := isAllDecimals(a)
		bIsInt := isAllDecimals(b)

		// numeric identifiers have lower precedence than non-numeric
		if aIsInt && !bIsInt {
			return -1
		} else if !aIsInt && bIsInt {
			return 1
		}

		if aIsInt && bIsInt {
			aInt, _ := strconv.Atoi(a)
			bInt, _ := strconv.Atoi(b)

			if aInt > bInt {
				return 1
			} else if aInt < bInt {
				return -1
			}
		} else {
			// string comparison
			if a > b {
				return 1
			} else if a < b {
				return -1
			}
		}

		// a larger set of pre-release fields has a higher precedence than a
		// smaller set, if all of the preceding identifiers are equal.
		if afterA != "" && afterB == "" {
			return 1
		} else if afterA == "" && afterB != "" {
			return -1
		}

		a, afterA = cut(afterA, '.')
		b, afterB = cut(afterB, '.')
	}
}

func isAllDecimals(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return s != ""
}

// length allows calculating the length of the version for pre-allocation.
func length(v Version) int {
	n := numDigitsInt64(v.Major) + numDigitsInt64(v.Minor) + numDigitsInt64(v.Patch) + 2
	if v.PreRelease != "" {
		n += len(v.PreRelease) + 1
	}

	if v.Metadata != "" {
		n += len(v.Metadata) + 1
	}

	return n
}

// cut is a *slightly* faster version of strings.Cut only accepting
// single byte separators, and skipping the boolean return value.
func cut(s string, sep byte) (before, after string) {
	if i := strings.IndexByte(s, sep); i >= 0 {
		return s[:i], s[i+1:]
	}

	return s, ""
}

// TODO: remove in favor of util.NumDigitsInt64 from OPA, when available.
func numDigitsInt64(n int64) int {
	if n == 0 {
		return 1
	}

	if n < 0 {
		n = -n
	}

	return int(math.Log10(float64(n))) + 1
}
