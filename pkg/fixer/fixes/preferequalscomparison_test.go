package fixes

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/regal/pkg/report"
)

func TestPreferEqualsComparison(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		contentAfterFix string
		fc              *FixCandidate
		fixExpected     bool
		runtimeOptions  *RuntimeOptions
	}{
		"no change": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package test\n\nallow = true\n"},
			contentAfterFix: "package test\n\nallow = true\n",
			fixExpected:     false,
			runtimeOptions:  &RuntimeOptions{},
		},
		"no change because no location": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test
allow := true

test_rule {
	allow = true
}`,
			},
			contentAfterFix: `package test
allow := true

test_rule {
	allow = true
}`,
			fixExpected:    false,
			runtimeOptions: &RuntimeOptions{},
		},
		"single change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test
allow := true

test_rule {
	allow = true
}`,
			},
			contentAfterFix: `package test
allow := true

test_rule {
	allow == true
}`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 5, Column: 2}}},
		},
		"bad change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test
allow := true
allow = true

test_rule {
	allow = false
}`,
			},
			contentAfterFix: `package test
allow := true
allow = true

test_rule {
	allow = false
}`,
			fixExpected:    false,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 1, Column: 1}}},
		},
		"many changes": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test
allow := true

test_rule {
	allow = true
	allow = false
}`,
			},
			contentAfterFix: `package test
allow := true

test_rule {
	allow == true
	allow == false
}`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 5, Column: 2}, {Row: 6, Column: 2}}},
		},
		"different columns": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test
allow := true

test_rule {
	allow = true
		allow = false
}`,
			},
			contentAfterFix: `package test
allow := true

test_rule {
	allow == true
		allow == false
}`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 5, Column: 2}, {Row: 6, Column: 2}}},
		},
		"multiple = signs": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

is_get := true if input.request.method = "GET"
`,
			},
			contentAfterFix: `package test

is_get := true if input.request.method == "GET"
`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 3, Column: 40}}},
		},
	}
	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			pec := PreferEqualsComparison{}

			fixResults, err := pec.Fix(tc.fc, tc.runtimeOptions)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.fixExpected && len(fixResults) != 0 {
				t.Fatalf("unexpected fix applied")
			}

			if !tc.fixExpected {
				return
			}

			if diff := cmp.Diff(fixResults[0].Contents, tc.contentAfterFix); tc.fixExpected && diff != "" {
				t.Fatalf(
					"unexpected content, got:\n%s---\nexpected:\n%s---",
					fixResults[0].Contents,
					tc.contentAfterFix,
				)
			}
		})
	}
}
