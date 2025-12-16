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
			runtimeOptions:  &RuntimeOptions{},
			fixExpected:     false,
			contentAfterFix: "package test\n\nallow = true\n",
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
			runtimeOptions: &RuntimeOptions{},
			fixExpected:    false,
			contentAfterFix: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
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
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 5, End: &report.Position{
							Row: 4, Column: 21,
						},
					},
				},
			},
			fixExpected: true,
			contentAfterFix: `package test

employee if {
    
    endswith(input.user.email, "@acmecorp.com")
}`,
		},
		"no change because bad location": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 1000, End: &report.Position{
							Row: 4, Column: 1000,
						},
					},
				},
			},
			fixExpected: false,
			contentAfterFix: `package test

employee if {
    input.user.email
    endswith(input.user.email, "@acmecorp.com")
}`,
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
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 5, End: &report.Position{
							Row: 4, Column: 21,
						},
					},
					{
						Row: 9, Column: 5, End: &report.Position{
							Row: 4, Column: 9,
						},
					},
				},
			},
			fixExpected: true,
			contentAfterFix: `package test

employee if {
    
    endswith(input.user.email, "@acmecorp.com")
}

is_admin(user) if {
    
    "admin" in user.roles
}`,
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
				t.Fatalf("unexpected content:\n%s", diff)
			}
		})
	}
}
