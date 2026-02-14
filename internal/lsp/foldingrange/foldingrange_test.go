package foldingrange_test

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/foldingrange"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestTokenFoldingRanges(t *testing.T) {
	t.Parallel()

	policy := `package p

rule if {
	arr := [
		1,
		2,
		3,
	]
	par := (
		1 +
		2 -
		3
	)
}`

	foldingRanges := foldingrange.TokenRanges(policy)
	must.Equal(t, 3, len(foldingRanges), "number of folding ranges")

	arr := foldingRanges[0]
	assert.Equal(t, 3, arr.StartLine, "start line")
	assert.DereferenceEqual(t, 9, arr.StartCharacter, "start character")
	assert.Equal(t, 6, arr.EndLine, "end line")

	parens := foldingRanges[1]
	assert.Equal(t, 8, parens.StartLine, "start line")
	assert.DereferenceEqual(t, 9, parens.StartCharacter, "start character")
	assert.Equal(t, 11, parens.EndLine, "end line")

	rule := foldingRanges[2]
	assert.Equal(t, 2, rule.StartLine, "start line")
	assert.DereferenceEqual(t, 9, rule.StartCharacter, "start character")
	assert.Equal(t, 12, rule.EndLine, "end line")
}

func TestTokenInvalidFoldingRanges(t *testing.T) {
	t.Parallel()

	policy := `package p

arr := ]]`

	must.Equal(t, 0, len(foldingrange.TokenRanges(policy)), "number of folding ranges")
}
