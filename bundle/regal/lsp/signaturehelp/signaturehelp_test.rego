package regal.lsp.signaturehelp_test

import rego.v1

import data.regal.lsp.signaturehelp as sh

test_result if {
	r := sh.signature with input as {
		"params": {"position": {
			"line": 2,
			"character": 15,
		}},
		"regal": {"file": {"lines": ["package test", "", "allow if count("]}},
	}
		with data.workspace.builtins.count as {
			"name": "count",
			"description": "Count takes a collection or string and returns the number of elements (or characters) in it.",
			"decl": {
				"args": [{
					"description": "the set/array/object/string to be counted",
					"name": "collection",
					"type": "any",
				}],
				"result": {"type": "number"},
				"type": "function",
			},
		}

	expected := {
		"signatures": [{
			"label": "count(collection: any) -> number",
			"documentation": "Count takes a collection or string and returns the number of elements (or characters) in it.",
			"parameters": [{
				"label": "collection: any",
				"documentation": "(collection: any): the set/array/object/string to be counted",
			}],
			"activeParameter": 0,
		}],
		"activeSignature": 0,
		"activeParameter": 0,
	}

	r == expected
}

test_function_at_position_simple if {
	content := `package test

import rego.v1

allow if func(`
	position := {"line": 4, "character": 14}
	expected := {
		"name": "func",
		"active_param": 1,
	}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_function_at_position_two_on_one_line if {
	content := `package test

import rego.v1

allow if {
	func1(1, 2) == func2(3,4
}`
	position := {"line": 5, "character": 25}
	expected := {
		"name": "func2",
		"active_param": 2,
	}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_function_at_position_after_function if {
	content := `package test

import rego.v1

allow if func(1) ==`
	position := {"line": 4, "character": 19}
	expected := {}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_function_at_position_multi_line if {
	content := `package test

import rego.v1

allow if func(
	1,
	2,
`
	position := {"line": 6, "character": 0}
	expected := {
		"name": "func",
		"active_param": 2,
	}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_function_at_position_cursor_in_middle_of_function if {
	content := `package test

import rego.v1

allow if func(arg1, arg2, arg3)`
	position := {"line": 4, "character": 20}
	expected := {
		"name": "func",
		"active_param": 2,
	}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_function_at_position_cursor_in_middle_multi_line if {
	content := `package test

import rego.v1

allow if func(
	arg1,
	arg2,
	arg3
) == other`
	position := {"line": 6, "character": 2}
	expected := {
		"name": "func",
		"active_param": 2,
	}

	sh._function_at_position(split(content, "\n"), position) == expected
}

test_text_up_to_position[name] if {
	some name, tc in {
		"single line": {
			"content": "hello world",
			"position": {"line": 0, "character": 5},
			"expected": "hello",
		},
		"multi-line": {
			"content": "line one\nline two\nline three",
			"position": {"line": 1, "character": 4},
			"expected": "line one\nline",
		},
		"multi-line rego": {
			"content": `package test

import rego.v1

allow if func(
	1,
	2,
`,
			"position": {"line": 6, "character": 0},
			"expected": `package test

import rego.v1

allow if func(
	1,
`,
		},
		"line beyond input": {
			"content": "hello\nworld",
			"position": {"line": 5, "character": 0},
			"expected": "hello\nworld",
		},
		"character beyond line end": {
			"content": "hello\nworld",
			"position": {"line": 1, "character": 20},
			"expected": "hello\nworld",
		},
	}

	sh._text_up_to_position(split(tc.content, "\n"), tc.content, tc.position) == tc.expected
}
