package semantictokens

import (
	"context"
	"slices"

	"github.com/open-policy-agent/regal/internal/lsp/types"
)

type (
	TokenType     = uint32
	TokenModifier = uint32
)

const (
	Namespace TokenType = iota
	Variable
	Import
	Keyword
)

const (
	ModifierDeclaration TokenModifier = 1 << iota
	ModifierDefinition
	ModifierReference
)

var Legend = types.SemanticTokensLegend{
	TokenTypes: []string{
		Namespace: "namespace",
		Variable:  "variable",
		Import:    "import",
		Keyword:   "keyword",
	},
	TokenModifiers: []string{
		"declaration",
		"definition",
		"reference",
	},
}

type Token struct {
	Line      uint32
	Col       uint32
	Length    uint32
	Type      uint32
	Modifiers uint32
}

// Represents the structured result from the Rego query
type SemanticTokensResult struct {
	PackageTokens []Token `json:"packages"`
	ImportTokens  []Token `json:"imports"`
	Vars          []Token `json:"vars"`
	DebugInfo     any     `json:"debug_info"`
}

func Full(ctx context.Context, result SemanticTokensResult) (*types.SemanticTokens, error) {
	tokens := slices.Concat(
		result.PackageTokens,
		result.Vars,
		result.ImportTokens,
	)

	return encodeTokens(tokens), nil
}

func encodeTokens(tokens []Token) *types.SemanticTokens {
	if len(tokens) == 0 {
		return &types.SemanticTokens{Data: []uint32{}}
	}

	// Sort tokens by position (line first, then column)
	slices.SortFunc(tokens, func(a, b Token) int {
		if a.Line != b.Line {
			return int(a.Line) - int(b.Line)
		}
		return int(a.Col) - int(b.Col)
	})

	data := make([]uint32, 0, len(tokens)*5)

	var prevLine, prevCol uint32

	for _, tok := range tokens {
		deltaLine := tok.Line - prevLine
		deltaCol := tok.Col

		// If on the same line as previous token, column is relative
		if deltaLine == 0 {
			deltaCol = tok.Col - prevCol
		}

		data = append(data,
			deltaLine,
			deltaCol,
			tok.Length,
			tok.Type,
			tok.Modifiers,
		)

		prevLine = tok.Line
		prevCol = tok.Col
	}

	return &types.SemanticTokens{Data: data}
}
