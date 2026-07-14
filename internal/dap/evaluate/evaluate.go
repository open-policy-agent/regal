// Package evaluate provides an interface for handling DAP evaluate requests
// and a default implementation that returns an error when no handler is set.
package evaluate

import (
	"context"
	"errors"

	godap "github.com/google/go-dap"

	"github.com/open-policy-agent/opa/v1/debug"
)

var (
	// DefaultHandler is the default implementation of Handler that returns an error
	// to indicate no evaluate handler is set, hinting that the DAP is not connected to a language server.
	DefaultHandler = fallback{}

	errNoEvalHandler = errors.New("no evaluate handler set (DAP not connected to language server?)")
)

type (
	// Request is a type alias for [*godap.EvaluateRequest] to simplify the interface definition.
	Request = *godap.EvaluateRequest
	// Response is a type alias for [*godap.EvaluateResponse] to simplify the interface definition.
	Response = *godap.EvaluateResponse
	// Handler defines a function that handles DAP evaluate requests.
	Handler interface {
		Evaluate(context.Context, debug.Session, Request) (Response, error)
	}

	fallback struct{}
)

// NewResponse creates a new EvaluateResponse with the given body.
func NewResponse(body godap.EvaluateResponseBody) Response {
	return &godap.EvaluateResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{Type: "response"},
			Command:         "evaluate",
			Success:         true,
		},
		Body: body,
	}
}

// NewEmptyResponse creates a new EvaluateResponse with an empty body.
func NewEmptyResponse() Response {
	return NewResponse(godap.EvaluateResponseBody{})
}

// Evaluate implements the Handler interface.
func (fallback) Evaluate(context.Context, debug.Session, Request) (Response, error) {
	return NewEmptyResponse(), errNoEvalHandler
}
