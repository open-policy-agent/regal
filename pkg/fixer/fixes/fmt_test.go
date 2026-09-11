package fixes

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"
)

func TestFmt(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		fmt             *Fmt
		fc              *FixCandidate
		contentAfterFix string
		fixExpected     bool
	}{
		"no change": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package testutil\n"},
			contentAfterFix: "package testutil\n",
			fixExpected:     false,
			fmt:             &Fmt{},
		},
		"add a new line": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package testutil"},
			contentAfterFix: "package testutil\n",
			fmt:             &Fmt{},
			fixExpected:     true,
		},
		"add a new line before rule": {
			fc:              &FixCandidate{Filename: "test.rego", Contents: "package testutil\nallow := true"},
			contentAfterFix: "package testutil\n\nallow := true\n",
			fmt:             &Fmt{},
			fixExpected:     true,
		},
		"rego version unknown, ambigous syntax": {
			fc: &FixCandidate{
				Filename:    "test.rego",
				Contents:    "package testutil\nallow := true",
				RegoVersion: ast.RegoUndefined,
			},
			fmt:         &Fmt{},
			fixExpected: true,
			contentAfterFix: `package testutil

allow := true
`,
		},
		"rego version unknown, v0 syntax": {
			fc: &FixCandidate{
				Filename:    "test.rego",
				Contents:    "package testutil\nallow[msg] { msg := 1}",
				RegoVersion: ast.RegoUndefined,
			},
			fmt:         &Fmt{},
			fixExpected: true,
			contentAfterFix: `package testutil

import rego.v1

allow contains msg if msg := 1
`,
		},
		"rego version unknown, v0v1 compat syntax": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package testutil
import rego.v1
allow contains msg if msg :=1
				`,
				RegoVersion: ast.RegoUndefined,
			},
			fmt:         &Fmt{},
			fixExpected: true,
			contentAfterFix: `package testutil

import rego.v1

allow contains msg if msg := 1
`,
		},
		"rego version unknown, v1 syntax": {
			fc: &FixCandidate{
				Filename: "test.rego",
				Contents: `package testutil

allow contains msg if msg :=1
				`,
				RegoVersion: ast.RegoUndefined,
			},
			fmt:         &Fmt{},
			fixExpected: true,
			contentAfterFix: `package testutil

allow contains msg if msg := 1
`,
		},
		"rego v1 (rego version 0)": {
			fc: &FixCandidate{
				Filename:    "test.rego",
				Contents:    "package testutil\nallow := true",
				RegoVersion: ast.RegoV0,
			},
			fmt:         &Fmt{},
			fixExpected: true,
			contentAfterFix: `package testutil

import rego.v1

allow := true
`,
		},
		"rego v1 (rego version > 1)": {
			fc: &FixCandidate{Filename: "test.rego", Contents: "package testutil\n\nallow := true\n"},
			fmt: &Fmt{
				OPAFmtOpts: format.Opts{
					RegoVersion: ast.RegoV0CompatV1,
				},
			},
			fixExpected: false,
		},
		"rego v1, version known": {
			fc: &FixCandidate{Filename: "test.rego", Contents: "package testutil\n\nallow := true\n"},
			fmt: &Fmt{
				OPAFmtOpts: format.Opts{
					RegoVersion: ast.RegoV1,
				},
			},
			fixExpected: false,
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			fixResults, err := tc.fmt.Fix(tc.fc, &RuntimeOptions{
				BaseDir: "",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.fixExpected && len(fixResults) != 0 {
				t.Fatal("unexpected fix applied")
			}

			if !tc.fixExpected {
				return
			}

			fixedContent := fixResults[0].Contents

			if fixedContent != tc.contentAfterFix {
				t.Fatalf(
					"unexpected content, got:\n%s---\nexpected:\n%s---",
					fixedContent,
					tc.contentAfterFix,
				)
			}
		})
	}
}
