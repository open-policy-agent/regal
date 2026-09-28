package hints

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestHints(t *testing.T) {
	t.Parallel()

	_, err := parse.Module("test.rego", "package foo\n\nincomplete")

	assert.Equal(t, "rego-parse-error/var-cannot-be-used-for-rule-name", GetForError(err))

	mod := must.Return(parse.Module("test.rego", "package foo\n\nrecursive(x) if recursive(1)"))(t)
	cmp := compile.NewCompilerWithRegalBuiltins()
	cmp.Compile(map[string]*ast.Module{"test.rego": mod})

	assert.Equal(t, "rego-recursion-error/rule-name-is-recursive", GetForError(cmp.Errors))
}
