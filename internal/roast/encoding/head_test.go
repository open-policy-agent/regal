package encoding

import (
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestRuleHeadEncoding(t *testing.T) {
	t.Parallel()

	head := ast.Head{
		Name: "omitted",
		Reference: ast.Ref{
			{Value: ast.Var("foo"), Location: &ast.Location{Row: 1, Col: 1, Text: []byte("foo")}},
			{Value: ast.String("bar"), Location: &ast.Location{Row: 1, Col: 5, Text: []byte("bar")}},
		},
		Value:    &ast.Term{Value: ast.Boolean(true), Location: &ast.Location{Row: 1, Col: 12, Text: []byte("true")}},
		Assign:   true,
		Location: &ast.Location{Row: 1, Col: 1, Text: []byte("foo.bar := true")},
	}
	bs := must.Return(jsoniter.ConfigFastest.MarshalIndent(head, "", "  "))(t)

	expect := `{
  "location": "1:1:1:16",
  "ref": [
    {
      "location": "1:1:1:4",
      "type": "var",
      "value": "foo"
    },
    {
      "location": "1:5:1:8",
      "type": "string",
      "value": "bar"
    }
  ],
  "assign": true,
  "value": {
    "location": "1:12:1:16",
    "type": "boolean",
    "value": true
  }
}`
	must.Equal(t, expect, string(bs), "rule head encoding")
}

func TestRuleHeadEncodingStripsLocationOfGeneratedValue(t *testing.T) {
	t.Parallel()

	head := ast.MustParseRule(`p[x] if { x := 1 }`).Head
	bs := must.Return(jsoniter.ConfigFastest.MarshalIndent(head, "", "  "))(t)

	expected := `{
  "location": "1:1:1:5",
  "ref": [
    {
      "location": "1:1:1:2",
      "type": "var",
      "value": "p"
    },
    {
      "location": "1:3:1:4",
      "type": "var",
      "value": "x"
    }
  ],
  "key": {
    "location": "1:3:1:4",
    "type": "var",
    "value": "x"
  },
  "value": {
    "type": "boolean",
    "value": true
  }
}`
	must.Equal(t, expected, string(bs), "rule head encoding with generated value")
}
