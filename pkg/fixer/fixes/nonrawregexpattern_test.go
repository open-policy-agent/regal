package fixes

import (
	"testing"

	"github.com/open-policy-agent/regal/pkg/report"
)

func TestNonRawRegexPattern(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		contentAfterFix string
		fc              *FixCandidate
		fixExpected     bool
		runtimeOptions  *RuntimeOptions
	}{
		"none needed, no location": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
    regex.match(` + "`" + `[\d]+` + "`" + `, "12345")
}
`},
			contentAfterFix: `package test

all_digits if {
    regex.match(` + "`" + `[\d]+` + "`" + `, "12345")
}
`,
			fixExpected: false,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{},
			},
		},
		"none needed, but location given": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
    regex.match(` + "`" + `[\d]+` + "`" + `, "12345")
}
`},
			contentAfterFix: `package test

all_digits if {
    regex.match(` + "`" + `[\d]+` + "`" + `, "12345")
}
`,
			fixExpected: false,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{{Row: 4, Column: 17, End: &report.Position{Row: 4, Column: 24}}},
			},
		},
		"bad request, but location given": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
	# this is a comment, not a violation of the rule
}
`},
			contentAfterFix: `package test

all_digits if {
	# this is a comment, not a violation of the rule
}
`,
			fixExpected: false,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{{Row: 4, Column: 17, End: &report.Position{Row: 4, Column: 25}}},
			},
		},
		"simple case": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
    regex.match("[\\d]+", "12345")
}
`},
			contentAfterFix: `package test

all_digits if {
    regex.match(` + "`" + `[\d]+` + "`" + `, "12345")
}
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{{Row: 4, Column: 17, End: &report.Position{Row: 4, Column: 25}}},
			},
		},
		"two on one line": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
  set := { regex.match("*", "1"), regex.match("+", "2") }
}
`},
			contentAfterFix: `package test

all_digits if {
  set := { regex.match(` + "`" + `*` + "`" + `, "1"), regex.match(` + "`" + `+` + "`" + `, "2") }
}
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{Row: 4, Column: 47, End: &report.Position{Row: 4, Column: 50}},
					{Row: 4, Column: 24, End: &report.Position{Row: 4, Column: 27}},
				},
			},
		},
		"two on one line with backslashes": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
  set := { regex.match("\\d+", "1"), regex.match("\\d*", "2") }
}
`},
			contentAfterFix: `package test

all_digits if {
  set := { regex.match(` + "`" + `\d+` + "`" + `, "1"), regex.match(` + "`" + `\d*` + "`" + `, "2") }
}
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{Row: 4, Column: 50, End: &report.Position{Row: 4, Column: 56}},
					{Row: 4, Column: 24, End: &report.Position{Row: 4, Column: 30}},
				},
			},
		},
		"two on separate lines": {
			fc: &FixCandidate{Filename: "test.rego", Contents: `package test

all_digits if {
  set := {
    regex.match("*", "1"),
    regex.match("+", "2"),
  }
}
`},
			contentAfterFix: `package test

all_digits if {
  set := {
    regex.match(` + "`" + `*` + "`" + `, "1"),
    regex.match(` + "`" + `+` + "`" + `, "2"),
  }
}
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{Row: 5, Column: 17, End: &report.Position{Row: 5, Column: 20}},
					{Row: 6, Column: 17, End: &report.Position{Row: 4, Column: 20}},
				},
			},
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			f := NonRawRegexPattern{}

			fixResults, err := f.Fix(tc.fc, tc.runtimeOptions)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.fixExpected && len(fixResults) != 0 {
				t.Fatalf("unexpected fix applied")
			}

			if !tc.fixExpected {
				return
			}

			if len(fixResults) == 0 {
				t.Fatalf("expected fix to be applied")
			}

			fixedContent := fixResults[0].Contents

			if tc.fixExpected && fixedContent != tc.contentAfterFix {
				t.Fatalf(
					"unexpected content, got:\n%s---\nexpected:\n%s---",
					fixedContent,
					tc.contentAfterFix,
				)
			}
		})
	}
}
