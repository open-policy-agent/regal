package lsp

import (
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

func TestEvalWorkspacePath(t *testing.T) {
	t.Parallel()

	workspace := workspace.New("file:///workspace").WithClient(client.NewGeneric())

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	ls.workspace = workspace

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

	policy1URI := workspace.URI("policy1.rego")
	policy1RelativeFileName := workspace.RelativePath(policy1URI)
	module1 := must.Return(rparse.ModuleWithOpts(policy1RelativeFileName, policy1, rparse.ParserOptions()))(t)

	policy2URI := workspace.URI("policy2.rego")
	policy2RelativeFileName := workspace.RelativePath(policy2URI)
	module2 := must.Return(rparse.ModuleWithOpts(policy2RelativeFileName, policy2, rparse.ParserOptions()))(t)

	ls.cache.SetFileContents(policy1URI, policy1)
	ls.cache.SetFileContents(policy2URI, policy2)
	ls.cache.SetModule(policy1URI, module1)
	ls.cache.SetModule(policy2URI, module2)

	input := ast.NewObject(rast.Item("exists", ast.InternedTerm(true)))

	value, _, err := ls.EvalInWorkspace(t.Context(), "data.policy1.allow", nil, rego.EvalParsedInput(input))
	res := must.Return(value, err)(t)
	assert.True(t, must.Be[bool](t, res.Value))

	expectedPrintOutput := map[string]map[int][]string{policy2URI: {4: {"1"}}}
	must.Equal(t, "", cmp.Diff(expectedPrintOutput, res.PrintOutput), "print output")
}

func TestEvalWorkspacePathWithCoverage(t *testing.T) {
	t.Parallel()

	workspace := workspace.New("file:///workspace").WithClient(client.NewGeneric())

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	ls.workspace = workspace

	policy1 := `package policy1

	default allow := false

	allow if input.exists
	`

	policy1URI := workspace.URI("policy1.rego")
	policy1RelativeFileName := workspace.RelativePath(policy1URI)
	module1 := must.Return(rparse.ModuleWithOpts(policy1RelativeFileName, policy1, rparse.ParserOptions()))(t)

	ls.cache.SetFileContents(policy1URI, policy1)
	ls.cache.SetModule(policy1URI, module1)

	input := ast.NewObject(rast.Item("exists", ast.InternedTerm(true)))

	value, report, err := ls.EvalInWorkspace(t.Context(), "data.policy1.allow", cover.New(), rego.EvalParsedInput(input))
	res := must.Return(value, err)(t)
	assert.True(t, must.Be[bool](t, res.Value))

	if report == nil {
		t.Fatal("expected a coverage report, got nil")
	}

	fileReport := report.Files[policy1RelativeFileName]
	if fileReport == nil {
		t.Fatalf("expected a coverage report entry for %q", policy1RelativeFileName)
	}

	assert.True(t, fileReport.CoveredLines > 0)
}

func TestEvalWorkspacePathInternalData(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})

	value, _, err := ls.EvalInWorkspace(
		t.Context(), "object.keys(data.internal)", nil, rego.EvalParsedInput(ast.InternedEmptyObjectValue),
	)
	res := must.Return(value, err)(t)
	val := must.Be[[]any](t, res.Value)
	act := outil.Sorted(must.Return(util.AnySliceTo[string](val))(t))

	assert.SlicesEqual(t, []string{"capabilities", "combined_config", "user_config"}, act)
}
