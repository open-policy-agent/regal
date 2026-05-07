package testgen_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/testgen"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestCreateTestModule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		policy      string
		inputJSON   string
		expectedErr string
		expected    []string
	}{
		{
			name:        "empty module returns error",
			policy:      `package policy`,
			expectedErr: "no rules found in this file",
		},
		{
			name: "single rule with no input deps creates basic test",
			policy: `package policy

allow if {
	1 == 1
}`,
			expected: []string{
				"package policy_test",
				"import data.policy",
				"test_allow if",
				"policy.allow",
			},
		},
		{
			name: "single rule with input deps and input.json adds with-clauses",
			policy: `package policy

allow if {
	input.foo == "woo"
	input.bar == "wee"
}`,
			inputJSON: `{"foo": "woo", "bar": "wee"}`,
			expected: []string{
				"test_allow if",
				"policy.allow",
				`input.foo as "woo"`,
				`input.bar as "wee"`,
			},
		},
		{
			name: "single rule with input deps but no input.json has no with-clauses",
			policy: `package policy

allow if {
	input.foo == "woo"
}`,
			expected: []string{
				"test_allow if",
				"policy.allow",
			},
		},
		{
			name: "multi-rule module produces a test for each rule",
			policy: `package policy

allow if {
	input.foo == "woo"
}

deny if {
	input.foo == "wee"
}`,
			inputJSON: `{"foo": "woo"}`,
			expected: []string{
				"package policy_test",
				"test_allow if",
				"test_deny if",
				"policy.allow",
				"policy.deny",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{"policy.rego": tc.policy}
			if tc.inputJSON != "" {
				files["input.json"] = tc.inputJSON
			}

			workspacePath := testutil.TempDirectoryOf(t, files)
			policyPath := filepath.Join(workspacePath, "policy.rego")
			fileURI := uri.FromPath(clients.IdentifierGoTest, policyPath)

			module, err := rparse.Module("policy.rego", tc.policy)
			if err != nil {
				t.Fatalf("failed to parse policy: %v", err)
			}

			result, ruleErrs, err := testgen.CreateTestModule(testgen.TestModuleOptions{
				Module:        module,
				AllModules:    map[string]*ast.Module{fileURI: module},
				WorkspacePath: workspacePath,
				FileURI:       fileURI,
			})

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (result=%q)", tc.expectedErr, result)
				}

				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error containing %q, got %v", tc.expectedErr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, re := range ruleErrs {
				t.Logf("rule %s failed (may or may not be intentional due to array access limitations): %v", re.RuleName, re.Err)
			}

			for _, want := range tc.expected {
				if !strings.Contains(result, want) {
					t.Errorf("result missing expected substring %q\n--- full output ---\n%s", want, result)
				}
			}
		})
	}
}
