package handler

import (
	"context"
	"encoding/json/v2"

	"github.com/sourcegraph/jsonrpc2"
)

type Func[T any] func(T) (any, error)

type ContextFunc[T any] func(context.Context, T) (any, error)

var ErrInvalidParams = &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}

func Decode[T any](req *jsonrpc2.Request, params *T) error {
	if req.Params == nil {
		return ErrInvalidParams
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
	}

	return nil
}

func WithParams[T any](req *jsonrpc2.Request, h Func[T]) (any, error) {
	var params T
	if err := Decode(req, &params); err != nil {
		return nil, err
	}

	return h(params)
}

func WithContextAndParams[T any](ctx context.Context, req *jsonrpc2.Request, h ContextFunc[T]) (any, error) {
	var params T
	if err := Decode(req, &params); err != nil {
		return nil, err
	}

	return h(ctx, params)
}
