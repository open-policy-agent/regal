package regal.lsp.semantictokens.vars.packages_test

import data.regal.lsp.semantictokens.vars.packages

test_packages if {
	policy := "package regal.woo\n"
	module := regal.parse_module("policy.rego", policy)

	tokens := packages.result with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 0, "length": 7, "line": 0, "modifiers": 0, "type": 3},
		{"col": 14, "length": 3, "line": 0, "modifiers": 0, "type": 0},
	}
}
