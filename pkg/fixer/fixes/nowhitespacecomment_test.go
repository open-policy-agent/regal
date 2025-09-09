package fixes

import (
	"testing"

	"github.com/open-policy-agent/regal/pkg/report"
)

func TestNoWhitespaceComment(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		contentAfterFix string
		fc              *FixCandidate
		fixExpected     bool
		runtimeOptions  *RuntimeOptions
	}{
		"no change needed": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package test\n\n# this is a comment\n"},
			contentAfterFix: "package test\n\n# this is a comment\n",
			fixExpected:     false,
			runtimeOptions:  &RuntimeOptions{},
		},
		"single change": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package test\n\n#this is a comment\n"},
			contentAfterFix: "package test\n\n# this is a comment\n",
			fixExpected:     true,
			runtimeOptions:  &RuntimeOptions{Locations: []report.Location{{Row: 3, Column: 1}}},
		},
		"many changes": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test\n

#this is a comment
#this is a comment
#this is a comment
`,
			},
			contentAfterFix: `package test\n

# this is a comment
# this is a comment
# this is a comment
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{{Row: 3, Column: 1}, {Row: 4, Column: 1}, {Row: 5, Column: 1}},
			},
		},
		"many changes, different columns": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package test\n

#this is a comment
 #this is a comment
  #this is a comment
`,
			},
			contentAfterFix: `package test\n

# this is a comment
 # this is a comment
  # this is a comment
`,
			fixExpected: true,
			runtimeOptions: &RuntimeOptions{
				Locations: []report.Location{{Row: 3, Column: 1}, {Row: 4, Column: 2}, {Row: 5, Column: 3}},
			},
		},
	}

	for testName, tc := range testCases {
		nwc := NoWhitespaceComment{}

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			fixResults, err := nwc.Fix(tc.fc, tc.runtimeOptions)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.fixExpected && len(fixResults) != 0 {
				t.Fatalf("unexpected fix applied")
			}

			if !tc.fixExpected {
				return
			}

			if tc.fixExpected && fixResults[0].Contents != tc.contentAfterFix {
				t.Fatalf("unexpected content, got:\n%s---\nexpected:\n%s---", fixResults[0].Contents, tc.contentAfterFix)
			}
		})
	}
}
