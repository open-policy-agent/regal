package semantictokens

import (
	"context"
	"encoding/json"
	"slices"

	"strconv"
	"strings"

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

// Represents location data from the AST
type ASTLocation struct {
	Location LocationInfo `json:"location"`
}

type LocationInfo struct {
	Line        uint32
	StartColumn uint32
	EndColumn   uint32
	Length      uint32
}

func (loc *ASTLocation) UnmarshalJSON(data []byte) error {
	var location struct {
		Location string
	}
	if err := json.Unmarshal(data, &location); err != nil {
		return err
	}

	parts := strings.Split(location.Location, ":")

	row, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}

	startcol, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	endcol, err := strconv.Atoi(parts[3])
	if err != nil {
		return err
	}

	loc.Location = LocationInfo{
		Line:        uint32(row),
		StartColumn: uint32(startcol),
		EndColumn:   uint32(endcol),
		Length:      uint32(endcol - startcol),
	}

	return nil
}

// Represents the structured result from the Rego query
type SemanticTokensResult struct {
	PackageTokens []Token `json:"packages"`
	ImportTokens  []Token `json:"imports"`
	Vars          []Token `json:"vars"`
	DebugInfo     any     `json:"debug_info"`
}

func Full(ctx context.Context, result SemanticTokensResult) (*types.SemanticTokens, error) {
	importTokens, err := processImportTokens(result.ImportTokens)
	if err != nil {
		return nil, err
	}

	tokens := slices.Concat(
		result.PackageTokens,
		result.Vars,
		importTokens,
	)

	return encodeTokens(tokens), nil
}

func extractTokens(astLoc ASTLocation, tokenType uint32, modifiers uint32) (Token, error) {
	return Token{
		Line:      astLoc.Location.Line - 1,
		Col:       astLoc.Location.StartColumn - 1,
		Length:    astLoc.Location.Length,
		Type:      tokenType,
		Modifiers: modifiers,
	}, nil
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
