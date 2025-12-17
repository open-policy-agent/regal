package fixes

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/regal/pkg/report"
)

func TestConstantCondition(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		contentAfterFix string
		fc              *FixCandidate
		fixExpected     bool
		runtimeOptions  *RuntimeOptions
	}{
		"no change": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package test\n\nallow := true\n"},
			runtimeOptions:  &RuntimeOptions{},
			fixExpected:     false,
			contentAfterFix: "package test\n\nallow := true\n",
		},
		"no change because no location": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			runtimeOptions: &RuntimeOptions{},
			fixExpected:    false,
			contentAfterFix: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
		},
		"single change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 5, End: &report.Position{
							Row: 4, Column: 9,
						},
					},
				},
			},
			fixExpected: true,
			contentAfterFix: `package test

allow if {
    
    endswith(input.user.email, "@acmecorp.com")
}`,
		},
		"bad change": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
			},
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 1000, End: &report.Position{
							Row: 4, Column: 1004,
						},
					},
				},
			},
			fixExpected: false,
			contentAfterFix: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
		},
		"single line": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if { true }`,
			},
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 3, Column: 12, End: &report.Position{
							Row: 4, Column: 16,
						},
					},
				},
			},
			fixExpected: true,
			contentAfterFix: `package test

allow if {  }`,
		},
		"many changes": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
    1 == 1
}`,
			},
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 5, End: &report.Position{
							Row: 4, Column: 9,
						},
					},
					{
						Row: 6, Column: 5, End: &report.Position{
							Row: 4, Column: 11,
						},
					},
				},
			},
			fixExpected: true,
			contentAfterFix: `package test

allow if {
    
    endswith(input.user.email, "@acmecorp.com")
    
}`,
		},
	}
	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			cc := ConstantCondition{}

			fixResults, err := cc.Fix(tc.fc, tc.runtimeOptions)
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
