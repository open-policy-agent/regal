package encoding

import (
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestTemplateStringMarshalling(t *testing.T) {
	t.Parallel()

	templateStr := ast.TemplateString{
		MultiLine: true,
		Parts: []ast.Node{
			ast.StringTerm("foo ").SetLocation(ast.NewLocation([]byte("foo "), "p.rego", 10, 1)),
			&ast.Expr{
				Location: ast.NewLocation([]byte("bar"), "p.rego", 10, 6),
				Terms:    ast.VarTerm("bar").SetLocation(ast.NewLocation([]byte("bar"), "p.rego", 10, 6)),
			},
		},
	}
	stream := jsoniter.ConfigFastest.BorrowStream(nil)
	stream.WriteVal(templateStr)

	// From JSON and back to JSON, with sorted keys for comparison
	m := must.Unmarshal[map[string]any](t, stream.Buffer())
	bs := must.Return(jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(m))(t)

	expected := `{"multi_line":true,` +
		`"parts":[` +
		`{"location":"10:1:10:5","type":"string","value":"foo "},` +
		`{"interpolated":true,"location":"10:6:10:9","terms":{"location":"10:6:10:9","type":"var","value":"bar"}}` +
		`]}`

	must.Equal(t, expected, string(bs), "template string encoding")
}
