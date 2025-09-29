package regal.lsp.completion.providers.locals_test

import data.regal.util

import data.regal.lsp.completion.providers.locals as provider
import data.regal.lsp.completion.providers.test_utils as utils

test_no_locals_in_completion_items if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

foo := 1

bar if {
	foo == 1
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 7, "character": 8},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}
	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 0
}

test_locals_in_completion_items if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

foo := 1

function(bar) if {
	baz := 1
	qux := b
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 8, "character": 9},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}

	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 2
	_expect_item(items, "bar", {"end": {"character": 9, "line": 8}, "start": {"character": 8, "line": 8}})
	_expect_item(items, "baz", {"end": {"character": 9, "line": 8}, "start": {"character": 8, "line": 8}})
}

test_locals_in_completion_items_function_call if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

foo := 1

function(bar) if {
	baz := 1
	qux := other_function(b)
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 8, "character": 24},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}

	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 2
	_expect_item(items, "bar", {"end": {"character": 24, "line": 8}, "start": {"character": 23, "line": 8}})
	_expect_item(items, "baz", {"end": {"character": 24, "line": 8}, "start": {"character": 23, "line": 8}})
}

test_locals_in_completion_items_rule_head_assignment if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

function(bar) := f if {
	foo := 1
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 4, "character": 18},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}
	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 1
	_expect_item(items, "foo", {"end": {"character": 18, "line": 4}, "start": {"character": 17, "line": 4}})
}

test_no_locals_in_completion_items_function_args if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

function() if {
	foo := 1
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 4, "character": 9},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}
	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 0
}

test_no_some_in_vars_suggested_on_same_line if {
	workspace := {"file:///p.rego": `package policy

import rego.v1

allow if {
	xyz := 1
	some xxx, yyy in x
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 6, "character": 18},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(workspace["file:///p.rego"], "\n"),
		}},
	}
	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	util.single_set_item(items).label == "xyz"
}

test_no_locals_in_completion_items_following_period if {
	workspace := {"file:///p.rego": `package policy

no_completion if {
	variable := "foo"
	input
}
`}

	_input := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 4, "character": 7},
		},
		"regal": {"file": {
			"uri": "file:///p.rego",
			"lines": split(replace(workspace["file:///p.rego"], "input", "input."), "\n"),
		}},
	}

	items := provider.items with input as _input with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 0
}

_expect_item(items, label, range) if {
	expected := {"detail": "local variable", "kind": 6}

	item := object.union(expected, {
		"label": label,
		"textEdit": {
			"newText": label,
			"range": range,
		},
	})

	item in items
}
