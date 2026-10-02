package client

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	c := must.UnmarshalTo[Client](t, `{
		"identifier": 1,
		"init_options": {
			"enableDebugCodelens": true
		},
		"capabilities": {
			"features": ["rego_v1"]
		}
	}`)

	assert.Equal(t, 1, c.Identifier)
	assert.Equal(t, true, c.InitOptions.EnableDebugCodelens)

	capsObj := must.Be[ast.Object](t, c.Capabilities)
	featArr := must.Be[*ast.Array](t, capsObj.Get(ast.InternedTerm("features")).Value)

	assert.Equal(t, `"rego_v1"`, featArr.Elem(0).String())
}
