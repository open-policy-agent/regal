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

func Full(ctx context.Context, queryResult map[string]any) (*types.SemanticTokens, error) {
	tokens := make([]Token, 0)

	packageTokens, err := extractPackageTokens(queryResult)
	if err != nil {
		return nil, fmt.Errorf("failed to extract package tokens: %w", err)
	}
	tokens = append(tokens, packageTokens...)

	variableTokens, err := extractVariableTokens(queryResult)
	if err != nil {
		return nil, fmt.Errorf("failed to extract variable tokens: %w", err)
	}
	tokens = append(tokens, variableTokens...)

	importTokens, err := extractImportTokens(queryResult)
	if err != nil {
		return nil, fmt.Errorf("failed to extract import tokens: %w", err)
	}
	tokens = append(tokens, importTokens...)

	return encodeTokens(tokens), nil
}

func extractPackageTokens(queryResult map[string]any) ([]Token, error) {
	var tokens []Token

	if packageTokens, ok := queryResult["package_tokens"]; ok {

		if tokenSlice, ok := packageTokens.([]any); ok {
			for _, tokenItem := range tokenSlice[1:] {
				if tokenMap, ok := tokenItem.(map[string]any); ok {
					locationStr, ok := tokenMap["location"].(string)
					if !ok {
						return nil, fmt.Errorf("Error parsing ast key")
					}

					rowStr, rest, _ := strings.Cut(locationStr, ":")
					colStr, _, _ := strings.Cut(rest, ":")

					row, _ := strconv.Atoi(rowStr)
					col, _ := strconv.Atoi(colStr)

					length, err := getTokenLengthFromLocation(locationStr)
					if err != nil {
						return nil, fmt.Errorf("failed to get token length: %w", err)
					}

					token := Token{
						Line:      uint(row - 1),
						Col:       uint(col - 1),
						Length:    length,
						Type:      TokenTypePackage,
						Modifiers: 0,
					}

					tokens = append(tokens, token)

				}
			}
		}
	}

	return tokens, nil
}

func extractVariableTokens(queryResult map[string]any) ([]Token, error) {
	var tokens []Token

	if argTokens, ok := queryResult["arg_tokens"]; ok {

		if argTokensMap, ok := argTokens.(map[string]any); ok {
			for typeKey, varData := range argTokensMap {
				typeStr := typeKey

				if varSlice, ok := varData.([]any); ok {
					for _, varItem := range varSlice {
						if varMap, ok := varItem.(map[string]any); ok {
							locationStr, ok := varMap["location"].(string)
							if !ok {
								return nil, fmt.Errorf("Error parsing ast key")
							}

							rowStr, rest, _ := strings.Cut(locationStr, ":")
							colStr, _, _ := strings.Cut(rest, ":")

							row, _ := strconv.Atoi(rowStr)
							col, _ := strconv.Atoi(colStr)

							length, err := getTokenLengthFromLocation(locationStr)
							if err != nil {
								return nil, fmt.Errorf("failed to get token length: %w", err)
							}

							modifier := ModifierReference
							if typeStr == "declaration" {
								modifier = ModifierDeclaration
							}

							token := Token{
								Line:      uint(row - 1),
								Col:       uint(col - 1),
								Length:    length,
								Type:      TokenTypeVariable,
								Modifiers: uint(modifier),
							}

							tokens = append(tokens, token)
						}
					}
				}
			}
		}
	}

	return tokens, nil
}

func extractImportTokens(queryResult map[string]any) ([]Token, error) {
	var tokens []Token

	if importTokens, ok := queryResult["import_tokens"]; ok {

		if tokenSlice, ok := importTokens.([]any); ok {
			for _, tokenItem := range tokenSlice {
				if tokenMap, ok := tokenItem.(map[string]any); ok {
					locationStr, ok := tokenMap["location"].(string)
					if !ok {
						return nil, fmt.Errorf("Error parsing ast key")
					}

					rowStr, rest, _ := strings.Cut(locationStr, ":")
					colStr, _, _ := strings.Cut(rest, ":")

					row, _ := strconv.Atoi(rowStr)
					col, _ := strconv.Atoi(colStr)

					length, err := getTokenLengthFromLocation(locationStr)
					if err != nil {
						return nil, fmt.Errorf("failed to get token length: %w", err)
					}

					token := Token{
						Line:      uint(row - 1),
						Col:       uint(col - 1),
						Length:    length,
						Type:      TokenTypeImport,
						Modifiers: 0,
					}

					tokens = append(tokens, token)

				}
			}
		}
	}

	return tokens, nil
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

// getTokenLengthFromLocation calculates token length from location span
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
