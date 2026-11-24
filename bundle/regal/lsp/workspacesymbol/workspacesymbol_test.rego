package regal.lsp.workspacesymbol_test

import rego.v2

import data.regal.lsp.workspacesymbol

# Note: this is mostly tested through the document symbol tests

test_workspace_symbols if {
	mod1 := regal.parse_module("p1.rego", "package p1\np := 1")
	mod2 := regal.parse_module("p2.rego", "package p2\nq := 2")

	result := workspacesymbol.result.response
		with data.workspace.parsed as {
			"file:///p1.rego": mod1,
			"file:///p2.rego": mod2,
		}

	count(result) == 4
}
