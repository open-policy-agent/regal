package semantictokens

import (
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

func Full(module *ast.Module) *types.SemanticTokens {
	// TODO: Implement full token generation
	// Returns empty tokens for now
	return &types.SemanticTokens{
		Data: []uint{},
	}
}
