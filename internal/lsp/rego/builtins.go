package rego

import (
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/io"
)

var BuiltinsForDefaultCapabilities = sync.OnceValue(func() map[string]*ast.Builtin {
	return BuiltinsForCapabilities(io.Capabilities())
})

// BuiltinsForCapabilities returns a list of builtins from the provided capabilities.
func BuiltinsForCapabilities(capabilities *ast.Capabilities) map[string]*ast.Builtin {
	m := make(map[string]*ast.Builtin, len(capabilities.Builtins))
	for _, b := range capabilities.Builtins {
		m[b.Name] = b
	}

	return m
}

func BuiltinCategory(builtin *ast.Builtin) string {
	if len(builtin.Categories) == 0 {
		if i := strings.Index(builtin.Name, "."); i > -1 {
			return builtin.Name[:i]
		}

		return builtin.Name
	}

	return builtin.Categories[0]
}
