package semver_test

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/semver"
)

type (
	fixture struct {
		greater string
		lesser  string
	}

	compareFixture struct {
		greater semver.Version
		lesser  semver.Version
	}
)

// test fixtures and comparison tests originally from: https://github.com/coreos/go-semver
var (
	fixtures = []fixture{
		{"0.0.0", "0.0.0-foo"},
		{"0.0.1", "0.0.0"},
		{"1.0.0", "0.9.9"},
		{"0.10.0", "0.9.0"},
		{"0.99.0", "0.10.0"},
		{"2.0.0", "1.2.3"},
		{"0.0.0", "0.0.0-foo"},
		{"0.0.1", "0.0.0"},
		{"1.0.0", "0.9.9"},
		{"0.10.0", "0.9.0"},
		{"0.99.0", "0.10.0"},
		{"2.0.0", "1.2.3"},
		{"0.0.0", "0.0.0-foo"},
		{"0.0.1", "0.0.0"},
		{"1.0.0", "0.9.9"},
		{"0.10.0", "0.9.0"},
		{"0.99.0", "0.10.0"},
		{"2.0.0", "1.2.3"},
		{"1.2.3", "1.2.3-asdf"},
		{"1.2.3", "1.2.3-4"},
		{"1.2.3", "1.2.3-4-foo"},
		{"1.2.3-5-foo", "1.2.3-5"},
		{"1.2.3-5", "1.2.3-4"},
		{"1.2.3-5-foo", "1.2.3-5-Foo"},
		{"3.0.0", "2.7.2+asdf"},
		{"3.0.0+foobar", "2.7.2"},
		{"1.2.3-a.10", "1.2.3-a.5"},
		{"1.2.3-a.b", "1.2.3-a.5"},
		{"1.2.3-a.b", "1.2.3-a"},
		{"1.2.3-a.b.c.10.d.5", "1.2.3-a.b.c.5.d.100"},
		{"1.0.0", "1.0.0-rc.1"},
		{"1.0.0-rc.2", "1.0.0-rc.1"},
		{"1.0.0-rc.1", "1.0.0-beta.11"},
		{"1.0.0-beta.11", "1.0.0-beta.2"},
		{"1.0.0-beta.2", "1.0.0-beta"},
		{"1.0.0-beta", "1.0.0-alpha.beta"},
		{"1.0.0-alpha.beta", "1.0.0-alpha.1"},
		{"1.0.0-alpha.1", "1.0.0-alpha"},
		{"1.2.3-rc.1-1-1hash", "1.2.3-rc.2"},
	}

	equalFixtures = []string{
		"0.0.0",
		"1.2.3",
		"1.2.3-rc.1",
		"1.2.3+build.123",
		"1.2.3-rc.1+build.123",
		"1.2.3-rc.1+build.123.444",
	}
)

func TestCompareEqual(t *testing.T) {
	t.Parallel()

	for _, v := range equalFixtures {
		a := semver.MustParse(v)
		o := a

		if a.Compare(o) != 0 {
			t.Errorf("Expected %s to be equal to %s", a, o)
		}
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	for _, v := range fixtures {
		gt := semver.MustParse(v.greater)
		lt := semver.MustParse(v.lesser)

		if gt.Compare(lt) <= 0 {
			t.Errorf("%s should be greater than %s", gt, lt)
		}

		if lt.Compare(gt) > 0 {
			t.Errorf("%s should not be greater than %s", lt, gt)
		}
	}
}

func TestBadInput(t *testing.T) {
	t.Parallel()

	bad := []string{
		"1.2",
		"1.2.3x",
		"0x1.3.4",
		"-1.2.3",
		"1.2.3.4",
		"0.88.0-11_e4e5dcabb",
		"0.88.0+11_e4e5dcabb",
	}
	for _, b := range bad {
		if _, err := semver.Parse(b); err == nil {
			t.Error("Improperly accepted value: ", b)
		}
	}
}

func BenchmarkCompare(b *testing.B) {
	versionFixtures := make([]compareFixture, 0, len(fixtures))
	for _, v := range fixtures {
		versionFixtures = append(versionFixtures, compareFixture{
			greater: semver.MustParse(v.greater),
			lesser:  semver.MustParse(v.lesser),
		})
	}

	for b.Loop() {
		for _, v := range versionFixtures {
			if v.greater.Compare(v.lesser) <= 0 {
				b.Fatalf("%s should be greater than %s", v.greater, v.lesser)
			}

			if v.lesser.Compare(v.greater) > 0 {
				b.Fatalf("%s should not be greater than %s", v.lesser, v.greater)
			}
		}
	}
}

func BenchmarkCompareEqual(b *testing.B) {
	versionFixtures := make([]semver.Version, 0, len(equalFixtures))
	for _, v := range equalFixtures {
		versionFixtures = append(versionFixtures, semver.MustParse(v))
	}

	for b.Loop() {
		for _, v := range versionFixtures {
			o := v
			if v.Compare(o) != 0 {
				b.Fatalf("Expected %s to be equal to %s", v, o)
			}
		}
	}
}

func BenchmarkString(b *testing.B) {
	v := semver.MustParse("1.2.3-alpha.1+build.123")

	for b.Loop() {
		if s := v.String(); s != "1.2.3-alpha.1+build.123" {
			b.Fatalf("unexpected version string: %s", s)
		}
	}
}

func BenchmarkAppendText(b *testing.B) {
	v := semver.MustParse("1.2.3-alpha.1+build.123")

	for b.Loop() {
		_, err := v.AppendText(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppendTextPreAllocated(b *testing.B) {
	v, err := semver.Parse("1.2.3-alpha.1+build.123")
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]byte, 0, 32)

	for b.Loop() {
		if buf, err = v.AppendText(buf); err != nil {
			b.Fatal(err)
		}

		if string(buf) != "1.2.3-alpha.1+build.123" {
			b.Fatal("unexpected version string")
		}

		buf = buf[:0]
	}
}

func BenchmarkParseSimple(b *testing.B) {
	for b.Loop() {
		semver.MustParse("1.2.3")
	}
}

func BenchmarkParsePrereleaseAndMetadata(b *testing.B) {
	for b.Loop() {
		semver.MustParse("1.2.3-alpha.1+build.123")
	}
}
