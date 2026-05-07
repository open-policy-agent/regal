package encoding

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestNotImport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		note string
		expr any
		exp  string
	}{
		{
			note: "implicit body",
			expr: ast.MustParseModule(`package test
				import future.keywords.not
				
				p if {
					not input.x + 2 == 42
				}
			`).Rules[0].Body[0],
			exp: `{
  "location": "5:6:5:27",
  "terms": {
    "location": "6:5:6:6",
    "type": "not",
    "body": [
      {
        "terms": [
          {
            "location": "5:22:5:24",
            "type": "ref",
            "value": [
              {
                "location": "5:22:5:24",
                "type": "var",
                "value": "equal"
              }
            ]
          },
          {
            "location": "5:10:5:21",
            "type": "call",
            "value": [
              {
                "location": "5:18:5:19",
                "type": "ref",
                "value": [
                  {
                    "location": "5:18:5:19",
                    "type": "var",
                    "value": "plus"
                  }
                ]
              },
              {
                "location": "5:10:5:17",
                "type": "ref",
                "value": [
                  {
                    "location": "5:10:5:15",
                    "type": "var",
                    "value": "input"
                  },
                  {
                    "location": "5:16:5:17",
                    "type": "string",
                    "value": "x"
                  }
                ]
              },
              {
                "location": "5:20:5:21",
                "type": "number",
                "value": 2
              }
            ]
          },
          {
            "location": "5:25:5:27",
            "type": "number",
            "value": 42
          }
        ]
      }
    ]
  }
}`,
		},
		{
			note: "explicit body",
			expr: ast.MustParseModule(`package test
				import future.keywords.not

				p if {
					not {
						x := input.x
						y := 2
						z := x + y
						z == 42
					}
				}
			`).Rules[0].Body[0],
			exp: `{
  "location": "5:6:10:7",
  "terms": {
    "location": "5:10:5:11",
    "type": "not",
    "explicit_body": true,
    "body": [
      {
        "location": "6:7:6:19",
        "terms": [
          {
            "location": "6:9:6:11",
            "type": "ref",
            "value": [
              {
                "location": "6:9:6:11",
                "type": "var",
                "value": "assign"
              }
            ]
          },
          {
            "location": "6:7:6:8",
            "type": "var",
            "value": "x"
          },
          {
            "location": "6:12:6:19",
            "type": "ref",
            "value": [
              {
                "location": "6:12:6:17",
                "type": "var",
                "value": "input"
              },
              {
                "location": "6:18:6:19",
                "type": "string",
                "value": "x"
              }
            ]
          }
        ]
      },
      {
        "location": "7:7:7:13",
        "terms": [
          {
            "location": "7:9:7:11",
            "type": "ref",
            "value": [
              {
                "location": "7:9:7:11",
                "type": "var",
                "value": "assign"
              }
            ]
          },
          {
            "location": "7:7:7:8",
            "type": "var",
            "value": "y"
          },
          {
            "location": "7:12:7:13",
            "type": "number",
            "value": 2
          }
        ]
      },
      {
        "location": "8:7:8:17",
        "terms": [
          {
            "location": "8:9:8:11",
            "type": "ref",
            "value": [
              {
                "location": "8:9:8:11",
                "type": "var",
                "value": "assign"
              }
            ]
          },
          {
            "location": "8:7:8:8",
            "type": "var",
            "value": "z"
          },
          {
            "location": "8:12:8:17",
            "type": "call",
            "value": [
              {
                "location": "8:14:8:15",
                "type": "ref",
                "value": [
                  {
                    "location": "8:14:8:15",
                    "type": "var",
                    "value": "plus"
                  }
                ]
              },
              {
                "location": "8:12:8:13",
                "type": "var",
                "value": "x"
              },
              {
                "location": "8:16:8:17",
                "type": "var",
                "value": "y"
              }
            ]
          }
        ]
      },
      {
        "location": "9:7:9:14",
        "terms": [
          {
            "location": "9:9:9:11",
            "type": "ref",
            "value": [
              {
                "location": "9:9:9:11",
                "type": "var",
                "value": "equal"
              }
            ]
          },
          {
            "location": "9:7:9:8",
            "type": "var",
            "value": "z"
          },
          {
            "location": "9:12:9:14",
            "type": "number",
            "value": 42
          }
        ]
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
