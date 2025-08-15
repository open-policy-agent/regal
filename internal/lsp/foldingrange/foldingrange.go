package foldingrange

import (
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/opa/scanner"
	"github.com/open-policy-agent/regal/internal/lsp/opa/tokens"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

type stack []scanner.Position

func (s stack) Push(p scanner.Position) stack {
	return append(s, p)
}

func (s stack) Pop() (stack, scanner.Position) {
	l := len(s)

	return s[:l-1], s[l-1]
}

//nolint:gosec
func TokenRanges(policy string) []types.FoldingRange {
	scn, err := scanner.New(strings.NewReader(policy))
	if err != nil {
		panic(err)
	}

	var lastPosition scanner.Position

	curlyBraceStack := stack{}
	bracketStack := stack{}
	parensStack := stack{}

	foldingRanges := make([]types.FoldingRange, 0)

	for {
		token, position, _, errors := scn.Scan()

		if token == tokens.EOF || len(errors) > 0 {
			break
		}

		switch {
		case token == tokens.LBrace:
			curlyBraceStack = curlyBraceStack.Push(position)
		case token == tokens.RBrace && len(curlyBraceStack) > 0:
			curlyBraceStack, lastPosition = curlyBraceStack.Pop()

			startChar := uint(lastPosition.Col)

			foldingRanges = append(foldingRanges, types.FoldingRange{
				StartLine:      uint(lastPosition.Row - 1),
				StartCharacter: &startChar,
				// Note that we stop at the line _before_ the closing curly brace
				// as that shows the end of the object/set in the client, which seems
				// to be how other implementations do it
				EndLine: uint(position.Row - 2),
				Kind:    "region",
			})
		case token == tokens.LBrack:
			bracketStack = bracketStack.Push(position)
		case token == tokens.RBrack && len(bracketStack) > 0:
			bracketStack, lastPosition = bracketStack.Pop()

			startChar := uint(lastPosition.Col)

			foldingRanges = append(foldingRanges, types.FoldingRange{
				StartLine:      uint(lastPosition.Row - 1),
				StartCharacter: &startChar,
				// Note that we stop at the line _before_ the closing bracket
				// as that shows the end of the array in the client, which seems
				// to be how other implementations do it
				EndLine: uint(position.Row - 2),
				Kind:    "region",
			})
		case token == tokens.LParen:
			parensStack = parensStack.Push(position)
		case token == tokens.RParen && len(parensStack) > 0:
			parensStack, lastPosition = parensStack.Pop()

			startChar := uint(lastPosition.Col)

			foldingRanges = append(foldingRanges, types.FoldingRange{
				StartLine:      uint(lastPosition.Row - 1),
				StartCharacter: &startChar,
				// Note that we stop at the line _before_ the closing bracket
				// as that shows the end of the array in the client, which seems
				// to be how other implementations do it
				EndLine: uint(position.Row - 2),
				Kind:    "region",
			})
		}
	}

	return foldingRanges
}

//nolint:gosec
func FindAll(text string, module *ast.Module) []types.FoldingRange {
	uintZero := uint(0)

	ranges := make([]types.FoldingRange, 0)

	// Comments

	numComments := len(module.Comments)
	isBlock := false

	var startLine uint

	for i, comment := range module.Comments {
		// the comment following this is on the next line
		if i+1 < numComments && module.Comments[i+1].Location.Row == comment.Location.Row+1 {
			isBlock = true

			if i == 0 || module.Comments[i-1].Location.Row != comment.Location.Row-1 {
				startLine = uint(comment.Location.Row - 1)
			}
		} else if isBlock {
			ranges = append(ranges, types.FoldingRange{
				StartLine:      startLine,
				EndLine:        uint(comment.Location.Row - 1),
				StartCharacter: &uintZero,
				Kind:           "comment",
			})

			isBlock = false
		}
	}

	// Imports

	if len(module.Imports) > 2 {
		lastImport := module.Imports[len(module.Imports)-1]

		// note that we treat *all* imports as a single folding range,
		// as it's likely the user wants to hide all of them... but we
		// could consider instead folding "blocks" of grouped imports

		ranges = append(ranges, types.FoldingRange{
			StartLine:      uint(module.Imports[0].Location.Row - 1),
			EndLine:        uint(lastImport.Location.Row - 1),
			StartCharacter: &uintZero,
			Kind:           "imports",
		})
	}

	// Tokens — {}, [], ()

	tokenRanges := TokenRanges(text)

	return append(ranges, tokenRanges...)
}
