package lsp

import (
	"reflect"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
)

func TestUpdateParse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fileURI string
		content string

		expectSuccess bool
		// ParseErrors are formatted as another type/source of diagnostic
		expectedParseErrors []types.Diagnostic
		expectModule        bool
		regoVersion         ast.RegoVersion
	}{
		"valid file": {
			fileURI: "file:///valid.rego",
			content: `package test
allow if { 1 == 1 }
`,
			expectModule:  true,
			expectSuccess: true,
			regoVersion:   ast.RegoV1,
		},
		"parse error": {
			fileURI: "file:///broken.rego",
			content: `package test

p = true { 1 == }
`,
			regoVersion: ast.RegoV1,
			expectedParseErrors: []types.Diagnostic{{
				Code:  "rego-parse-error",
				Range: types.RangeBetween(2, 13, 2, 13),
			}},
			expectModule:  false,
			expectSuccess: false,
		},
		"empty file": {
			fileURI:     "file:///empty.rego",
			content:     "",
			regoVersion: ast.RegoV1,
			expectedParseErrors: []types.Diagnostic{{
				Code:  "rego-parse-error",
				Range: types.RangeBetween(0, 0, 0, 0),
			}},
			expectModule:  false,
			expectSuccess: false,
		},
		"parse error due to version": {
			fileURI: "file:///valid.rego",
			content: `package test
allow if { 1 == 1 }
`,
			expectModule:  false,
			expectSuccess: false,
			expectedParseErrors: []types.Diagnostic{{
				Code:  "rego-parse-error",
				Range: types.RangeBetween(1, 0, 1, 0),
			}},
			regoVersion: ast.RegoV0,
		},
		"unknown rego version, rego v1 code": {
			fileURI: "file:///valid.rego",
			content: `package test
allow if { 1 == 1 }
`,
			expectModule:  true,
			expectSuccess: true,
			regoVersion:   ast.RegoUndefined,
		},
		"unknown rego version, rego v0 code": {
			fileURI: "file:///valid.rego",
			content: `package test
allow[msg] { 1 == 1; msg := "hello" }
`,
			expectModule:  true,
			expectSuccess: true,
			regoVersion:   ast.RegoUndefined,
		},
	}

	for testName, testData := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			c := cache.NewCache()
			c.SetFileContents(testData.fileURI, testData.content)

			success, err := updateParse(t.Context(), updateParseOpts{
				Cache:            c,
				Store:            NewRegalStore(),
				FileURI:          testData.fileURI,
				Builtins:         ast.BuiltinMap,
				RegoVersion:      testData.regoVersion,
				WorkspaceRootURI: "",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if success != testData.expectSuccess {
				t.Fatalf("expected success to be %v, got %v", testData.expectSuccess, success)
			}

			_, ok := c.GetModule(testData.fileURI)
			if testData.expectModule && !ok {
				t.Fatalf("expected module to be set, but it was not")
			}

			diags, _ := c.GetParseErrors(testData.fileURI)

			if len(testData.expectedParseErrors) != len(diags) {
				t.Fatalf("expected %v parse errors, got %v", len(testData.expectedParseErrors), len(diags))
			}

			for i, diag := range testData.expectedParseErrors {
				if diag.Code != diags[i].Code {
					t.Errorf("expected diagnostic code to be %v, got %v", diag.Code, diags[i].Code)
				}

				if diag.Range.Start.Line != diags[i].Range.Start.Line {
					t.Errorf("expected diagnostic start line to be %v, got %v", diag.Range.Start.Line, diags[i].Range.Start.Line)
				}

				if diag.Range.End.Line != diags[i].Range.End.Line {
					t.Errorf("expected diagnostic end line to be %v, got %v", diag.Range.End.Line, diags[i].Range.End.Line)
				}
			}
		})
	}
}

func TestConvertReportToDiagnostics(t *testing.T) {
	t.Parallel()

	violation1 := report.Violation{
		Level:       "error",
		Description: "Mock Error",
		Category:    "mock_category",
		Title:       "mock_title",
		Location:    report.Location{File: "file1"},
		IsAggregate: false,
	}
	violation2 := report.Violation{
		Level:       "warning",
		Description: "Mock Warning",
		Category:    "mock_category",
		Title:       "mock_title",
		Location:    report.Location{File: ""},
		IsAggregate: true,
	}

	rpt := &report.Report{Violations: []report.Violation{violation1, violation2}}

	expectedFileDiags := map[string][]types.Diagnostic{
		"file1": {
			{
				Severity: util.Pointer(uint(2)),
				Range:    getRangeForViolation(violation1),
				Message:  "Mock Error",
				Source:   util.Pointer("regal/mock_category"),
				Code:     "mock_title",
				CodeDescription: &types.CodeDescription{
					Href: "https://docs.styra.com/regal/rules/mock_category/mock_title",
				},
			},
		},
		"workspaceRootURI": {
			{
				Severity: util.Pointer(uint(3)),
				Range:    getRangeForViolation(violation2),
				Message:  "Mock Warning",
				Source:   util.Pointer("regal/mock_category"),
				Code:     "mock_title",
				CodeDescription: &types.CodeDescription{
					Href: "https://docs.styra.com/regal/rules/mock_category/mock_title",
				},
			},
		},
	}

	fileDiags := convertReportToDiagnostics(rpt, "workspaceRootURI")

	if !reflect.DeepEqual(fileDiags, expectedFileDiags) {
		t.Errorf("Expected file diagnostics: %v, got: %v", expectedFileDiags, fileDiags)
	}
}

func TestLintWithConfigIgnoreWildcards(t *testing.T) {
	t.Parallel()

	conf := &config.Config{
		Rules: map[string]config.Category{
			"idiomatic": {"directory-package-mismatch": config.Rule{Level: "ignore"}},
		},
	}

	contents := "package p\n\ncamelCase := 1\n"
	fileURI := "file:///workspace/ignore/p.rego"

	state := cache.NewCache()
	state.SetFileContents(fileURI, contents)
	state.SetModule(fileURI, parse.MustParseModule(contents))
	state.SetFileDiagnostics(fileURI, []types.Diagnostic{})

	opts := diagnosticsRunOpts{
		Cache:            state,
		RegalConfig:      conf,
		FileURI:          fileURI,
		WorkspaceRootURI: "file:///workspace",
		UpdateForRules:   []string{"prefer-snake-case"},
	}

	if err := updateFileDiagnostics(t.Context(), opts); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	diagnostics, _ := state.GetFileDiagnostics(fileURI)
	if len(diagnostics) != 1 {
		t.Fatalf("Expected one diagnostic item, got %v", diagnostics)
	}

	if diagnostics[0].Code != "prefer-snake-case" {
		t.Errorf("Expected diagnostic code to be prefer-snake-case, got %v", diagnostics[0].Code)
	}

	// Clear the diagnostic and update the config with a wildcard ignore
	// for any file in the ignore directory.

	state.SetFileDiagnostics(fileURI, []types.Diagnostic{})

	conf.Rules["style"] = config.Category{
		"prefer-snake-case": config.Rule{
			Level: "error",
			Ignore: &config.Ignore{
				Files: []string{"ignore/**"},
			},
		},
	}

	opts.UpdateForRules = []string{"prefer-snake-case"}

	if err := updateFileDiagnostics(t.Context(), opts); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diagnostics, _ := state.GetFileDiagnostics(fileURI); len(diagnostics) != 0 {
		t.Fatalf("Expected no diagnostics, got %v", diagnostics)
	}
}
