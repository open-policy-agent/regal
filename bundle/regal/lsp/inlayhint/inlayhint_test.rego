package regal.lsp.inlayhint_test

import data.regal.ast

import data.regal.lsp.inlayhint

test_inlayhints_rendered_correctly_no_tooltip_resolve if {
	module := ast.policy(`inlay_hints if {
		startswith(text, sub)
	}`)

	res := inlayhint.result.response
		with input as {
			"regal": {"file": {
				"parse_errors": [],
				"uri": "file:///p.rego",
			}},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 100, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == {
		{
			"kind": 2,
			"label": "base:",
			"paddingRight": true,
			"position": {
				"character": 13,
				"line": 3,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`base` — string: base string",
			},
		},
		{
			"kind": 2,
			"label": "search:",
			"paddingRight": true,
			"position": {
				"character": 19,
				"line": 3,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`search` — string: search string",
			},
		},
	}
}

test_inlayhints_rendered_correctly_tooltip_resolve if {
	module := ast.policy(`inlay_hints if {
		startswith(text, sub)
	}`)

	res := inlayhint.result.response
		with input as {
			"regal": {
				"client": {"capabilities": {"textDocument": {"inlayHint": {"resolveSupport": {"properties": ["tooltip"]}}}}},
				"file": {
					"parse_errors": [],
					"uri": "file:///p.rego",
				},
			},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 100, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == {
		{
			"kind": 2,
			"label": "base:",
			"paddingRight": true,
			"position": {
				"character": 13,
				"line": 3,
			},
			"data": {
				"name": "base",
				"type": "string",
				"description": "base string",
			},
		},
		{
			"kind": 2,
			"label": "search:",
			"paddingRight": true,
			"position": {
				"character": 19,
				"line": 3,
			},
			"data": {
				"name": "search",
				"type": "string",
				"description": "search string",
			},
		},
	}
}

test_inlayhints_rendered_correctly_custom_function if {
	module := ast.policy(`
	custom(foo, bar) if foo > bar

	inlay_hints if custom(a, b)`)

	res := inlayhint.result.response
		with input as {
			"regal": {"file": {
				"parse_errors": [],
				"uri": "file:///p.rego",
			}},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 100, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == {
		{
			"kind": 2,
			"label": "bar:",
			"paddingRight": true,
			"position": {
				"character": 26,
				"line": 5,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`bar` — any",
			},
		},
		{
			"kind": 2,
			"label": "foo:",
			"paddingRight": true,
			"position": {
				"character": 23,
				"line": 5,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`foo` — any",
			},
		},
	}
}

test_inlayhints_rendered_correctly_custom_function_non_var_argument if {
	module := ast.policy(`
	custom(foo, "bar") if foo == "foo"

	inlay_hints if custom(a, b)`)

	res := inlayhint.result.response
		with input as {
			"regal": {"file": {
				"parse_errors": [],
				"uri": "file:///p.rego",
			}},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 100, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == {
		{
			"kind": 2,
			"label": "foo:",
			"paddingRight": true,
			"position": {
				"character": 23,
				"line": 5,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`foo` — any",
			},
		},
		{
			"kind": 2,
			"label": "bar:",
			"paddingRight": true,
			"position": {
				"character": 26,
				"line": 5,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`bar` — string",
			},
		},
	}
}

test_only_calls_in_range if {
	module := ast.policy(`inlay_hints if {
		startswith(text1, sub)

		startswith(text2, sub2)
	}`)

	res := inlayhint.result.response
		with input as {
			"regal": {"file": {
				"parse_errors": [],
				"uri": "file:///p.rego",
			}},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 3, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == {
		{
			"kind": 2,
			"label": "base:",
			"paddingRight": true,
			"position": {
				"character": 13,
				"line": 3,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`base` — string: base string",
			},
		},
		{
			"kind": 2,
			"label": "search:",
			"paddingRight": true,
			"position": {
				"character": 20,
				"line": 3,
			},
			"tooltip": {
				"kind": "markdown",
				"value": "`search` — string: search string",
			},
		},
	}
}

test_null_if_parse_errors if {
	module := ast.policy(`inlay_hints if {
		startswith(text, sub)
	}`)

	res := inlayhint.result.response
		with input as {
			"regal": {"file": {
				"parse_errors": ["some error"],
				"uri": "file:///p.rego",
			}},
			"params": {"range": {
				"start": {"line": 0, "character": 0},
				"end": {"line": 100, "character": 100},
			}},
		}
		with data.workspace.parsed["file:///p.rego"] as module
		with data.workspace.builtins as _builtins

	res == null
}

_builtins := {"startswith": {"decl": {"args": [
	{"name": "base", "type": "string", "description": "base string"},
	{"name": "search", "type": "string", "description": "search string"},
]}}}
