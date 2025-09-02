package foldingrange_test

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/foldingrange"
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

	if len(foldingRanges) != 3 {
		t.Fatalf("Expected 3 folding ranges, got %d", len(foldingRanges))
	}

	arr := foldingRanges[0]

	if arr.StartLine != 3 || *arr.StartCharacter != 9 {
		t.Errorf("Expected start line 3 and start character 9, got %d and %d", arr.StartLine, *arr.StartCharacter)
	}

	if arr.EndLine != 6 {
		t.Errorf("Expected end line 6, got %d", arr.EndLine)
	}

	parens := foldingRanges[1]

	if parens.StartLine != 8 || *parens.StartCharacter != 9 {
		t.Errorf("Expected start line 8 and start character 9, got %d and %d", parens.StartLine, *parens.StartCharacter)
	}

	if parens.EndLine != 11 {
		t.Errorf("Expected end line 11, got %d", parens.EndLine)
	}

	rule := foldingRanges[2]

	if rule.StartLine != 2 || *rule.StartCharacter != 9 {
		t.Errorf("Expected start line 2 and start character 9, got %d and %d", rule.StartLine, *rule.StartCharacter)
	}

	if rule.EndLine != 12 {
		t.Errorf("Expected end line 12, got %d", rule.EndLine)
	}
}

func TestTokenInvalidFoldingRanges(t *testing.T) {
	t.Parallel()

	policy := `package p

arr := ]]`

	foldingRanges := foldingrange.TokenRanges(policy)

	if len(foldingRanges) != 0 {
		t.Fatalf("Expected no folding ranges, got %d", len(foldingRanges))
	}
}
