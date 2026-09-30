package lsp

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/cover"
	"github.com/open-policy-agent/opa/v1/rego"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/lsp/client"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func TestEvalWorkspace(t *testing.T) {
	t.Parallel()

	policy1 := `package policy1

	import data.policy2

	default allow := false

	allow if policy2.allow
	`

	policy2 := `package policy2

	allow if {
		print(1)
		input.exists
	}
	`

	ls, modules := evalTestServer(t, map[string]string{
		"policy1.rego": policy1,
		"policy2.rego": policy2,
	})
	module1 := modules["policy1.rego"]

	input := ast.NewObject(rast.Item("exists", ast.InternedTerm(true)))

	value, _, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{
		Query:    "data.policy1.allow",
		Package:  module1.Package,
		RegoOpts: []rego.EvalOption{rego.EvalParsedInput(input)},
	})
	res := must.Return(value, err)(t)
	assert.True(t, must.Be[bool](t, res.Value))

	policy2URI := ls.Workspace().URI("policy2.rego")
	expectedPrintOutput := map[string]map[int][]string{policy2URI: {4: {"1"}}}
	must.Equal(t, "", cmp.Diff(expectedPrintOutput, res.PrintOutput), "print output")
}

func TestEvalWorkspaceValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query         string
		wantUndefined bool
		wantValue     any
	}{
		"false is a defined value, not undefined": {
			query:     "false",
			wantValue: false,
		},
		"equality comparison false is a defined value": {
			query:     "1 == 2",
			wantValue: false,
		},
		"ending in assignment": {
			query:     "x := 1; y := x + 1",
			wantValue: json.Number("2"),
		},
		"equality, variable on right": {
			query:     "1 = x",
			wantValue: json.Number("1"),
		},
		"unification failure has no results": {
			query:         "1 = 2",
			wantUndefined: true,
		},
		"whitespace-only selection has no results": {
			query:         "   \n  ",
			wantUndefined: true,
		},
		"comment-only selection has no results": {
			query:         "# just a comment\n",
			wantUndefined: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})

			value, _, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{Query: tc.query})
			res := must.Return(value, err)(t)

			assert.Equal(t, tc.wantUndefined, res.IsUndefined)

			if !tc.wantUndefined {
				assert.DeepEqual(t, tc.wantValue, res.Value)
			}
		})
	}
}

func TestEvalWorkspaceWithCoverage(t *testing.T) {
	t.Parallel()

	policy1 := `package policy1

	default allow := false

	allow if input.exists
	`

	cases := map[string]struct {
		query string
	}{
		"rule reference": {
			query: "data.policy1.allow",
		},
		"multi-statement query": {
			query: "x := data.policy1.allow; x",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ls, modules := evalTestServer(t, map[string]string{"policy1.rego": policy1})
			module1 := modules["policy1.rego"]

			input := ast.NewObject(rast.Item("exists", ast.InternedTerm(true)))

			value, report, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{
				Query:    tc.query,
				Package:  module1.Package,
				Coverage: cover.New(),
				RegoOpts: []rego.EvalOption{rego.EvalParsedInput(input)},
			})
			res := must.Return(value, err)(t)
			assert.True(t, must.Be[bool](t, res.Value))

			if report == nil {
				t.Fatal("expected a coverage report, got nil")
			}

			policy1RelativeFileName := ls.Workspace().RelativePath(ls.Workspace().URI("policy1.rego"))

			fileReport := report.Files[policy1RelativeFileName]
			if fileReport == nil {
				t.Fatalf("expected a coverage report entry for %q", policy1RelativeFileName)
			}

			assert.True(t, fileReport.CoveredLines > 0)
		})
	}
}

func TestEvalWorkspaceInternalData(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})

	value, _, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{
		Query:    "object.keys(data.internal)",
		RegoOpts: []rego.EvalOption{rego.EvalParsedInput(ast.InternedEmptyObjectValue)},
	})
	res := must.Return(value, err)(t)
	val := must.Be[[]any](t, res.Value)
	act := outil.Sorted(must.Return(util.AnySliceTo[string](val))(t))

	assert.SlicesEqual(t, []string{"capabilities", "combined_config", "user_config"}, act)
}

