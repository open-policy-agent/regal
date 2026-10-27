package roast

import (
	"embed"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/url"
	"sync"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/encoding/noast"
	"github.com/open-policy-agent/regal/internal/roast/transforms/module"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/util"
)

var (
	//go:embed testdata
	testData    embed.FS
	annotations = sync.OnceValue(func() *ast.Annotations {
		return &ast.Annotations{
			Scope:         "document",
			Title:         "this is a title",
			Entrypoint:    true,
			Description:   "this is a description",
			Organizations: []string{"org1", "org2"},
			RelatedResources: []*ast.RelatedResourceAnnotation{
				{Description: "documentation", Ref: mustParseURL("https://example.com")},
				{Description: "other", Ref: mustParseURL("https://example.com/other")},
			},
			Authors: []*ast.AuthorAnnotation{
				{Name: "John Doe", Email: "john@example.com"},
				{Name: "Jane Doe", Email: "jane@example.com"},
			},
			Schemas: []*ast.SchemaAnnotation{
				{Path: ast.MustParseRef("input"), Schema: ast.MustParseRef("schema.input")},
				{Path: ast.MustParseRef("data.foo.bar"), Schema: ast.MustParseRef("schema.foo.bar")},
				{Path: ast.MustParseRef("data.foo.baz"), Definition: new(any(map[string]any{"type": "boolean"}))},
			},
			Custom: map[string]any{
				"key":    "value",
				"object": map[string]any{"nested": "value"},
				"list":   []any{"value1", 2, true},
			},
			Location: &ast.Location{Row: 1, Col: 2, File: "file.rego"},
		}
	})
	pkg = &ast.Package{
		Location: &ast.Location{Row: 6, Col: 1, Text: []byte("foo")},
		Path:     ast.Ref{ast.DefaultRootDocument, ast.InternedTerm("foo")},
	}
)

func TestAnnotationsEncoding(t *testing.T) {
	t.Parallel()

	roast := must.Marshal(t, annotations(), Options, jsontext.WithIndent("  "))
	expected := mustReadTestFile(t, "testdata/annotations_all.json")

	assert.JSONEqual(t, expected, roast)
}

