# METADATA
# description: |
#   Provides a hierarchical view of all (supported/relevant) symbols in the requested document.
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_documentSymbol
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.documentsymbol

import rego.v2

import data.regal.ast
import data.regal.lsp.symbol
import data.regal.lsp.util.range

# METADATA
# description: |
#   Return the symbol hierarchy for the requested document.
# entrypoint: true
default result["response"] := null

result["response"] := [_symbol] if _module

_symbol["name"] := ast.ref_to_string(_module.package.path)
_symbol["kind"] := symbol.kind.package
_symbol["range"] := {
	"start": {
		"line": 0,
		"character": 0,
	},
	"end": {
		"line": count(input.regal.file.lines) - 1,
		"character": count(regal.last(input.regal.file.lines)),
	},
}

_symbol["selectionRange"] := _symbol.range # regal ignore:prefer-snake-case

_symbol["children"] := [sym | some sym in _rule_symbols] if _rule_symbols != set()

_rule_symbols contains rule_symbol if {
	some name, rules in ast.rules_by_name

	count(rules) == 1

	some rule in rules

	rule_range := range.parse(rule.location)

	rule_symbol := {
		"name": name,
		"kind": symbol.rule_kind(rule),
		"range": rule_range,
		"selectionRange": rule_range,
		"detail": symbol.rule_detail(rule),
	}
}

_rule_symbols contains rule_symbol if {
	some name, rules in ast.rules_by_name
	count(rules) > 1

	sorted := sort([[rule_range.start.line, rule, rule_range] |
		some rule in rules
		rule_range := range.parse(rule.location)
	])
	group_range := {"start": sorted[0][2].start, "end": regal.last(sorted)[2].end}
	first_rule := sorted[0][1]
	kind := symbol.rule_kind(first_rule)

	rule_symbol := {
		"name": name,
		"kind": kind,
		"range": group_range,
		"selectionRange": group_range,
		"detail": symbol.rule_detail(first_rule),
		"children": [child_symbol |
			some i, [_, rule, rule_range] in sorted
			child_symbol := {
				"name": $`#{i + 1}`,
				"kind": kind,
				"range": rule_range,
				"selectionRange": rule_range,
				"detail": symbol.rule_detail(rule),
			}
		],
	}
}

_module := data.workspace.parsed[input.params.textDocument.uri]
