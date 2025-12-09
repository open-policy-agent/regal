package fixes

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/regal/pkg/report"
)

func TestRedundantExistenceCheck(t *testing.T) {
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

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			contentAfterFix: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected:    false,
			runtimeOptions: &RuntimeOptions{},
		},
		"single change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			contentAfterFix: `package test

employee if {
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 4, Column: 2}}},
		},
		"bad change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			contentAfterFix: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected:    false,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 4, Column: 1000}}},
		},
		"many changes": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}

is_admin(user) if {
    user
    "admin" in user.roles
}`,
			},
			contentAfterFix: `package test

employee if {
    endswith(input.user.email, "@acmecorp.com")
}

is_admin(user) if {
    "admin" in user.roles
}`,
			fixExpected:    true,
			runtimeOptions: &RuntimeOptions{Locations: []report.Location{{Row: 4, Column: 2}, {Row: 9, Column: 2}}},
		},
	}
	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			rec := RedundantExistenceCheck{}

			fixResults, err := rec.Fix(tc.fc, tc.runtimeOptions)
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
