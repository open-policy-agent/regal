package lsp

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/tester"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestHandleRunTest(t *testing.T) {
	t.Parallel()

	testRegoContents := `package bar_test

test_with_output if {
	print("hello from test")
	true
}
`

	files := map[string]string{
		"bar_test.rego": testRegoContents,
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	testRegoURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tempDir, "bar_test.rego"))

	receivedMessages := make(chan types.FileDiagnostics, defaultBufferedChannelSize)
	clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedMessages))

	_, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)
	openParams := types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{URI: testRegoURI, Text: testRegoContents},
	}

	if err := connClient.Notify(ctx, "textDocument/didOpen", openParams); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	// Wait for diagnostics to be published (ensures file is parsed before running tests)
	// We don't care about the actual diagnostic codes, just that parsing completed
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	select {
	case <-receivedMessages:
		// Diagnostics received, file is parsed
	case <-timeout.C:
		t.Fatal("timeout waiting for diagnostics")
	}

	params := types.RunTestsParams{
		Targets: []types.TestTarget{{Package: "data.bar_test", Name: "test_with_output"}},
	}

	var result []tester.Result
	if err := connClient.Call(ctx, "regal/runTests", params, &result); err != nil {
		t.Fatalf("failed to call regal/runTests: %s", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 test result, got %d", len(result))
	}

	if !result[0].Pass() {
		t.Error("expected test to pass, but it failed")
	}

	if len(result[0].Output) == 0 {
		t.Error("expected output to be captured, but it was empty")
	}
}

