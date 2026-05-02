package funsafe_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/funsafe"
)

func TestFunsafeObject(t *testing.T) {
	t.Parallel()

	obj := ast.NewObject(
		ast.Item(ast.InternedTerm("foo"), ast.InternedTerm("bar")),
		ast.Item(ast.InternedTerm("baz"), ast.InternedTerm("qux")),
	)

	if elm := funsafe.ObjectElems(obj); len(elm) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elm))
	}
}
