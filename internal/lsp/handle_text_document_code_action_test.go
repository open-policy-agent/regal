package lsp

import (
	"reflect"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/internal/web"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

const ruleNameUseAssignmentOperator = "use-assignment-operator"

func TestHandleTextDocumentCodeAction(t *testing.T) {
	t.Parallel()

	webServer := &web.Server{}
	webServer.SetBaseURL("http://foo.bar")

	l := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	l.workspaceRootURI = "file:///"
	l.client = types.NewGenericClient()
	l.webServer = webServer
	l.loadedConfig = &config.Config{}

	diag := types.Diagnostic{
		Code:    ruleNameUseAssignmentOperator,
		Message: "foobar",
		Range:   types.RangeBetween(2, 4, 2, 10),
		Source:  new("regal/style"),
	}

	params := types.CodeActionParams{
		TextDocument: types.TextDocumentIdentifier{URI: "file:///example.rego"},
		Context:      types.CodeActionContext{Diagnostics: []types.Diagnostic{diag}},
		Range:        types.RangeBetween(2, 4, 2, 10),
	}

	expectedAction := types.CodeAction{
		Title:       "Replace = with := in assignment",
		Kind:        "quickfix",
		Diagnostics: params.Context.Diagnostics,
		IsPreferred: new(true),
		Command: types.Command{
			Title:   "Replace = with := in assignment",
			Command: "regal.fix.use-assignment-operator",
			Tooltip: "Replace = with := in assignment",
			Arguments: toAnySlicePtr(string(util.Must(encoding.JSON().Marshal(types.CommandArgs{
				Target:     params.TextDocument.URI,
				Diagnostic: &diag,
			})))),
		},
	}

	actualAction := invokeCodeActionHandler(t, l, params, 3)

	assertExpectedCodeAction(t, expectedAction, actualAction)

	expArgs, actualArgs := *expectedAction.Command.Arguments, *actualAction.Command.Arguments
	if exp, got := len(expArgs), len(actualArgs); exp != got {
		t.Fatalf("expected %d arguments, got %d", exp, got)
	}

	expDecoded := testutil.Must(encoding.JSONUnmarshalTo[map[string]any]([]byte(expArgs[0].(string))))(t)
	actDecoded := testutil.Must(encoding.JSONUnmarshalTo[map[string]any]([]byte(actualArgs[0].(string))))(t)

	if !reflect.DeepEqual(expDecoded, actDecoded) {
		t.Errorf("expected Command.Arguments to be %v, got %v", expDecoded, actDecoded)
	}
}

func assertExpectedCodeAction(t *testing.T, expected, actual types.CodeAction) {
	t.Helper()

	if expected.Title != actual.Title {
		t.Errorf("expected Title %q, got %q", expected.Title, actual.Title)
	}

	if expected.Kind != actual.Kind {
		t.Errorf("expected Kind %q, got %q", expected.Kind, actual.Kind)
	}

	if len(expected.Diagnostics) != len(actual.Diagnostics) {
		t.Errorf("expected %d diagnostics, got %d", len(expected.Diagnostics), len(actual.Diagnostics))
	}

	if expected.IsPreferred == nil && actual.IsPreferred != nil { //nolint:gocritic
		t.Error("expected IsPreferred to be nil")
	} else if expected.IsPreferred != nil && actual.IsPreferred == nil {
		t.Error("expected IsPreferred to be non-nil")
	} else if expected.IsPreferred != nil && actual.IsPreferred != nil && *expected.IsPreferred != *actual.IsPreferred {
		t.Errorf("expected IsPreferred to be %v, got %v", *expected.IsPreferred, *actual.IsPreferred)
	}

	if expected.Command.Command != actual.Command.Command {
		t.Errorf("expected Command %q, got %q", expected.Command.Command, actual.Command.Command)
	}

	if expected.Command.Title != actual.Command.Title {
		t.Errorf("expected Command.Title %q, got %q", expected.Command.Title, actual.Command.Title)
	}

	if expected.Command.Tooltip != actual.Command.Tooltip {
		t.Errorf("expected Command.Tooltip %q, got %q", expected.Command.Tooltip, actual.Command.Tooltip)
	}

	// Just check nilness here, and leave the actual content to the test.
	if expected.Command.Arguments == nil && actual.Command.Arguments != nil {
		t.Error("expected Command.Arguments to be nil")
	} else if expected.Command.Arguments != nil && actual.Command.Arguments == nil {
		t.Error("expected Command.Arguments to be non-nil")
	}
}

func invokeCodeActionHandler(
	t *testing.T,
	l *LanguageServer,
	params types.CodeActionParams,
	expectedCount int,
) types.CodeAction {
	t.Helper()

	req := &jsonrpc2.Request{Method: "textDocument/codeAction", Params: testutil.ToJSONRawMessage(t, params)}

	result, err := l.Handle(t.Context(), nil, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	actions, ok := result.([]types.CodeAction)
	if !ok {
		t.Errorf("Expected result to be of type []types.CodeAction, got %T", result)
	}

	if exp, got := expectedCount, len(actions); exp != got {
		t.Fatalf("Expected %d action(s), got %d", exp, got)
	}

	return actions[0]
}

// 63243 ns/op	   59576 B/op	    1110 allocs/op - the OPA JSON roundtrip method
// 42402 ns/op	   37822 B/op	     738 allocs/op - build input Value by hand
// 45049 ns/op	   39731 B/op	     790 allocs/op - build input Value using reflection
// 44024 ns/op	   38040 B/op	     749 allocs/op - build input Value using reflection + interning
// ...
// "real world" usage shows a number somewhere between 0.1 - 0.5 ms
// of which most of the cost is in JSON marshaling and unmarshaling.
func BenchmarkHandleTextDocumentCodeAction(b *testing.B) {
	l := NewLanguageServer(b.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelMessage, b.Output())})

	l.workspaceRootURI = "file:///"
	l.client = types.NewGenericClient()
	l.webServer = &web.Server{}
	l.loadedConfig = &config.Config{}

	params := types.CodeActionParams{
		TextDocument: types.TextDocumentIdentifier{URI: "file:///example.rego"},
		Context: types.CodeActionContext{
			Diagnostics: []types.Diagnostic{{
				Code:    ruleNameUseAssignmentOperator,
				Message: "foobar",
				Range:   types.RangeBetween(2, 4, 2, 10),
				Source:  new("regal/style"),
			}},
		},
	}

	for b.Loop() {
		res, err := l.Handle(b.Context(), nil, &jsonrpc2.Request{
			Method: "textDocument/codeAction",
			Params: testutil.ToJSONRawMessage(b, params),
		})
		if err != nil {
			b.Fatal(err)
		}

		if len(res.([]types.CodeAction)) != 3 {
			b.Fatalf("expected 3 code actions, got %d", len(res.([]types.CodeAction)))
		}
	}
}

func toAnySlicePtr(a ...string) *[]any {
	b := make([]any, len(a))
	for i := range a {
		b[i] = a[i]
	}

	return &b
}
