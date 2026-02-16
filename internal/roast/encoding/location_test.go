package encoding

import (
	"fmt"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestLocation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		location ast.Location
		expected string
	}{{
		name:     "multiple lines",
		location: ast.Location{Row: 5, Col: 2, Text: []byte("allow if {\n	input.foo == true\n}")},
		expected: "5:2:7:2",
	}, {
		name:     "single line",
		location: ast.Location{Row: 1, Col: 1, Text: []byte("package example")},
		expected: "1:1:1:16",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stream := jsoniter.ConfigFastest.BorrowStream(nil)
			stream.WriteVal(tc.location)
			assert.Equal(t, fmt.Sprintf("%q", tc.expected), string(stream.Buffer()), "location encoding")
			jsoniter.ConfigFastest.ReturnStream(stream)
		})
	}
}

func TestLocationHeadValue(t *testing.T) {
	// Separate test for this as we found the end position would sometimes be off,
	// e.g. the end column would be presented as before the start column.
	t.Parallel()

	mod := ast.MustParseModule("package foo.bar\n\nrule := true")
	out := must.Return(jsoniter.ConfigFastest.MarshalIndent(mod, "", "  "))(t)
	expect := `{
  "package": {
    "location": "1:1:1:8",
    "path": [
      {
        "type": "var",
        "value": "data"
      },
      {
        "location": "1:9:1:12",
        "type": "string",
        "value": "foo"
      },
      {
        "location": "1:13:1:16",
        "type": "string",
        "value": "bar"
      }
    ]
  },
  "rules": [
    {
      "location": "3:1:3:13",
      "head": {
        "location": "3:1:3:13",
        "ref": [
          {
            "location": "3:1:3:5",
            "type": "var",
            "value": "rule"
          }
        ],
        "assign": true,
        "value": {
          "location": "3:9:3:13",
          "type": "boolean",
          "value": true
        }
      }
    }
  ]
}`
	assert.Equal(t, expect, string(out), "module encoding")
}
