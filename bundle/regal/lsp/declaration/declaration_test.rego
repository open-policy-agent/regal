package regal.lsp.declaration_test

import data.regal.ast
import data.regal.lsp.declaration

test_declaration_location if {
	policy := `allow if {
		some variable

		input[variable] = "some value"
	}`
	module := ast.policy(policy)

	r := declaration.result.response
		with data.workspace.parsed["file:///p.rego"] as module
		with input.regal as module.regal
		with input.params.position.line as 5
		with input.params.position.character as 12
		with input.params.textDocument.uri as "file:///p.rego"

	r == {
		"uri": "file:///p.rego",
		"range": {
			"start": {
				"line": 3,
				"character": 7,
			},
			"end": {
				"line": 3,
				"character": 15,
			},
		},
	}
}

test_declaration_ref_head_location if {
	policy := `allow if {
		some variable

		input[variable]

		variable.foo.bar
	}`
	module := ast.policy(policy)

	r := declaration.result.response
		with data.workspace.parsed["file:///p.rego"] as module
		with input.regal as module.regal
		with input.params.position.line as 7
		with input.params.position.character as 4
		with input.params.textDocument.uri as "file:///p.rego"

	r == {
		"uri": "file:///p.rego",
		"range": {
			"start": {
				"line": 3,
				"character": 7,
			},
			"end": {
				"line": 3,
				"character": 15,
			},
		},
	}
}

test_declaration_location_link_support if {
	policy := `allow if {
		some variable

		input[variable] = "some value"
	}`
	module := ast.policy(policy)

	r := declaration.result.response
		with data.client.capabilities.textDocument.declaration.linkSupport as true
		with data.workspace.parsed["file:///p.rego"] as module
		with input.regal as module.regal
		with input.params.position.line as 5
		with input.params.position.character as 12
		with input.params.textDocument.uri as "file:///p.rego"

	r == [{
		"targetUri": "file:///p.rego",
		"targetRange": {
			"start": {
				"line": 2,
				"character": 0,
			},
			"end": {
				"line": 6,
				"character": 2,
			},
		},
		"targetSelectionRange": {
			"start": {
				"line": 3,
				"character": 7,
			},
			"end": {
				"line": 3,
				"character": 15,
			},
		},
	}]
}

test_declaration_ref_head_location_link_support if {
	policy := `allow if {
		some variable

		input[variable]

		variable.foo.bar
	}`
	module := ast.policy(policy)

	r := declaration.result.response
		with data.client.capabilities.textDocument.declaration.linkSupport as true
		with data.workspace.parsed["file:///p.rego"] as module
		with input.regal as module.regal
		with input.params.position.line as 7
		with input.params.position.character as 4
		with input.params.textDocument.uri as "file:///p.rego"

	r == [{
		"targetUri": "file:///p.rego",
		"targetRange": {
			"start": {
				"line": 2,
				"character": 0,
			},
			"end": {
				"line": 8,
				"character": 2,
			},
		},
		"targetSelectionRange": {
			"start": {
				"line": 3,
				"character": 7,
			},
			"end": {
				"line": 3,
				"character": 15,
			},
		},
	}]
}
