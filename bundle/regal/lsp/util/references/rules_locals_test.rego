package regal.lsp.util.references_test

import data.regal.lsp.util.references

test_rule_local_name_hides_the_rule_only_in_its_own_scope[scope] if {
	some scope, policy in {
		"comprehension": `package p

allow if input.a

r if {
	allow
	_ = [allow | allow := input.xs[_]]
}`,
		"else body": `package p

allow if input.a

r := 1 if {
	allow
} else := 2 if {
	allow := input.b
	allow
}`,
		"every body": `package p

allow if input.a

r if {
	allow
	every allow in input.xs {
		allow == 1
	}
}`,
	}

	refs := references.refs
		with input as {"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 2, "character": 1},
			"context": {"includeDeclaration": true},
		}}
		with input.regal.file.lines as split(policy, "\n")
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 1}, "end": {"line": 5, "character": 6}},
			"kind": references.kind_read,
		},
	}
}
