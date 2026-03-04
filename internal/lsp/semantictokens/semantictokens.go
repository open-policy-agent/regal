package semantictokens

import (
	"context"
	"encoding/json"
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
	Location LocationInfo `json:"location"`
}

type LocationInfo struct {
	Line        uint
	StartColumn uint
	EndColumn   uint
	Length      uint
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
		Line:        uint(row),
		StartColumn: uint(startcol),
		EndColumn:   uint(endcol),
		Length:      uint(endcol - startcol),
	}

	return nil
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

	packageTokens, err := processPackageTokens(result.PackageTokens)
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, packageTokens...)

	varTokens, err := processVariableTokens(result.Vars)
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, varTokens...)

	importTokens, err := processImportTokens(result.ImportTokens)
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, importTokens...)

	return encodeTokens(tokens), nil
}

func processPackageTokens(packageTokens []ASTLocation) ([]Token, error) {
	tokens := make([]Token, 0)

	for _, pkgToken := range packageTokens {
		token, err := extractTokens(pkgToken, TokenTypePackage, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create package token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func processVariableTokens(vars VarsSection) ([]Token, error) {
	tokens := make([]Token, 0)

	argTokens, err := processTokenCategory(vars.ArgTokens, "function argument")
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, argTokens...)

	compTokens, err := processTokenCategory(vars.ComprehensionTokens, "comprehension")
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, compTokens...)

	everyTokens, err := processTokenCategory(vars.EveryTokens, "every construct")
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, everyTokens...)

	someTokens, err := processTokenCategory(vars.SomeTokens, "some construct")
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, someTokens...)

	return tokens, nil
}

func processTokenCategory(category ArgTokenCategory, categoryName string) ([]Token, error) {
	tokens := make([]Token, 0)

	for _, declToken := range category.Declaration {
		token, err := extractTokens(declToken, TokenTypeVariable, ModifierDeclaration)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s declaration token: %w", categoryName, err)
		}
		tokens = append(tokens, token)
	}

	for _, refToken := range category.Reference {
		token, err := extractTokens(refToken, TokenTypeVariable, ModifierReference)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s reference token: %w", categoryName, err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func processImportTokens(importTokens []ASTLocation) ([]Token, error) {
	tokens := make([]Token, 0)

	for _, importToken := range importTokens {
		token, err := extractTokens(importToken, TokenTypeImport, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create import token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func extractTokens(astLoc ASTLocation, tokenType uint, modifiers uint) (Token, error) {
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
