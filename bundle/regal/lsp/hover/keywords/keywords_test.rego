package regal.lsp.hover.keywords_test

import data.regal.ast

import data.regal.lsp.hover.keywords

test_keywords_package if {
	module := ast.policy("")
	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

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
	module := ast.policy(`import data.foo`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

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
	module := ast.policy(`allow if {
	# if things
	true
}
`)
	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

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
	module := ast.policy(`allow contains {
	"foo": true,
} if {
	# if things
	true
}
`)
	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

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
	module := ast.policy(`allow if {
	some e in [1,2,3]
}`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		4,
		{"name": "some", "location": {"row": 4, "col": 2}},
	)

	_keyword_on_row(
		kwds,
		4,
		{"name": "in", "location": {"row": 4, "col": 9}},
	)
}

test_keywords_some_no_body if {
	module := ast.policy(`list := [e|
	some e in [1,2,3]
]`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

	count(kwds) == 2 # lines with keywords

	_keyword_on_row(
		kwds,
		4,
		{
			"name": "some",
			"location": {"row": 4, "col": 2, "end": {"col": 6, "row": 4}, "text": "some"},
		},
	)

	_keyword_on_row(
		kwds,
		4,
		{
			"name": "in",
			"location": {"row": 4, "col": 9, "end": {"col": 11, "row": 4}, "text": "in"},
		},
	)
}

test_keywords_some_in_func_arg if {
	module := ast.policy(`foo := concat(".", [part |
	some part in ["a","b","c"]
])`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

	count(kwds) == 2 # lines with keywords

	_keyword_on_row(
		kwds,
		4,
		{"name": "some", "location": {"row": 4, "col": 2}},
	)

	_keyword_on_row(
		kwds,
		4,
		{"name": "in", "location": {"row": 4, "col": 12}},
	)
}

test_keywords_contains if {
	module := ast.policy(`
messages contains "hello" if {
	1 == 1
}`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

	count(kwds) == 2 # lines with keywords

	_keyword_on_row(
		kwds,
		4,
		{"name": "contains", "location": {"row": 4, "col": 10}},
	)

	_keyword_on_row(
		kwds,
		4,
		{"name": "if", "location": {"row": 4, "col": 27}},
	)
}

test_keywords_every if {
	module := ast.policy(`
allow if {
	every k in [1,2,3] {
		k == "foo"
	}
}`)

	kwds := keywords.by_row
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as module.regal.file.lines

	count(kwds) == 3 # lines with keywords

	_keyword_on_row(
		kwds,
		5,
		{"name": "every", "location": {"row": 5, "col": 2, "end": {"col": 7, "row": 5}}},
	)

	_keyword_on_row(
		kwds,
		5,
		{"name": "in", "location": {"row": 5, "col": 10, "end": {"col": 12, "row": 5}}},
	)
}

_keyword_on_row(kwds, row, keyword) if {
	some kwd in kwds[row]

	kwd.name == keyword.name
	kwd.location.row == keyword.location.row
	kwd.location.col == keyword.location.col
}
