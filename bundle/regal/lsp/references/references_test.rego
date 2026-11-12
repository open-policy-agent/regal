package regal.lsp.references_test

import data.regal.lsp.references
import data.regal.lsp.testing

test_references_response_is_the_symbol_references if {
	[tdp, module] := testing.at_cursor("allow| if input.a\n\nother if allow")

	refs := references.result.response
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	refs == {{
		"uri": "file:///p.rego",
		"range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 14}},
	}}
}
