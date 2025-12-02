package opa

import "github.com/open-policy-agent/opa/v1/ast"

// NOTE(anders):
// Set CanSkipBctx on the built-in functions that we use in Regal, as we need neither cancellation checks nor the
// inter-query cache (the only two uses of the BuiltinContext parameter for these functions). Building the context
// for no good reason hurts performance. Long-term I hope we find a more robust solution, as this feels rather
// brittle. And not just that, but global-only toggles mean we can't decide what to use based on context. While we
// aren't likely to ever care for cancellation when evaluating our own Rego, this could make sense for e.g. the
// "Evaluate" code lens in the language server, where user code is evaluated.
func init() {
	// Unwanted cancellation checks
	ast.Concat.CanSkipBctx = true
	ast.Replace.CanSkipBctx = true
	ast.NumbersRange.CanSkipBctx = true
	ast.RegexReplace.CanSkipBctx = true

	// Unwanted inter query cache checks
	ast.GlobMatch.CanSkipBctx = true
	ast.RegexMatch.CanSkipBctx = true
	ast.RegexFind.CanSkipBctx = true
	ast.RegexSplit.CanSkipBctx = true
}
