// Package exp provides experimental features for Regal.
package exp

import (
	"github.com/open-policy-agent/opa/v1/rego"
)

var (
	ExternalCancelNoOp = rego.EvalExternalCancel(topDownCancelNoOp)
	topDownCancelNoOp  = &topdownCancelNoOp{}
)

// topDownCancelNoOp is a no-op implementation of the rego.TopDownCancel interface.
// When no external cancel function is provided to evaluation, OPA instantiates one
// itself (per evaluation, or file linted). This cost is negligible. More interesting is
// the fact that OPA also starts a goroutine per cancel function it creates, launching
// a "waitForDone" function that blocks (inside its goroutine) until evaluation completes.
// While this comes with some overhead, it too is rather insignificant. It does however mean
// that the type of multi-threaded evaluations we typically in the linter will have pprof
// report 99% of all "blocked" time here, making it much harder to identify blocking we do
// ourselves / care about.
//
// This no-op implementation is not meant to be permanent, but one we use while thinking
// more about what we want to do here — how this plays with our own passing of context,
// whether we should expose something around this in Regal's own API, and so on.
type topdownCancelNoOp struct{}

func (*topdownCancelNoOp) Cancel()         {}
func (*topdownCancelNoOp) Cancelled() bool { return false }