func TestAnnotationsOnPackage(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package:     pkg,
		Annotations: []*ast.Annotations{{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"}},
	}

	// package annotations should end up on the package object and *not*
	// on the module object, contrary to how OPA currently does it
	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [{"type": "var", "value": "data"}, {"type": "string", "value": "foo"}],
    "annotations": [{"location": "1:1:1:1", "scope": "package", "title": "foo"}]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestAnnotationsOnPackageBothPackageAndSubpackagesScope(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{
			{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"},
			{Location: &ast.Location{Row: 3, Col: 1}, Scope: "subpackages", Title: "bar"},
		},
	}

	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [{"type": "var", "value": "data"}, {"type": "string", "value": "foo"}],
    "annotations": [
      {"location": "1:1:1:1", "scope": "package", "title": "foo"},
      {"location": "3:1:3:1", "scope": "subpackages", "title": "bar"}
    ]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestRuleAndDocumentScopedAnnotationsOnPackageAreDropped(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{
			{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"},
			{Location: &ast.Location{Row: 3, Col: 1}, Scope: "rule", Title: "bar"},
			{Location: &ast.Location{Row: 4, Col: 1}, Scope: "document", Title: "baz"},
		},
	}

	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [{"type": "var", "value": "data"}, {"type": "string", "value": "foo"}],
    "annotations": [{"location": "1:1:1:1", "scope": "package", "title": "foo"}]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestSerializedModuleSize(t *testing.T) {
	t.Parallel()

	policy := mustReadTestFile(t, "testdata/policy.rego")
	module := ast.MustParseModuleWithOpts(string(policy), ast.ParserOptions{ProcessAnnotation: true})

	// This test will fail whenever the size of the serialized module changes,
	// which not often and when it happens it's good to know about it, update
	// and move on.
	must.Equal(t, 79194, len(must.Marshal(t, module, Options)), "serialized module size")
}

func TestSerializedModuleNoASTRoASTEquality(t *testing.T) {
	t.Parallel()

	policy := string(mustReadTestFile(t, "testdata/policy.rego"))
	mod := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{ProcessAnnotation: true})

	v1 := must.Return(module.ToValue(mod))(t)
	v2 := must.Return(json.Marshal(mod, Options))(t)

	assert.JSONEqual(t, must.Return(json.Marshal(v1, noast.Options))(t), v2)
}

func TestRuleHeadEncoding(t *testing.T) {
	t.Parallel()

	head := &ast.Head{
		Name: "omitted",
		Reference: ast.Ref{
			{Value: ast.Var("foo"), Location: &ast.Location{Row: 1, Col: 1, Text: []byte("foo")}},
			{Value: ast.String("bar"), Location: &ast.Location{Row: 1, Col: 5, Text: []byte("bar")}},
		},
		Value:    &ast.Term{Value: ast.Boolean(true), Location: &ast.Location{Row: 1, Col: 12, Text: []byte("true")}},
		Assign:   true,
		Location: &ast.Location{Row: 1, Col: 1, Text: []byte("foo.bar := true")},
	}
	bs := must.Marshal(t, head, Options, jsontext.WithIndent("  "))

	expect := `{
  "location": "1:1:1:16",
  "ref": [
    {"location": "1:1:1:4", "type": "var", "value": "foo"},
    {"location": "1:5:1:8", "type": "string", "value": "bar"}
  ],
  "assign": true,
  "value": {"location": "1:12:1:16", "type": "boolean", "value": true}
}`
	assert.JSONEqual(t, expect, bs)
}

func TestRuleHeadEncodingStripsLocationOfGeneratedValue(t *testing.T) {
	t.Parallel()

	head := ast.MustParseRule(`p[x] if { x := 1 }`).Head
	bs := must.Marshal(t, head, Options, jsontext.WithIndent("  "))

	expected := `{
  "location": "1:1:1:5",
  "ref": [
    {"location": "1:1:1:2", "type": "var", "value": "p"},
    {"location": "1:3:1:4", "type": "var", "value": "x"}
  ],
  "key": {"location": "1:3:1:4", "type": "var", "value": "x"},
  "value": {"type": "boolean", "value": true}
}`
	assert.JSONEqual(t, expected, bs)
}

func TestTemplateStringMarshalling(t *testing.T) {
	t.Parallel()

	templateStr := &ast.TemplateString{
		MultiLine: true,
		Parts: []ast.Node{
			ast.StringTerm("foo ").SetLocation(ast.NewLocation([]byte("foo "), "p.rego", 10, 1)),
			&ast.Expr{
				Location: ast.NewLocation([]byte("bar"), "p.rego", 10, 6),
				Terms:    ast.VarTerm("bar").SetLocation(ast.NewLocation([]byte("bar"), "p.rego", 10, 6)),
			},
		},
	}

	assert.JSONEqual(t, []byte(`{"multi_line":true,`+
		`"parts":[`+
		`{"location":"10:1:10:5","type":"string","value":"foo "},`+
		`{"interpolated":true,"location":"10:6:10:9","terms":{"location":"10:6:10:9","type":"var","value":"bar"}}`+
		`]}`),
		must.Marshal(t, templateStr, Options))
}

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
    "location": "5:6:5:27",
    "type": "not",
    "body": [
      {
        "location": "5:6:5:27",
        "terms": [
          {
            "location": "5:22:5:24",
            "type": "ref",
            "value": [{"location": "5:22:5:24", "type": "var", "value": "equal"}]
          },
          {
            "location": "5:10:5:21",
            "type": "call",
            "value": [
              {
                "location": "5:18:5:19",
                "type": "ref",
                "value": [{"location": "5:18:5:19", "type": "var", "value": "plus"}]
              },
              {
                "location": "5:10:5:17",
                "type": "ref",
                "value": [
                  {"location": "5:10:5:15", "type": "var", "value": "input"},
                  {"location": "5:16:5:17", "type": "string", "value": "x"}
                ]
              },
              {"location": "5:20:5:21", "type": "number", "value": 2}
            ]
          },
          {"location": "5:25:5:27", "type": "number", "value": 42}
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
    "location": "5:6:10:7",
    "type": "not",
    "explicit_body": true,
    "body": [
      {
        "location": "6:7:6:19",
        "terms": [
          {
            "location": "6:9:6:11",
            "type": "ref",
            "value": [{"location": "6:9:6:11", "type": "var", "value": "assign"}]
          },
          {"location": "6:7:6:8", "type": "var", "value": "x"},
          {
            "location": "6:12:6:19",
            "type": "ref",
            "value": [
              {"location": "6:12:6:17", "type": "var", "value": "input"},
              {"location": "6:18:6:19", "type": "string", "value": "x"}
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
            "value": [{"location": "7:9:7:11", "type": "var", "value": "assign"}]
          },
          {"location": "7:7:7:8", "type": "var", "value": "y"},
          {"location": "7:12:7:13", "type": "number", "value": 2}
        ]
      },
      {
        "location": "8:7:8:17",
        "terms": [
          {
            "location": "8:9:8:11",
            "type": "ref",
            "value": [{"location": "8:9:8:11", "type": "var", "value": "assign"}]
          },
          {"location": "8:7:8:8", "type": "var", "value": "z"},
          {
            "location": "8:12:8:17",
            "type": "call",
            "value": [
              {
                "location": "8:14:8:15",
                "type": "ref",
                "value": [{"location": "8:14:8:15", "type": "var", "value": "plus"}]
              },
              {"location": "8:12:8:13", "type": "var", "value": "x"},
              {"location": "8:16:8:17", "type": "var", "value": "y"}
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
            "value": [{ "location": "9:9:9:11", "type": "var", "value": "equal"}]
          },
          {"location": "9:7:9:8", "type": "var", "value": "z"},
          {"location": "9:12:9:14", "type": "number", "value": 42}
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

			assert.JSONEqual(t, tc.exp, must.Marshal(t, tc.expr, Options))
		})
	}
}

func TestLocation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		location *ast.Location
		expected string
	}{{
		name:     "multiple lines",
		location: &ast.Location{Row: 5, Col: 2, Text: []byte("allow if {\n	input.foo == true\n}")},
		expected: `"5:2:7:2"`,
	}, {
		name:     "single line",
		location: &ast.Location{Row: 1, Col: 1, Text: []byte("package example")},
		expected: `"1:1:1:16"`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.JSONEqual(t, tc.expected, must.Marshal(t, tc.location, Options))
		})
	}
}

func TestLocationHeadValue(t *testing.T) {
	// Separate test for this as we found the end position would sometimes be off,
	// e.g. the end column would be presented as before the start column.
	t.Parallel()

	mod := ast.MustParseModule("package foo.bar\n\nrule := true")
	expect := `{
  "package": {
    "location": "1:1:1:8",
    "path": [
      {"type": "var", "value": "data"},
      {"location": "1:9:1:12", "type": "string", "value": "foo"},
      {"location": "1:13:1:16", "type": "string", "value": "bar"}
    ]
  },
  "rules": [
    {
      "location": "3:1:3:13",
      "head": {
        "location": "3:1:3:13",
        "ref": [{"location": "3:1:3:5", "type": "var", "value": "rule"}],
        "assign": true,
        "value": {"location": "3:9:3:13", "type": "boolean", "value": true}
      }
    }
  ]
}`
	assert.JSONEqual(t, expect, must.Marshal(t, mod, Options))
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
			expr: ast.MustParseModule(`package test
import future.keywords.or

p if {
	input.a or input.b
}
`).Rules[0].Body[0],
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
            {"location": "5:2:5:7", "type": "var", "value": "input"},
            {"location": "5:8:5:9", "type": "string", "value": "a"}
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
            {"location": "5:13:5:18", "type": "var", "value": "input"},
            {"location": "5:19:5:20", "type": "string", "value": "b"}
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
			expr: ast.MustParseModule(`package test
import future.keywords.and
import future.keywords.or

p if {
	input.a or input.b and input.c
}
`).Rules[0].Body[0],
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
            {"location": "6:2:6:7", "type": "var", "value": "input"},
            {"location": "6:8:6:9", "type": "string", "value": "a"}
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
                  {"location": "6:13:6:18", "type": "var", "value": "input"},
                  {"location": "6:19:6:20", "type": "string", "value": "b"}
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
                  {"location": "6:25:6:30", "type": "var", "value": "input"},
                  {"location": "6:31:6:32", "type": "string", "value": "c"}
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
			expr: ast.MustParseModule(`package test
import future.keywords.and

p if {
	{
		x := input.a
		x == 1
	} and input.b
}
`).Rules[0].Body[0],
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
          {"location": "6:5:6:7", "type": "ref", "value": [{"location": "6:5:6:7", "type": "var", "value": "assign"}]},
          {"location": "6:3:6:4", "type": "var", "value": "x"},
          {
            "location": "6:8:6:15",
            "type": "ref",
            "value": [
              {"location": "6:8:6:13", "type": "var", "value": "input"},
              {"location": "6:14:6:15", "type": "string", "value": "a"}
            ]
          }
        ]
      },
      {
        "location": "7:3:7:9",
        "terms": [
          {"location": "7:5:7:7", "type": "ref", "value": [{"location": "7:5:7:7", "type": "var", "value": "equal"}]},
          {"location": "7:3:7:4", "type": "var", "value": "x"},
          {"location": "7:8:7:9", "type": "number", "value": 1}
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
            {"location": "8:8:8:13", "type": "var", "value": "input"},
            {"location": "8:14:8:15", "type": "string", "value": "b"}
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

			assert.JSONEqual(t, tc.exp, must.Marshal(t, tc.expr, Options))
		})
	}
}

func mustReadTestFile(tb testing.TB, path string) []byte {
	tb.Helper()

	b, err := testData.ReadFile(path)
	if err != nil {
		tb.Fatalf("Read file %s: %v", path, err)
	}

	return b
}

func mustParseURL(raw string) url.URL {
	return *util.Must(url.Parse(raw))
}
