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
			contentAfterFix: "package test\n\nallow := true\n",
			fixExpected:     false,
			runtimeOptions:  &RuntimeOptions{},
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
			contentAfterFix: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected:    false,
			runtimeOptions: &RuntimeOptions{},
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
			contentAfterFix: `package test

allow if {
    
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 5, End: &report.Position{
							Row: 4, Column: 9,
						},
					},
				},
			},
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
			contentAfterFix: `package test

allow if {
    true
    endswith(input.user.email, "@acmecorp.com")
}`,
			fixExpected: false,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 4, Column: 1000, End: &report.Position{
							Row: 4, Column: 1004,
						},
					},
				},
			},
		},
		"single line": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test

allow if { true }`,
			},
			contentAfterFix: `package test

allow if {  }`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{
					{
						Row: 3, Column: 12, End: &report.Position{
							Row: 4, Column: 16,
						},
					},
				},
			},
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
			contentAfterFix: `package test

allow if {
    
    endswith(input.user.email, "@acmecorp.com")
    
}`,
			fixExpected: true,
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
				t.Fatalf(
					"unexpected content, got:\n%s---\nexpected:\n%s---",
					fixResults[0].Contents,
					tc.contentAfterFix,
				)
			}
		})
	}
}
