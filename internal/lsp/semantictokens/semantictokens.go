package semantictokens

import (
	"context"
	"fmt"

	"sort"
	"strconv"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

const (
	TokenTypePackage  = 0
	TokenTypeVariable = 1
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

func Full(module *ast.Module) (*types.SemanticTokens, error) {
	tokens := make([]Token, 0)

	packageTokens, err := extractPackageTokens(module)
	if err != nil {
		return nil, fmt.Errorf("failed to extract package tokens: %w", err)
	}
	tokens = append(tokens, packageTokens...)

	variableTokens, err := extractVariableTokens(module)
	if err != nil {
		return nil, fmt.Errorf("failed to extract variable tokens: %w", err)
	}
	tokens = append(tokens, variableTokens...)

	return encodeTokens(tokens), nil
}

func extractPackageTokens(module *ast.Module) ([]Token, error) {
	var tokens []Token

	if module.Package != nil && module.Package.Path != nil {
		for _, term := range module.Package.Path {
			packageString := term.Value.String()
			if packageString == "data" {
				continue
			}

			trimmedValue := strings.Trim(packageString, `"`)
			length := uint(len(trimmedValue))

			tokens = append(tokens, Token{
				Line:      uint(term.Location.Row - 1),
				Col:       uint(term.Location.Col - 1),
				Length:    length,
				Type:      TokenTypePackage,
				Modifiers: 0,
			})
		}
	}

	return tokens, nil
}

func extractVariableTokens(module *ast.Module) ([]Token, error) {
	ctx := context.Background()
	var tokens []Token

	roastInput, err := transform.ToAST("policy.rego", "", module, true)
	if err != nil {
		return nil, fmt.Errorf("failed to transform to roast format: %w", err)
	}

	query := ast.MustParseBody(`arg_tokens = data.regal.lsp.semantictokens.arg_tokens`)

	resultHandler := func(result ast.Value) error {
		resultObj := result.(ast.Object)

		resultObj.Foreach(func(varTerm, typeTerm *ast.Term) {
			varObj := varTerm.Value.(ast.Object)
			typeStr := string(typeTerm.Value.(ast.String))

			locationStr := string(varObj.Get(ast.StringTerm("location")).Value.(ast.String))
			varName := string(varObj.Get(ast.StringTerm("value")).Value.(ast.String))

			trimmedVarName := strings.Trim(varName, `"' `)

			parts := strings.Split(locationStr, ":")
			row, _ := strconv.Atoi(parts[0])
			col, _ := strconv.Atoi(parts[1])

			modifier := ModifierReference
			if typeStr == "declaration" {
				modifier = ModifierDeclaration
			}

			token := Token{
				Line:      uint(row - 1),
				Col:       uint(col - 1),
				Length:    uint(len(trimmedVarName)),
				Type:      TokenTypeVariable,
				Modifiers: uint(modifier),
			}
			tokens = append(tokens, token)
		})

		return nil
	}

	q, err := ogre.New(query).Prepare(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run ogre query: %w", err)
	}

	err = q.Evaluator().
		WithInput(roastInput).
		WithResultHandler(resultHandler).
		Eval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate ogre query: %w", err)
	}

	return tokens, nil
}

func encodeTokens(tokens []Token) *types.SemanticTokens {
	if len(tokens) == 0 {
		return &types.SemanticTokens{Data: []uint{}}
	}

	// Sort tokens by position (line first, then column)
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Line != tokens[j].Line {
			return tokens[i].Line < tokens[j].Line
		}
		return tokens[i].Col < tokens[j].Col
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
