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

// Represents the vars section containing different token categories
type VarsSection struct {
	ArgTokens           ArgTokenCategory `json:"function_args"`
	ComprehensionTokens ArgTokenCategory `json:"comprehensions"`
	EveryTokens         ArgTokenCategory `json:"every_expr"`
	SomeTokens          ArgTokenCategory `json:"some_expr"`
}

// Represents the structured result from the Rego query
type SemanticTokensResult struct {
	PackageTokens []ASTLocation `json:"packages"`
	ImportTokens  []ASTLocation `json:"imports"`
	Vars          VarsSection   `json:"vars"`
	DebugInfo     interface{}   `json:"debug_info"`
}

func Full(ctx context.Context, result SemanticTokensResult) (*types.SemanticTokens, error) {
	tokens := make([]Token, 0)

	for _, pkgToken := range result.PackageTokens {
		token, err := extractTokens(pkgToken, TokenTypePackage, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create package token: %w", err)
		}
		tokens = append(tokens, token)
	}

	for _, varToken := range result.Vars.ArgTokens.Declaration {
		if varToken.Location == "" {
			continue
		}

		token, err := extractTokens(varToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create declaration token: %w", err)
		}
		tokens = append(tokens, token)
	}

	for _, varToken := range result.Vars.ArgTokens.Reference {
		if varToken.Location == "" {
			continue
		}

		token, err := extractTokens(varToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create reference token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process comprehension variable declarations
	for _, compToken := range result.Vars.ComprehensionTokens.Declaration {
		if compToken.Location == "" {
			continue
		}

		token, err := extractTokens(compToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create comprehension declaration token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process comprehension variable references
	for _, compToken := range result.Vars.ComprehensionTokens.Reference {
		if compToken.Location == "" {
			continue
		}

		token, err := extractTokens(compToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create comprehension reference token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process every variable declarations
	for _, constructToken := range result.Vars.EveryTokens.Declaration {
		if constructToken.Location == "" {
			continue
		}

		token, err := extractTokens(constructToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create construct declaration token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process every variable references
	for _, constructToken := range result.Vars.EveryTokens.Reference {
		if constructToken.Location == "" {
			continue
		}

		token, err := extractTokens(constructToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create construct reference token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process every variable declarations
	for _, constructToken := range result.Vars.SomeTokens.Declaration {
		if constructToken.Location == "" {
			continue
		}

		token, err := extractTokens(constructToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create construct declaration token: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Process every variable references
	for _, constructToken := range result.Vars.SomeTokens.Reference {
		if constructToken.Location == "" {
			continue
		}

		token, err := extractTokens(constructToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create construct reference token: %w", err)
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
