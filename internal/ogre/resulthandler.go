package ogre

import "github.com/open-policy-agent/opa/v1/ast"

type (
	// Result carries the value of a single evaluation result along with the evaluator that produced it.
	Result struct {
		Value     ast.Value
		Evaluator *Evaluator
	}
	// ResultHandler defines the interface for handling evaluation results.
	ResultHandler interface {
		// Handle processes a single evaluation [Result].
		Handle(Result) error
	}
	resultHandlerFunc struct {
		f func(Result) error
	}
)

// ResultHandlerFunc creates a [ResultHandler] from a function.
// Note that this is generally more expensive than implementing
// the [ResultHandler] interface directly, so should only be used
// for convenience where performance is less of a concern.
func ResultHandlerFunc(f func(Result) error) ResultHandler {
	return &resultHandlerFunc{f: f}
}

// Handle calls the underlying function to process the evaluation result.
func (h *resultHandlerFunc) Handle(result Result) error {
	return h.f(result)
}
