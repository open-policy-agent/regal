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

	if module.Package != nil && module.Package.Path != nil {
		for _, term := range module.Package.Path {
			if term.Value.String() == "data" {
				continue
			}

			trimmedValue := strings.Trim(term.Value.String(), `"`)
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

	ctx := context.Background()

	roastInput, err := transform.ToAST("semantictokens.rego", "", module, true)
	if err != nil {
		return nil, fmt.Errorf("Failed to transform to roast format: %w", err)
	}

	queryModule, err := ast.ParseModule("query.rego", `
		package semantictokens

		import data.regal.ast

		# Function argument declarations
		arg_tokens[var] := "declaration" if {
			some rule_index, contexts in ast.found.vars
			some var in contexts.args
		}
		
		# Variable references from function calls
		arg_tokens[var] := "reference" if {
			some rule_index, calls in ast.function_calls
			some call in calls
			some var in call.args
			var.type == "var"
			
			arg_names := {v.value | some v in ast.found.vars[rule_index].args}
			var.value in arg_names
		}
		
		# Variable references from expressions
		arg_tokens[var] := "reference" if {
			some rule_index, expressions in ast.found.expressions
			some expr in expressions
			some term in expr.terms
			term.type == "var"
			
			arg_names := {v.value | some v in ast.found.vars[rule_index].args}
			term.value in arg_names
			
			var := term
		}
	`)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse module here: %w", err)
	}

	query := ast.MustParseBody(`arg_tokens = data.semantictokens.arg_tokens`)

	resultHandler := func(result ast.Value) error {
		// Process variable tokens (args) from rego
		resultObj := result.(ast.Object)

		resultObj.Iter(func(varTerm, typeTerm *ast.Term) error {
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

			return nil
		})

		return nil
	}

	q, err := ogre.New(query).
		WithModules(map[string]*ast.Module{"query.rego": queryModule}).
		Prepare(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to run ogre query: %w", err)
	}

	err = q.Evaluator().
		WithInput(roastInput).
		WithResultHandler(resultHandler).
		Eval(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to evaluate ogre query: %w", err)
	}

	result := encodeTokens(tokens)

	return result, nil
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
