package semantictokens

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/open-policy-agent/regal/internal/lsp/types"
)

type TokenType = uint32

const (
	Namespace TokenType = iota
	Variable
	Import
	Keyword
)

type Token struct {
	Line      uint32
	Col       uint32
	Length    uint32
	Type      uint32
	Modifiers uint32
}

// SemanticTokensResult represents the structured result from the Rego query.
type SemanticTokensResult struct {
	PackageTokens []Token `json:"packages"`
	ImportTokens  []Token `json:"imports"`
	Vars          []Token `json:"vars"`
	DebugInfo     any     `json:"debug_info"`
}

func Full(result SemanticTokensResult) (*types.SemanticTokens, error) {
	tokens := slices.Concat(result.PackageTokens, result.Vars, result.ImportTokens)
	if len(tokens) == 0 {
		return &types.SemanticTokens{Data: []uint32{}}, nil
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

	return &types.SemanticTokens{Data: data}, nil
}

func ResultHandler(_ context.Context, result any) (any, error) {
	if raw, ok := result.(*json.RawMessage); ok {
		var semTokRes SemanticTokensResult
		// this looks like a false positive as the struct fields are tagged
		// "the given struct should be annotated with the `json` tag"
		//nolint: musttag
		if err := json.Unmarshal(*raw, &semTokRes); err != nil {
			return nil, err
		}

		full, err := Full(semTokRes)
		if err != nil {
			return nil, err
		}

		bs, err := json.Marshal(full)
		if err != nil {
			return nil, err
		}

		return new(json.RawMessage(bs)), nil
	}

	return nil, fmt.Errorf("expected *json.RawMessage, got: %T", result)
}
