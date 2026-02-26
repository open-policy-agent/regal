package regal.lsp.semantictokens.vars.packages_test

import data.regal.lsp.semantictokens.vars.packages

test_packages if {
	policy := `package regal.woo`
	tokens := packages.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "1:15:1:18", "type": "string", "value": "woo"} in tokens
}
