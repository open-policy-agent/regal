package lsp

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/cover"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/lsp/client"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
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

	assert.True(t, ast.Boolean(true).Equal(res.Value))

	policy2URI := ls.Workspace().URI("policy2.rego")
	expectedPrintOutput := PrintOutput{policy2URI: {4: {"1"}}}

	must.Equal(t, "", cmp.Diff(expectedPrintOutput, res.PrintOutput), "print output")
}

func TestEvalWorkspaceValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query         string
		wantUndefined bool
		wantValue     ast.Value
	}{
		"false is a defined value, not undefined": {
			query:     "false",
			wantValue: ast.Boolean(false),
		},
		"equality comparison false is a defined value": {
			query:     "1 == 2",
			wantValue: ast.Boolean(false),
		},
		"ending in assignment": {
			query:     "x := 1; y := x + 1",
			wantValue: ast.Number("2"),
		},
		"equality, variable on right": {
			query:     "1 = x",
			wantValue: ast.Number("1"),
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
		"rule reference":        {query: "data.policy1.allow"},
		"multi-statement query": {query: "x := data.policy1.allow; x"},
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

			assert.True(t, ast.Boolean(true).Equal(res.Value))
			must.NotEqual(t, nil, report, "expected a coverage report, got nil")

			ws := ls.Workspace()
			policy1RelativeFileName := ws.RelativePath(ws.URI("policy1.rego"))
			fileReport := report.Files[policy1RelativeFileName]

			must.NotEqual(t, nil, fileReport, "expected a coverage report entry for %q", policy1RelativeFileName)
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
	arr := must.Be[ast.Set](t, res.Value).Sorted()

	assert.Equal(t, `["capabilities", "combined_config", "user_config"]`, arr.String())
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

	assert.True(t, ast.Boolean(true).Equal(res.Value))
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

	assert.True(t, ast.Boolean(true).Equal(res.Value))
}

func TestFinalExpressionValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query     string
		exprValue any
		bindings  map[string]any
		expected  ast.Value
	}{
		"rule reference Go expression value": {
			query:     "data.pkg.rule",
			exprValue: true,
			expected:  ast.Boolean(true),
		},
		"rule reference AST expression value": {
			query:     "data.pkg.rule",
			exprValue: ast.Boolean(true),
			expected:  ast.Boolean(true),
		},
		"bare variable reference": {
			query:     "x := 1; y := x + 1; y",
			exprValue: ast.Number("2"),
			bindings:  map[string]any{"x": 1, "y": 2},
			expected:  ast.Number("2"),
		},
		"ending in assignment": {
			query:     "x := 1; y := x + 1",
			exprValue: true,
			bindings:  map[string]any{"x": 1, "y": 2},
			expected:  ast.Number("2"),
		},
		"equality, variable on right": {
			query:     "1 = x",
			exprValue: true,
			bindings:  map[string]any{"x": 1},
			expected:  ast.Number("1"),
		},
		"destructuring assignment": {
			query:     "[a, b] := [1, 2]",
			exprValue: true,
			bindings:  map[string]any{"a": 1, "b": 2},
			expected:  ast.Boolean(true),
		},
		"multi-line query with comments": {
			query:     "# leading comment\nx := 1\ny := x + 1 # trailing comment\ny",
			exprValue: 2,
			bindings:  map[string]any{"x": 1, "y": 2},
			expected:  ast.Number("2"),
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

			assert.True(t, ast.ValueEqual(tc.expected, finalExpressionValue(body, result)))
		})
	}
}

func TestEvalInWorkspaceHandler(t *testing.T) {
	t.Parallel()

	ls := evalTestServerWithFS(t)

	must.Equal(t, nil, ls.handleEvalCommand(t.Context(), types.CommandArgs{
		Target: ls.workspace.URI("policy1.rego"),
		Query:  "data.policy1.allow",
		Row:    7,
	}))

	exp := `{
		"a": 1,
		"b": 3.1,
		"c": "string",
		"d": true,
		"e": null,
		"f": [1, 2, 3],
		"g": {"nested": "object", "array": [1, false, 3], "set": [1, 2, 3]}
	}`

	assert.JSONEqual(t, exp, must.ReadFile(t, ls.workspace.Path("output.json")))
}

// Note: cost is almost entirely from compiling / building the bundle
// 1331922 ns/op	 1436070 B/op	   19098 allocs/op
// _342727 ns/op	  281612 B/op	    5656 allocs/op // use store instead of building bundle
func BenchmarkEvalInWorkspaceHandler(b *testing.B) {
	ls := evalTestServerWithFS(b)
	args := types.CommandArgs{
		Target: ls.workspace.URI("policy1.rego"),
		Query:  "data.policy1.allow",
		Row:    7,
	}

	// result is verified by the test above.. no need to check it here.
	for b.Loop() {
		_ = ls.handleEvalCommand(b.Context(), args)
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
		module := must.Return(rparse.ModuleWithOpts(relativeFileName, source, rparse.Options()))(t)

		ls.cache.SetFileContents(fileURI, source)
		ls.cache.SetModule(fileURI, module)

		parsed[filename] = module
	}

	return ls, parsed
}

func evalTestServerWithFS(tb testing.TB) *LanguageServer {
	tb.Helper()

	policy1 := `package policy1

	import data.policy2

	obj := {
		"a": 1,
		"b": 3.1,
		"c": "string",
		"d": true,
		"e": null,
		"f": [1, 2, 3],
		"g": {"nested": "object", "array": [1, false, 3], "set": {1, 2, 3}},
	}

	allow := obj if policy2.allow
	`

	policy2 := `package policy2

	allow if {
		print(1)
		input.exists
	}
	`

	mods := map[string]string{
		"policy1.rego": policy1,
		"policy2.rego": policy2,
		"input.json":   `{"exists": true}`,
	}
	root := testutil.TempDirectoryOf(tb, mods)

	ls := NewLanguageServer(tb.Context(), &LanguageServerOptions{Logger: test.DebugLogger(tb)})
	ls.workspace = workspace.New("file://" + root)

	ls.input.LoadFromWorkspace(tb.Context(), ls.workspace)

	for filename, source := range mods {
		if !strings.HasSuffix(filename, ".rego") {
			continue
		}

		fileURI := ls.workspace.URI(filename)
		relPath := ls.workspace.RelativePath(fileURI)

		ls.cache.SetFileContents(fileURI, source)
		ls.cache.SetModule(fileURI, must.Return(rparse.ModuleWithOpts(relPath, source, rparse.Options()))(tb))
	}

	return ls
}
