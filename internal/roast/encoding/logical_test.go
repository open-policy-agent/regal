package encoding

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
)

// experimentalOpts as the `and` and `or` keywords aren't advertised in OPA's
// default capabilities while still considered experimental.
var experimentalOpts = ast.ParserOptions{
	Capabilities: ast.CapabilitiesForThisVersion(ast.CapabilitiesExperimentalKeywords(true)),
}

func TestLogicalKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		note string
		expr any
		exp  string
	}{
		{
			note: "or",
			expr: ast.MustParseModuleWithOpts(`package test
import future.keywords.or

p if {
	input.a or input.b
}
`, experimentalOpts).Rules[0].Body[0],
			exp: `{
  "location": "5:2:5:20",
  "terms": {
    "location": "5:2:5:20",
    "type": "or",
    "lhs": [
      {
        "location": "5:2:5:9",
        "terms": {
          "location": "5:2:5:9",
          "type": "ref",
          "value": [
            {
              "location": "5:2:5:7",
              "type": "var",
              "value": "input"
            },
            {
              "location": "5:8:5:9",
              "type": "string",
              "value": "a"
            }
          ]
        }
      }
    ],
    "rhs": [
      {
        "location": "5:13:5:20",
        "terms": {
          "location": "5:13:5:20",
          "type": "ref",
          "value": [
            {
              "location": "5:13:5:18",
              "type": "var",
              "value": "input"
            },
            {
              "location": "5:19:5:20",
              "type": "string",
              "value": "b"
            }
          ]
        }
      }
    ]
  }
}`,
		},
		{
			// `and` binds tighter than `or`, so this is `a or (b and c)`
			note: "and nested in or",
			expr: ast.MustParseModuleWithOpts(`package test
import future.keywords.and
import future.keywords.or

p if {
	input.a or input.b and input.c
}
`, experimentalOpts).Rules[0].Body[0],
			exp: `{
  "location": "6:2:6:32",
  "terms": {
    "location": "6:2:6:32",
    "type": "or",
    "lhs": [
      {
        "location": "6:2:6:9",
        "terms": {
          "location": "6:2:6:9",
          "type": "ref",
          "value": [
            {
              "location": "6:2:6:7",
              "type": "var",
              "value": "input"
            },
            {
              "location": "6:8:6:9",
              "type": "string",
              "value": "a"
            }
          ]
        }
      }
    ],
    "rhs": [
      {
        "location": "6:13:6:32",
        "terms": {
          "location": "6:13:6:32",
          "type": "and",
          "lhs": [
            {
              "location": "6:13:6:20",
              "terms": {
                "location": "6:13:6:20",
                "type": "ref",
                "value": [
                  {
                    "location": "6:13:6:18",
                    "type": "var",
                    "value": "input"
                  },
                  {
                    "location": "6:19:6:20",
                    "type": "string",
                    "value": "b"
                  }
                ]
              }
            }
          ],
          "rhs": [
            {
              "location": "6:25:6:32",
              "terms": {
                "location": "6:25:6:32",
                "type": "ref",
                "value": [
                  {
                    "location": "6:25:6:30",
                    "type": "var",
                    "value": "input"
                  },
                  {
                    "location": "6:31:6:32",
                    "type": "string",
                    "value": "c"
                  }
                ]
              }
            }
          ]
        }
      }
    ]
  }
}`,
		},
		{
			note: "and with explicit lhs body",
			expr: ast.MustParseModuleWithOpts(`package test
import future.keywords.and

p if {
	{
		x := input.a
		x == 1
	} and input.b
}
`, experimentalOpts).Rules[0].Body[0],
			exp: `{
  "location": "5:2:8:15",
  "terms": {
    "location": "5:2:8:15",
    "type": "and",
    "explicit_lhs": true,
    "lhs": [
      {
        "location": "6:3:6:15",
        "terms": [
          {
            "location": "6:5:6:7",
            "type": "ref",
            "value": [
              {
                "location": "6:5:6:7",
                "type": "var",
                "value": "assign"
              }
            ]
          },
          {
            "location": "6:3:6:4",
            "type": "var",
            "value": "x"
          },
          {
            "location": "6:8:6:15",
            "type": "ref",
            "value": [
              {
                "location": "6:8:6:13",
                "type": "var",
                "value": "input"
              },
              {
                "location": "6:14:6:15",
                "type": "string",
                "value": "a"
              }
            ]
          }
        ]
      },
      {
        "location": "7:3:7:9",
        "terms": [
          {
            "location": "7:5:7:7",
            "type": "ref",
            "value": [
              {
                "location": "7:5:7:7",
                "type": "var",
                "value": "equal"
              }
            ]
          },
          {
            "location": "7:3:7:4",
            "type": "var",
            "value": "x"
          },
          {
            "location": "7:8:7:9",
            "type": "number",
            "value": 1
          }
        ]
      }
    ],
    "rhs": [
      {
        "location": "8:8:8:15",
        "terms": {
          "location": "8:8:8:15",
          "type": "ref",
          "value": [
            {
              "location": "8:8:8:13",
              "type": "var",
              "value": "input"
            },
            {
              "location": "8:14:8:15",
              "type": "string",
              "value": "b"
            }
          ]
        }
      }
    ]
  }
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.note, func(t *testing.T) {
			t.Parallel()

			bs := must.Return(jsoniter.ConfigFastest.MarshalIndent(tc.expr, "", "  "))(t)
			act := string(bs)

			if diff := cmp.Diff(tc.exp, act); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
