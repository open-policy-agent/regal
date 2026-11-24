# METADATA
# description: |
#   Provides a list of all symbols in the workspace
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#workspace_symbol
# schemas:
#   - input:              schema.regal.lsp.common
#   - input.params:       {type: object}
#   - input.params.query: {type: string}
package regal.lsp.workspacesymbol

import rego.v2

import data.regal.ast
import data.regal.lsp.symbol
import data.regal.lsp.util.range

# Note: currently ignoring params.Query, as the client seems to do a good
# job of filtering anyway, and that would merely be an optimization here.
# But perhaps a good one to do at some point, and I'm not sure all clients
# do this filtering.

# METADATA
# description: |
#   Returns the list of symbols in the workspace
# entrypoint: true
result.response contains package_symbol if {
	some url, module in data.workspace.parsed

	package_symbol := {
		"name": ast.ref_to_string(module.package.path),
		"kind": symbol.kind.package,
		"location": {
			"uri": url,
			"range": {
				"start": {
					"line": 0,
					"character": 0,
				},
				"end": range.parse(regal.last(module.rules).location).end,
			},
		},
	}
}

result.response contains rule_symbol if {
	some url, module in data.workspace.parsed

	pkg_path := ast.ref_to_string(module.package.path)

	by_name := {name: rules |
		some rule in module.rules
		name := ast.ref_static_to_string(rule.head.ref)
		rules := [rule |
			some rule in module.rules
			ast.ref_static_to_string(rule.head.ref) == name
		]
	}

	some name, rules in by_name
	first_rule := rules[0]

	rule_symbol := {
		"name": $"{pkg_path}.{ast.ref_static_to_string(first_rule.head.ref)}",
		"kind": symbol.rule_kind(first_rule),
		"location": {
			"uri": url,
			"range": {
				"start": range.parse(first_rule.location).start,
				"end": range.parse(regal.last(rules).location).end,
			},
		},
	}
}
