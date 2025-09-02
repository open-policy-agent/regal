package test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

type rpcHandler func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error)

func HandlerFor[T any](method string, h handler.Func[T]) rpcHandler {
	return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
		if req.Method != method {
			return nil, fmt.Errorf("unexpected method: %s for handler of: %s", req.Method, method)
		}

		return handler.WithParams(req, h)
	}
}

func SendsToChannel[T any](c chan T) func(T) (any, error) {
	return func(msg T) (any, error) {
		c <- msg

		return struct{}{}, nil
	}
}

func Labels(completions []types.CompletionItem) []string {
	labels := make([]string, len(completions))
	for i, c := range completions {
		labels[i] = c.Label
	}

	return labels
}

func AssertLabels(t *testing.T, result []types.CompletionItem, expected []string) {
	t.Helper()

	labels := Labels(result)
	if !slices.Equal(expected, labels) {
		t.Fatalf("expected %v, got %v", expected, labels)
	}
}
