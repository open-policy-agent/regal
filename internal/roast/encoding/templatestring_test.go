package encoding

import (
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
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

	var m map[string]any
	if err := jsoniter.ConfigFastest.Unmarshal(stream.Buffer(), &m); err != nil {
		t.Fatalf("unexpected error during unmarshalling: %v", err)
	}

	// Back to JSON, with sorted keys for comparison
	bs, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(m)
	if err != nil {
		t.Fatalf("unexpected error during marshalling: %v", err)
	}

	expected := `{"multi_line":true,` +
		`"parts":[` +
		`{"location":"10:1:10:5","type":"string","value":"foo "},` +
		`{"interpolated":true,"location":"10:6:10:9","terms":{"location":"10:6:10:9","type":"var","value":"bar"}}` +
		`]}`

	if string(bs) != expected {
		t.Fatalf("expected marshalled template string to be:\n%s\nbut got:\n%s", expected, string(bs))
	}
}