func TestEvalWorkspacePackageRelativeQuery(t *testing.T) {
	t.Parallel()

	policy1 := `package policy1

	allow := true
	`

	ls, modules := evalTestServer(t, map[string]string{"policy1.rego": policy1})
	module1 := modules["policy1.rego"]

	// "allow", not "data.policy1.allow": this only resolves because Package is set to
	// module1.Package.
	value, _, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{Query: "allow", Package: module1.Package})
	res := must.Return(value, err)(t)
	assert.True(t, must.Be[bool](t, res.Value))
}

func TestEvalWorkspaceImportedAlias(t *testing.T) {
	t.Parallel()

	// This module's own import (of policy2) is included alongside the aliased import
	// added below, since ParsedImports receives every import from the module, not just
	// the one under test.
	policy1 := `package policy1

	import data.policy2

	allow if 1 in [1, 2]
	`

	ls, modules := evalTestServer(t, map[string]string{"policy1.rego": policy1})
	module1 := modules["policy1.rego"]

	imports := must.Return(ast.ParseImports("import data.policy1 as p"))(t)
	imports = append(imports, module1.Imports...)

	value, _, err := ls.EvalInWorkspace(t.Context(), EvalWorkspaceOptions{Query: "p.allow", Imports: imports})
	res := must.Return(value, err)(t)
	assert.True(t, must.Be[bool](t, res.Value))
}

func TestFinalExpressionValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query     string
		exprValue any
		bindings  map[string]any
		expected  any
	}{
		"rule reference": {
			query:     "data.pkg.rule",
			exprValue: true,
			expected:  true,
		},
		"bare variable reference": {
			query:     "x := 1; y := x + 1; y",
			exprValue: json.Number("2"),
			bindings:  map[string]any{"x": json.Number("1"), "y": json.Number("2")},
			expected:  json.Number("2"),
		},
		"ending in assignment": {
			query:     "x := 1; y := x + 1",
			exprValue: true,
			bindings:  map[string]any{"x": json.Number("1"), "y": json.Number("2")},
			expected:  json.Number("2"),
		},
		"equality, variable on right": {
			query:     "1 = x",
			exprValue: true,
			bindings:  map[string]any{"x": json.Number("1")},
			expected:  json.Number("1"),
		},
		"destructuring assignment": {
			query:     "[a, b] := [1, 2]",
			exprValue: true,
			bindings:  map[string]any{"a": json.Number("1"), "b": json.Number("2")},
			expected:  true,
		},
		"multi-line query with comments": {
			query:     "# leading comment\nx := 1\ny := x + 1 # trailing comment\ny",
			exprValue: json.Number("2"),
			bindings:  map[string]any{"x": json.Number("1"), "y": json.Number("2")},
			expected:  json.Number("2"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := must.Return(ast.ParseBody(tc.query))(t)
			result := rego.Result{
				Expressions: []*rego.ExpressionValue{{Value: tc.exprValue}},
				Bindings:    tc.bindings,
			}

			assert.DeepEqual(t, tc.expected, finalExpressionValue(body, result))
		})
	}
}

func evalTestServer(t *testing.T, modules map[string]string) (*LanguageServer, map[string]*ast.Module) {
	t.Helper()

	workspace := workspace.New("file:///workspace").WithClient(client.NewGeneric())

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	ls.workspace = workspace

	parsed := make(map[string]*ast.Module, len(modules))

	for filename, source := range modules {
		fileURI := workspace.URI(filename)
		relativeFileName := workspace.RelativePath(fileURI)
		module := must.Return(rparse.ModuleWithOpts(relativeFileName, source, rparse.ParserOptions()))(t)

		ls.cache.SetFileContents(fileURI, source)
		ls.cache.SetModule(fileURI, module)

		parsed[filename] = module
	}

	return ls, parsed
}
