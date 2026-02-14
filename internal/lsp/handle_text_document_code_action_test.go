package lsp

import (
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/internal/web"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

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
		Code:    "use-assignment-operator",
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
	must.Equal(t, len(expArgs), len(actualArgs), "number of arguments")

	expDecoded := must.Return(encoding.JSONUnmarshalTo[map[string]any]([]byte(expArgs[0].(string))))(t)
	actDecoded := must.Return(encoding.JSONUnmarshalTo[map[string]any]([]byte(actualArgs[0].(string))))(t)

	assert.DeepEqual(t, expDecoded, actDecoded, "decoded command arguments")
}

func assertExpectedCodeAction(t *testing.T, expected, actual types.CodeAction) {
	t.Helper()

	assert.Equal(t, expected.Title, actual.Title, "Title")
	assert.Equal(t, expected.Kind, actual.Kind, "Kind")
	assert.Equal(t, len(expected.Diagnostics), len(actual.Diagnostics), "# Diagnostics")

	if expected.IsPreferred == nil && actual.IsPreferred != nil { //nolint:gocritic
		t.Error("expected IsPreferred to be nil")
	} else if expected.IsPreferred != nil && actual.IsPreferred == nil {
		t.Error("expected IsPreferred to be non-nil")
	} else if expected.IsPreferred != nil && actual.IsPreferred != nil && *expected.IsPreferred != *actual.IsPreferred {
		t.Errorf("expected IsPreferred to be %v, got %v", *expected.IsPreferred, *actual.IsPreferred)
	}

	assert.Equal(t, expected.Command.Command, actual.Command.Command, "Command.Command")
	assert.Equal(t, expected.Command.Title, actual.Command.Title, "Command.Title")
	assert.Equal(t, expected.Command.Tooltip, actual.Command.Tooltip, "Command.Tooltip")

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

	result := must.Return(l.Handle(t.Context(), nil, req))(t)
	actions := must.Be[[]types.CodeAction](t, result)
	must.Equal(t, expectedCount, len(actions), "number of code actions")

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
				Code:    "use-assignment-operator",
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
