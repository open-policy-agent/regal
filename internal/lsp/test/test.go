package test

import (
	"context"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/connection"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
)

func HandlerFor[T any](method string, h handler.Func[T]) connection.HandlerFunc {
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
