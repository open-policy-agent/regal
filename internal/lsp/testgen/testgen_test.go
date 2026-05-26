package testgen_test

import (
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/input"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/store"
	"github.com/open-policy-agent/regal/internal/lsp/testgen"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
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

			wsRootURI := uri.FromPath(clients.IdentifierGoTest, testutil.TempDirectoryOf(t, files))
			workspace := workspace.New(wsRootURI)
			fileURI := uri.FromPath(clients.IdentifierGoTest, workspace.Path("policy.rego"))

			module, err := rparse.Module("policy.rego", tc.policy)
			if err != nil {
				t.Fatalf("failed to parse policy: %v", err)
			}

			im := input.NewManager(store.NewRegalStore(), log.NewLogger(log.LevelDebug, t.Output()))
			im.LoadFromWorkspace(t.Context(), workspace)

			result, err := testgen.CreateTestModule(t.Context(), testgen.TestModuleOptions{
				Module:        module,
				AllModules:    map[string]*ast.Module{fileURI: module},
				WorkspacePath: workspace.Path(),
				FileURI:       fileURI,
				InputManager:  im,
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
				t.Logf("rule errors (may or may not be intentional due to array access limitations): %v", err)
			}

			for _, want := range tc.expected {
				if !strings.Contains(result, want) {
					t.Errorf("result missing expected substring %q\n--- full output ---\n%s", want, result)
				}
			}
		})
	}
}
