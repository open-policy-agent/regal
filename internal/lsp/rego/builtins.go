package rego

import (
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
