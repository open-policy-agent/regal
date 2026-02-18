# METADATA
# description: Ambiguous metadata scope
package regal.rules.idiomatic["ambiguous-scope"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some name, rules in _incremental_rules

	# don't report on internal rules
	not startswith(name, "_")

	rules_with_annotations := [rule |
		some rule in rules

		rule.annotations
	]

	# a single rule with a single annotation — that means it is also
	# has scope == rule as if it had a scope == document it would be seen
	# on each of the rules
	count(rules_with_annotations) == 1
	count(rules_with_annotations[0].annotations) == 1

	annotation := rules_with_annotations[0].annotations[0]

	# treat this as an exception for now — it's arguably an issue in itself
	# that an entrypoint can be scoped to "rule", as clearly that's not how
	# entrypoints work
	not annotation.entrypoint

	not _explicit_scope(rules_with_annotations[0], input.regal.file.lines)

	violation := result.fail(rego.metadata.chain(), result.location(annotation))
}

_incremental_rules[name] contains rule if {
	some rule in input.rules

	name := ast.ref_to_string(rule.head.ref)

	util.has_duplicates(_rule_names, name)
}

_rule_names := [ast.ref_to_string(rule.head.ref) | some rule in input.rules]

_explicit_scope(rule, lines) if {
	some i in numbers.range(
		util.to_location_object(rule.annotations[0].location).row - 1,
		util.to_location_object(rule.head.location).row - 2,
	)
	line := lines[i]

	startswith(line, "# scope:")
}
