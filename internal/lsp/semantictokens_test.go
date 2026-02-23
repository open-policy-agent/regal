package lsp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/web"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestFull(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		policy         string
		expectedTokens []semanticTokenInstance
	}{
		"package only": {
			policy: `package regal.woo`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
			},
		},
		"variable declarations": {
			policy: `package regal.woo

test_function(param1, param2) := result if {
      true
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 8, Length: 6, Type: 1, Modifier: 1},
			},
		},
		"variable references": {
			policy: `package regal.woo
			
test_function(param1) := result if {
      calc3 := 1
      calc3 == param1
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 2, DeltaCol: 15, Length: 6, Type: 1, Modifier: 2},
			},
		},
		"imports": {
			policy: `package regal.woo

import data.regal.ast
			
test_function(param1) := result if {
      calc3 := 1
      calc3 == param1
	  ast.is_constant
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 18, Length: 3, Type: 2, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 2, DeltaCol: 15, Length: 6, Type: 1, Modifier: 2},
			},
		},
		"full policy with package, declarations and references": {
			policy: `package regal.woo
			
test_function(param1) := result if {
	  calc1 := param1 * 2
      calc2 := param2 + 10
      result := calc1 + calc2
	  
      calc3 := 1
      calc3 == param1
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 12, Length: 6, Type: 1, Modifier: 2},
				{DeltaLine: 5, DeltaCol: 15, Length: 6, Type: 1, Modifier: 2},
			},
		},
		"comprehensions": {
			policy: `package regal.woo

array_comprehensions := [x |  
    some i, x in [1, 2, 3]    
    i == 2                    
]

set_comprehensions := {x |    
    some i, x in [1, 2, 3]    
    i == 2                    
}

object_comprehensions := {k: v |  
    some k, v in [1, 2, 3]       
    v == 2                        
}`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 25, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 1, DeltaCol: 9, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 3, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 4, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 3, DeltaCol: 23, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 1, DeltaCol: 9, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 3, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 4, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 3, DeltaCol: 26, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 0, DeltaCol: 3, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 1, DeltaCol: 9, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 3, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 4, Length: 1, Type: 1, Modifier: 2},
			},
		},
		"every constructs": {
			policy: `package regal.woo

every_two_vars_construct if {
    every k, v in input.object {  
        is_string(k)             
        v > 0                    
    }
}

every_one_var_construct if {
    every k in input.object {  
        is_string(k)                                
    }
}`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 3, DeltaCol: 10, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 3, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 18, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 1, DeltaCol: 8, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 5, DeltaCol: 10, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 18, Length: 1, Type: 1, Modifier: 2},
			},
		},
		"some constructs": {
			policy: `package regal.woo

some_two_vars_construct if {
    some i, item in input.array   
    i < 10                        
    item > 0                        
}

some_one_var_construct if {
    some i in input.array   
    i < 10                                              
}`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 3, DeltaCol: 9, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 3, Length: 4, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 4, Length: 1, Type: 1, Modifier: 2},
				{DeltaLine: 1, DeltaCol: 4, Length: 4, Type: 1, Modifier: 2},
				{DeltaLine: 4, DeltaCol: 9, Length: 1, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 4, Length: 1, Type: 1, Modifier: 2},
			},
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			l, params := setupLanguageServerWithPolicy(t, tc.policy)

			result := invokeSemanticTokensHandler(t, l, params)

			actualTokens := uintsToTestTokens(result.Data)

			t.Logf("Actual tokens:\n%+v", actualTokens)
			t.Logf("Expected tokens:\n%+v", tc.expectedTokens)

			if diff := cmp.Diff(actualTokens, tc.expectedTokens); diff != "" {
				t.Errorf("unexpected token data (-got +want):\n%s", diff)
			}
		})
	}
}

// semanticTokenInstance adds structure to uint data stream in the SemanticToken
// return type making it more readable for error messages and comparisons in tests
type semanticTokenInstance struct {
	DeltaLine uint
	DeltaCol  uint
	Length    uint
	Type      uint
	Modifier  uint
}

func uintsToTestTokens(data []uint) []semanticTokenInstance {
	if len(data)%5 != 0 {
		panic("invalid token data length, must be multiple of 5")
	}

	tokens := make([]semanticTokenInstance, 0, len(data)/5)
	for i := 0; i < len(data); i += 5 {
		tokens = append(tokens, semanticTokenInstance{
			DeltaLine: data[i],
			DeltaCol:  data[i+1],
			Length:    data[i+2],
			Type:      data[i+3],
			Modifier:  data[i+4],
		})
	}

	return tokens
}

// generateLargePolicy creates a policy with the specified number of functions for benchmarking
func generateLargePolicy(numFunctions int) string {
	if numFunctions <= 0 {
		return "package regal.woo\n"
	}

	var policy strings.Builder
	policy.WriteString("package regal.woo\n\n")

	for i := range numFunctions {
		fmt.Fprintf(&policy, `test_function_%d(param1, param2) := result if {
	calc1 := param1 * %d
	calc2 := param2 + %d
	result := calc1 + calc2
}

`, i, i+1, i+10)
	}

	return policy.String()
}

// setupLanguageServerWithPolicy sets up a language server for testing/benchmarking with the given policy
func setupLanguageServerWithPolicy(tb testing.TB, policy string) (*LanguageServer, types.SemanticTokensParams) {
	tb.Helper()

	webServer := web.NewServer(log.NewLogger(log.LevelDebug, tb.Output()))
	webServer.SetBaseURL("http://foo.bar")

	l := NewLanguageServer(tb.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, tb.Output())})

	l.workspaceRootURI = "file:///foo"
	l.client = types.Client{Identifier: clients.IdentifierVSCode}
	l.webServer = webServer
	l.loadedConfig = &config.Config{}

	fileURI := "file:///foo/test.rego"
	l.cache.SetFileContents(fileURI, policy)

	module := ast.MustParseModule(policy)
	l.cache.SetModule(fileURI, module)

	err := PutFileMod(tb.Context(), l.regoStore, fileURI, module)
	if err != nil {
		tb.Fatalf("failed to store module: %v", err)
	}

	params := types.SemanticTokensParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: fileURI,
		},
	}

	return l, params
}

// Benchmark function that runs the language server request for a policy containing x amount of rules
func BenchmarkFullCustomRuleCount(b *testing.B) {
	policy := generateLargePolicy(100)
	l, params := setupLanguageServerWithPolicy(b, policy)

	b.ResetTimer()

	for b.Loop() {
		result := invokeSemanticTokensHandler(b, l, params)
		_ = result
	}
}

func invokeSemanticTokensHandler(
	tb testing.TB,
	l *LanguageServer,
	params types.SemanticTokensParams,
) *types.SemanticTokens {
	tb.Helper()

	req := &jsonrpc2.Request{
		Method: "textDocument/semanticTokens/full",
		Params: testutil.ToJSONRawMessage(tb, params),
	}

	result, err := l.Handle(tb.Context(), nil, req)
	if err != nil {
		tb.Errorf("Unexpected error: %v", err)
	}

	tokens, ok := result.(*types.SemanticTokens)
	if !ok {
		tb.Errorf("Expected result to be of type *types.SemanticTokens, got %T", result)
	}

	return tokens
}
