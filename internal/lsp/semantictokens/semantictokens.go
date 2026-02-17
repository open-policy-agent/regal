package semantictokens

import (
	"context"
	"fmt"
	"slices"

	"strconv"
	"strings"

	"github.com/open-policy-agent/regal/internal/lsp/types"
)

const (
	TokenTypePackage  = 0
	TokenTypeVariable = 1
	TokenTypeImport   = 2
)

const (
	ModifierDeclaration = 1 << 0
	ModifierReference   = 1 << 1
)

type Token struct {
	Line      uint
	Col       uint
	Length    uint
	Type      uint
	Modifiers uint
}

// Represents location data from the AST
type ASTLocation struct {
	Location string `json:"location"`
}

// Represents different token categories
type ArgTokenCategory struct {
	Declaration []ASTLocation `json:"declaration,omitempty"`
	Reference   []ASTLocation `json:"reference,omitempty"`
}

// Represents the structured result from the Rego query
type SemanticTokensResult struct {
	ArgTokens     ArgTokenCategory `json:"arg_tokens"`
	PackageTokens []ASTLocation    `json:"package_tokens"`
	ImportTokens  []ASTLocation    `json:"import_tokens"`
}

func Full(ctx context.Context, result SemanticTokensResult) (*types.SemanticTokens, error) {
	tokens := make([]Token, 0)

	for _, pkgToken := range result.PackageTokens[1:] {

		token, err := extractTokens(pkgToken, TokenTypePackage, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create package token: %w", err)
		}
		tokens = append(tokens, token)
	}

	for _, varToken := range result.ArgTokens.Declaration {
		if varToken.Location == "" {
			continue
		}

		token, err := extractTokens(varToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create declaration token: %w", err)
		}
		tokens = append(tokens, token)
	}

	for _, varToken := range result.ArgTokens.Reference {
		if varToken.Location == "" {
			continue
		}

		token, err := extractTokens(varToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create reference token: %w", err)
		}
		tokens = append(tokens, token)
	}

	for _, importToken := range result.ImportTokens {
		if importToken.Location == "" {
			continue
		}

		token, err := extractTokens(importToken, TokenTypeImport, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create import token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return encodeTokens(tokens), nil
}

func extractTokens(astLoc ASTLocation, tokenType uint, modifiers uint) (Token, error) {
	rowStr, rest, _ := strings.Cut(astLoc.Location, ":")
	colStr, _, _ := strings.Cut(rest, ":")

	row, err := strconv.Atoi(rowStr)
	if err != nil {
		return Token{}, fmt.Errorf("failed to parse row: %w", err)
	}

	col, err := strconv.Atoi(colStr)
	if err != nil {
		return Token{}, fmt.Errorf("failed to parse column: %w", err)
	}

	length, err := getTokenLengthFromLocation(astLoc.Location)
	if err != nil {
		return Token{}, fmt.Errorf("failed to get token length: %w", err)
	}

	return Token{
		Line:      uint(row - 1),
		Col:       uint(col - 1),
		Length:    length,
		Type:      tokenType,
		Modifiers: modifiers,
	}, nil
}

func encodeTokens(tokens []Token) *types.SemanticTokens {
	if len(tokens) == 0 {
		return &types.SemanticTokens{Data: []uint{}}
	}

	// Sort tokens by position (line first, then column)
	slices.SortFunc(tokens, func(a, b Token) int {
		if a.Line != b.Line {
			return int(a.Line) - int(b.Line)
		}
		return int(a.Col) - int(b.Col)
	})

	data := make([]uint, 0, len(tokens)*5)

	var prevLine, prevCol uint

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

func getTokenLengthFromLocation(locationStr string) (uint, error) {
	parts := strings.Split(locationStr, ":")

	startCol, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse start column: %w", err)
	}

	endCol, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, fmt.Errorf("failed to parse end column: %w", err)
	}

	return uint(endCol - startCol), nil
}
