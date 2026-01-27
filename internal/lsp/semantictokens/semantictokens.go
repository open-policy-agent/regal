package semantictokens

import (
	"sort"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/regal/internal/lsp/types"
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

func Full(module *ast.Module) *types.SemanticTokens {

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

	// My thought process here was:
	// 1. mark all variables found in rule head as declared via map
	// 2. variables found in body are then references
	//
	// This is a faulty assumption though, as variables could be redeclared
	// and not necessarily be references. Could use some feedback on how to account
	// for this, I wasn't certain after looking at what the AST provides

	for _, rule := range module.Rules {
		declaredVars := make(map[string]bool)

		// Parsing rule head for declarations
		if rule.Head.Args != nil {
			for _, arg := range rule.Head.Args {
				if v, ok := arg.Value.(ast.Var); ok {
					varName := string(v)
					declaredVars[varName] = true

					tokens = append(tokens, Token{
						Line:      uint(arg.Location.Row - 1),
						Col:       uint(arg.Location.Col - 1),
						Length:    uint(len(varName)),
						Type:      TokenTypeVariable,
						Modifiers: ModifierDeclaration,
					})
				}
			}
		}

		// Parse body for usages
		bodyVisitor := ast.NewGenericVisitor(func(x any) bool {
			if term, ok := x.(*ast.Term); ok {
				if v, ok := term.Value.(ast.Var); ok {
					varName := string(v)

					if declaredVars[varName] {
						tokens = append(tokens, Token{
							Line:      uint(term.Location.Row - 1),
							Col:       uint(term.Location.Col - 1),
							Length:    uint(len(varName)),
							Type:      TokenTypeVariable,
							Modifiers: ModifierReference,
						})
					}
				}
			}
			return false
		})

		bodyVisitor.Walk(rule.Body)
	}

	result := encodeTokens(tokens)

	return result
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
