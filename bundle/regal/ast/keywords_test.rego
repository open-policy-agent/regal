package regal.ast_test

import data.regal.ast

test_keywords_package if {
	kwds := ast.keywords with input as ast.policy("")

	count(kwds) == 1 # lines with keywords

	_keyword_on_row(
		kwds,
		1,
		{
			"name": "package",
			"location": {"row": 1, "col": 1},
		},
	)
}

test_keywords_import if {
	kwds := ast.keywords with input as ast.policy("import data.foo")

	count(kwds) == 2 # lines with keywords

	_keyword_on_row(
		kwds,
		3,
		{
			"name": "import",
			"location": {"row": 3, "col": 1},
		},
	)
}

test_keywords_if if {
	policy := `package policy

allow if {
	# if things
	true
}
`

	kwds := ast.keywords with input as object.union(
		regal.parse_module("p.rego", policy),
		{"regal": {"file": {"lines": split(policy, "\n")}}},
	)

	count(kwds) == 2 # lines with keywords

	_keyword_on_row(
		kwds,
		3,
		{
			"name": "if",
			"location": {"row": 3, "col": 7},
		},
	)
}

test_keywords_if_on_another_line if {
	policy := `package policy

allow contains {
	"foo": true,
} if {
	# if things
	true
}
`

	kwds := ast.keywords with input as object.union(
		regal.parse_module("p.rego", policy),
		{"regal": {"file": {"lines": split(policy, "\n")}}},
	)

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		5,
		{
			"name": "if",
			"location": {"row": 5, "col": 3},
		},
	)
}

test_keywords_some_in if {
	policy := ast.with_rego_v1(`
allow if {
	some e in [1,2,3]
}`)

	kwds := ast.keywords with input as policy

	count(kwds) == 4 # lines with keywords

	_keyword_on_row(
		kwds,
		7,
		{"name": "some", "location": {"row": 7, "col": 2}},
	)

	_keyword_on_row(
		kwds,
		7,
		{"name": "in", "location": {"row": 7, "col": 9}},
	)
}

test_keywords_some_no_body if {
	policy := ast.with_rego_v1(`list := [e|
	some e in [1,2,3]
]`)

	kwds := ast.keywords with input as policy

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		6,
		{
			"name": "some",
			"location": {"row": 6, "col": 2, "end": {"col": 6, "row": 6}, "text": "some"},
		},
	)

	_keyword_on_row(
		kwds,
		6,
		{
			"name": "in",
			"location": {"row": 6, "col": 9, "end": {"col": 11, "row": 6}, "text": "in"},
		},
	)
}

test_keywords_some_in_func_arg if {
	policy := ast.with_rego_v1(`foo := concat(".", [part |
	some part in ["a","b","c"]
])`)

	kwds := ast.keywords with input as policy

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		6,
		{"name": "some", "location": {"row": 6, "col": 2}},
	)

	_keyword_on_row(
		kwds,
		6,
		{"name": "in", "location": {"row": 6, "col": 12}},
	)
}

test_keywords_contains if {
	policy := ast.with_rego_v1(`
messages contains "hello" if {
	1 == 1
}`)

	kwds := ast.keywords with input as policy

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		6,
		{"name": "contains", "location": {"row": 6, "col": 10}},
	)

	_keyword_on_row(
		kwds,
		6,
		{"name": "if", "location": {"row": 6, "col": 27}},
	)
}

test_keywords_every if {
	policy := ast.with_rego_v1(`
allow if {
	every k in [1,2,3] {
		k == "foo"
	}
}`)

	kwds := ast.keywords with input as policy

	count(kwds) == 4 # lines with keywords

	_keyword_on_row(
		kwds,
		7,
		{"name": "every", "location": {"row": 7, "col": 2, "end": {"col": 7, "row": 7}}},
	)

	_keyword_on_row(
		kwds,
		7,
		{"name": "in", "location": {"row": 7, "col": 10, "end": {"col": 12, "row": 7}}},
	)
}

_keyword_on_row(kwds, row, keyword) if {
	some kwd in kwds[row]

	kwd.name == keyword.name
	kwd.location.row == keyword.location.row
	kwd.location.col == keyword.location.col
}
