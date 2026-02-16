package lsp

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
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
		},
		"empty file": {
			fileURI:     "file:///empty.rego",
			content:     "",
			regoVersion: ast.RegoV1,
			expectedParseErrors: []types.Diagnostic{{
				Code:  "rego-parse-error",
				Range: types.RangeBetween(0, 0, 0, 0),
			}},
		},
		"parse error due to version": {
			fileURI: "file:///valid.rego",
			content: `package test
allow if { 1 == 1 }
`,
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

			success := must.Return(updateParse(t.Context(), updateParseOpts{
				Cache:            c,
				Store:            NewRegalStore(),
				FileURI:          testData.fileURI,
				Builtins:         ast.BuiltinMap,
				RegoVersion:      testData.regoVersion,
				ClientIdentifier: clients.IdentifierGeneric,
			}))(t)

			must.Equal(t, testData.expectSuccess, success, "success")

			if _, ok := c.GetModule(testData.fileURI); testData.expectModule && !ok {
				t.Fatalf("expected module to be set, but it was not")
			}

			diags, _ := c.GetParseErrors(testData.fileURI)
			must.Equal(t, len(testData.expectedParseErrors), len(diags), "number of parse errors")

			for i, diag := range testData.expectedParseErrors {
				assert.Equal(t, diag.Code, diags[i].Code, "diagnostic code")
				assert.Equal(t, diag.Range.Start.Line, diags[i].Range.Start.Line, "diagnostic start line")
				assert.Equal(t, diag.Range.End.Line, diags[i].Range.End.Line, "diagnostic end line")
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
		"file1": {{
			Severity: new(uint(2)),
			Range:    getRangeForViolation(violation1),
			Message:  "Mock Error",
			Source:   new("regal/mock_category"),
			Code:     "mock_title",
			CodeDescription: &types.CodeDescription{
				Href: "https://www.openpolicyagent.org/projects/regal/rules/mock_category/mock_title",
			},
		}},
		"workspaceRootURI": {{
			Severity: new(uint(3)),
			Range:    getRangeForViolation(violation2),
			Message:  "Mock Warning",
			Source:   new("regal/mock_category"),
			Code:     "mock_title",
			CodeDescription: &types.CodeDescription{
				Href: "https://www.openpolicyagent.org/projects/regal/rules/mock_category/mock_title",
			},
		}},
	}

	assert.DeepEqual(t, expectedFileDiags, convertReportToDiagnostics(rpt, "workspaceRootURI"), "file diagnostics")
}

func TestLintWithConfigIgnoreWildcards(t *testing.T) {
	t.Parallel()

	rule := map[string]config.Category{"idiomatic": {"directory-package-mismatch": config.Rule{Level: "ignore"}}}
	conf := &config.Config{Rules: rule}

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

	must.Equal(t, nil, updateFileDiagnostics(t.Context(), opts))

	diagnostics, _ := state.GetFileDiagnostics(fileURI)

	must.Equal(t, 1, len(diagnostics), "number of diagnostics")
	assert.Equal(t, "prefer-snake-case", diagnostics[0].Code, "diagnostic code")

	// Clear the diagnostic and update the config with a wildcard ignore
	// for any file in the ignore directory.
	state.SetFileDiagnostics(fileURI, []types.Diagnostic{})

	conf.Rules["style"] = config.Category{
		"prefer-snake-case": config.Rule{
			Level:  "error",
			Ignore: &config.Ignore{Files: []string{"ignore/**"}},
		},
	}
	opts.UpdateForRules = []string{"prefer-snake-case"}

	must.Equal(t, nil, updateFileDiagnostics(t.Context(), opts))

	if diagnostics, _ := state.GetFileDiagnostics(fileURI); len(diagnostics) != 0 {
		t.Fatalf("Expected no diagnostics, got %v", diagnostics)
	}
}
