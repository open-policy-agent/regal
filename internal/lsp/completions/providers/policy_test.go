package providers

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

// testCaseFileURI is used in various tests in the providers package.
const testCaseFileURI = "file:///foo/bar/file.rego"

//nolint:paralleltest
func TestPolicyProvider_Example1(t *testing.T) {
	policy := `package p

allow if {
	user := data.users[0]
	# try completion on next line
	roles := u
}
`
	module := parse.MustParseModule(policy)
	moduleMap := make(map[string]any)

	encoding.MustJSONRoundTrip(module, &moduleMap)

	c := cache.NewCache()
	c.SetFileContents(testCaseFileURI, policy)

	store := inmem.NewFromObjectWithOpts(map[string]any{
		"workspace": map[string]any{
			"parsed": map[string]any{
				testCaseFileURI: moduleMap,
			},
		},
	}, inmem.OptRoundTripOnWrite(false))

	params := types.NewCompletionParams(testCaseFileURI, 5, 11, nil)
	opts := &Options{Client: types.NewGenericClient()}

	result, err := NewPolicy(t.Context(), store, query.NewCache()).Run(t.Context(), c, params, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test.AssertLabels(t, result, []string{"user"})
}

//nolint:paralleltest
func TestPolicyProvider_Example2(t *testing.T) {
	file1 := ast.MustParseModule("package example\n\nfoo := true\n")
	file2 := ast.MustParseModule("package example2\n\nimport data.example\n")

	store := inmem.NewFromObject(map[string]any{
		"workspace": map[string]any{
			"parsed": map[string]any{
				"file:///file1.rego": file1,
				"file:///file2.rego": file2,
			},
			"defined_refs": map[string][]string{
				"file:///file1.rego": {"example.foo"},
				"file:///file2.rego": {},
			},
		},
	})

	fileEdited := "package example2\n\nimport data.example\n\nallow if {\n\tfoo :=\n}\n"

	c := cache.NewCache()
	c.SetFileContents("file:///file2.rego", fileEdited)

	params := types.NewCompletionParams("file:///file2.rego", 5, 11, nil)
	opts := &Options{Client: types.NewGenericClient()}

	result, err := NewPolicy(t.Context(), store, query.NewCache()).Run(t.Context(), c, params, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test.AssertLabels(t, result, []string{"input", "example", "example.foo"})
}