func TestHandleRunTestsTargets(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		files map[string]string
		// target URIs are file names, resolved against the temp directory
		params   types.RunTestsParams
		expected []string
	}{
		"root target runs everything": {
			files: map[string]string{
				"a_test.rego": `package a_test

test_one if true
`,
				"c_test.rego": `package c_test

test_five if true
`,
			},
			params:   types.RunTestsParams{Targets: []types.TestTarget{{}}},
			expected: []string{"data.a_test.test_one", "data.c_test.test_five"},
		},
		"file runs only its tests": {
			files: map[string]string{
				"a_test.rego": `package a_test

test_one if true

test_two if true
`,
				"a_other_test.rego": `package a_test

test_three if true
`,
			},
			params:   types.RunTestsParams{Targets: []types.TestTarget{{URI: "a_test.rego"}}},
			expected: []string{"data.a_test.test_one", "data.a_test.test_two"},
		},
		"exclude from a package": {
			files: map[string]string{
				"a_test.rego": `package a_test

test_one if true

test_two if true
`,
				"a_sub_test.rego": `package a_test.sub

test_three if true
`,
			},
			params: types.RunTestsParams{
				Targets: []types.TestTarget{{Package: "data.a_test"}},
				Exclude: []types.TestTarget{{Package: "data.a_test.sub"}, {Package: "data.a_test", Name: "test_two"}},
			},
			expected: []string{"data.a_test.test_one"},
		},
		"excluding everything selected runs nothing": {
			files: map[string]string{
				"c_test.rego": `package c_test

test_five if true
`,
			},
			params: types.RunTestsParams{
				Targets: []types.TestTarget{{Package: "data.c_test"}},
				Exclude: []types.TestTarget{{URI: "c_test.rego"}},
			},
			expected: []string{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutil.TempDirectoryOf(t, tc.files)
			rootURI := uri.FromPath(clients.IdentifierGeneric, tempDir)

			resolve := func(targets []types.TestTarget) []types.TestTarget {
				return outil.Map(targets, func(target types.TestTarget) types.TestTarget {
					if target.URI != "" {
						target.URI = uri.FromRelativePath(clients.IdentifierGeneric, target.URI, rootURI)
					}

					return target
				})
			}

			params := types.RunTestsParams{Targets: resolve(tc.params.Targets), Exclude: resolve(tc.params.Exclude)}
			clientHandler := createPublishDiagnosticsHandler(t, test.DebugLogger(t), createMessageChannels(tc.files))

			_, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

			var results []tester.Result
			if err := connClient.Call(ctx, "regal/runTests", params, &results); err != nil {
				t.Fatalf("failed to call regal/runTests: %s", err)
			}

			got := make([]string, 0, len(results))
			for _, r := range results {
				if !r.Pass() {
					t.Errorf("expected %s.%s to pass", r.Package, r.Name)
				}

				got = append(got, r.Package+"."+r.Name)
			}

			slices.Sort(got)

			if !slices.Equal(got, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestTestPrefixes(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		modules  map[string]string
		targets  []types.TestTarget
		exclude  []types.TestTarget
		expected []string
		wantErr  bool
	}{
		"no targets": {
			wantErr: true,
		},
		"empty target is the root": {
			targets:  []types.TestTarget{{}},
			expected: []string{"data"},
		},
		"package": {
			targets:  []types.TestTarget{{Package: "data.a_test"}},
			expected: []string{"data.a_test"},
		},
		"package and name": {
			targets:  []types.TestTarget{{Package: "data.a_test", Name: "test_one"}},
			expected: []string{"data.a_test.test_one"},
		},
		"nested name": {
			targets:  []types.TestTarget{{Package: "data.a_test.sub", Name: "nested.test_three"}},
			expected: []string{"data.a_test.sub.nested.test_three"},
		},
		"quoted package": {
			targets:  []types.TestTarget{{Package: `data.e["my-pkg"]`, Name: "test_four"}},
			expected: []string{`data.e["my-pkg"].test_four`},
		},
		"name without package": {
			targets: []types.TestTarget{{Name: "test_one"}},
			wantErr: true,
		},
		"package without a data root": {
			targets: []types.TestTarget{{Package: "a_test"}},
			wantErr: true,
		},
		"unparseable package": {
			targets: []types.TestTarget{{Package: "data.e.my-pkg"}},
			wantErr: true,
		},
		"file lists its tests, including skipped ones, but not other rules": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

helper := 1

test_one if true

todo_test_skipped if true

test_two if true`,
			},
			targets:  []types.TestTarget{{URI: "file:///a_test.rego"}},
			expected: []string{"data.a_test.test_one", "data.a_test.test_two", "data.a_test.todo_test_skipped"},
		},
		"file with a nested test name": {
			modules: map[string]string{
				"file:///a_sub_test.rego": `package a_test.sub

nested.test_three if true`,
			},
			targets:  []types.TestTarget{{URI: "file:///a_sub_test.rego"}},
			expected: []string{"data.a_test.sub.nested.test_three"},
		},
		"file with a non-ground test rule ref uses its ground prefix": {
			modules: map[string]string{
				"file:///f_test.rego": `package f_test

test_gen[x] if x := 1

testing if true

my_test_x if true`,
			},
			targets:  []types.TestTarget{{URI: "file:///f_test.rego"}},
			expected: []string{"data.f_test.test_gen"},
		},
		"unknown file": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

test_one if true`,
			},
			targets: []types.TestTarget{{URI: "file:///missing_test.rego"}},
			wantErr: true,
		},
		"multiple targets are combined": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

test_one if true

test_two if true`,
				"file:///a_sub_test.rego": `package a_test.sub

nested.test_three if true`,
			},
			targets:  []types.TestTarget{{URI: "file:///a_sub_test.rego"}, {Package: "data.a_test", Name: "test_one"}},
			expected: []string{"data.a_test.sub.nested.test_three", "data.a_test.test_one"},
		},
		"exclude from the root expands everything, and keeps sibling packages": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

test_one if true`,
				"file:///a_testing_test.rego": `package a_testing

test_five if true`,
				"file:///e_test.rego": `package e["my-pkg"]

test_four if true`,
			},
			targets:  []types.TestTarget{{}},
			exclude:  []types.TestTarget{{Package: "data.a_test"}},
			expected: []string{"data.a_testing.test_five", `data.e["my-pkg"].test_four`},
		},
		"exclude one test from a package": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

test_one if true

test_two if true`,
				"file:///a_sub_test.rego": `package a_test.sub

nested.test_three if true`,
			},
			targets:  []types.TestTarget{{Package: "data.a_test"}},
			exclude:  []types.TestTarget{{Package: "data.a_test", Name: "test_two"}},
			expected: []string{"data.a_test.sub.nested.test_three", "data.a_test.test_one"},
		},
		"exclude a file": {
			modules: map[string]string{
				"file:///a_test.rego": `package a_test

test_one if true`,
				"file:///a_sub_test.rego": `package a_test.sub

nested.test_three if true`,
			},
			targets:  []types.TestTarget{{Package: "data.a_test"}},
			exclude:  []types.TestTarget{{URI: "file:///a_test.rego"}},
			expected: []string{"data.a_test.sub.nested.test_three"},
		},
		"exclude everything selected": {
			modules: map[string]string{"file:///e_test.rego": `package e["my-pkg"]

test_four if true`},
			targets: []types.TestTarget{{Package: "data.e"}},
			exclude: []types.TestTarget{{Package: `data.e["my-pkg"]`}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			modules := make(map[string]*ast.Module, len(tc.modules))
			for uri, src := range tc.modules {
				modules[uri] = ast.MustParseModule(src)
			}

			refs, err := testPrefixes(modules, tc.targets, tc.exclude)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("unexpected error: %s", err)
				}

				return
			}

			if tc.wantErr {
				t.Fatal("expected an error")
			}

			got := make([]string, 0, len(refs))
			for _, ref := range refs {
				got = append(got, ref.String())
			}

			slices.Sort(got)

			if !slices.Equal(got, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
