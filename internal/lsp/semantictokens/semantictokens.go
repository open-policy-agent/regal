package semantictokens

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
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
	Line      uint32 `json:"line"`
	Col       uint32 `json:"col"`
	Length    uint32 `json:"length"`
	Type      uint32 `json:"type"`
	Modifiers uint32 `json:"modifiers"`
}

// SemanticTokensResult represents the structured result from the Rego query.
type SemanticTokensResult struct {
	PackageTokens []Token `json:"packages"`
	ImportTokens  []Token `json:"imports"`
	Vars          []Token `json:"vars"`
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
		deltaLine, deltaCol := tok.Line-prevLine, tok.Col

		// If on the same line as previous token, column is relative
		if deltaLine == 0 {
			deltaCol = tok.Col - prevCol
		}

		data = append(data, deltaLine, deltaCol, tok.Length, tok.Type, tok.Modifiers)

		prevLine, prevCol = tok.Line, tok.Col
	}

	return &types.SemanticTokens{Data: data}, nil
}

func ResultHandler(_ context.Context, result any) (any, error) {
	if result == nil {
		return nil, nil //nolint:nilnil
	}

	if raw, ok := result.(*jsontext.Value); ok {
		var semTokRes SemanticTokensResult
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

		return new(jsontext.Value(bs)), nil
	}

	return nil, fmt.Errorf("expected *jsontext.Value, got: %T", result)
}
