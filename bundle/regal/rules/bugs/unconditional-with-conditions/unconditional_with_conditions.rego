# METADATA
# description: Rule defined unconditionally alongside conditional definitions
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/unconditional-with-conditions
package regal.rules.bugs["unconditional-with-conditions"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some definitions in _single_value_definitions

	# at least one definition decides its value based on conditions...
	definitions[true]

	# ...and at least one is defined whatever the input, which leaves those
	# conditions unable to decide the answer: either the values differ and
	# evaluation fails with a conflict, or they agree and the conditional
	# definitions are dead code
	some rule in definitions[false]
	_unconditional(rule)

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

# Definitions of single-value rules, grouped by the name they define and by
# whether the definition carries conditions.
#
# Multi-value rules (`deny contains msg if ...`) are excluded, as their
# definitions union rather than conflict, and an unconditional `deny contains "x"`
# says nothing about whether the other definitions matter.
_single_value_definitions[name][_has_conditions(rule)] contains rule if {
	some rule in input.rules

	# a head with a key and no value is multi-value. Testing for the key alone
	# would be wrong: a ref head like `config.mode := "loose"` also carries a
	# key, which is only the last part of its ref, and it is single-value
	rule.head.value

	not rule.default

	# skip dynamic ref heads, as `p[x] := v` generates a collection
	not _dynamic_ref(rule.head.ref)

	name := ast.ref_to_string(rule.head.ref)
}

# `ast.static_ref` takes a term rather than a bare ref, and a rule head's `ref`
# is the array itself, so the test is inlined from it. Comparing
# `ast.ref_static_to_string(ref)` with `ast.ref_to_string(ref)` does not work as
# a substitute: `_format_term` has no format for a `ref`, `call` or
# `templatestring` part, so both functions drop it and both return `config` for
# `config[input.key]`.
_dynamic_ref(ref) if array.slice(ref, 1, 100)[_].type in {"call", "var", "ref", "templatestring"}

# `body` is omitted from the AST rather than empty on a definition written
# without `if`. The default is what keeps such a definition in the group, rather
# than having the function be undefined for it and the whole entry dropped.
default _has_conditions(_) := false

_has_conditions(rule) if rule.body

# whether a definition without conditions answers for every input
_unconditional(rule) if {
	not rule["else"]

	# a non-constant value can itself be undefined, and the conditional
	# definitions are then not pointless at all: in
	#
	#   allow := input.override
	#   allow := false if not input.override
	#
	# the second definition is what answers when the first is undefined
	ast.is_constant(rule.head.value)

	_matches_any_arguments(rule)
}

# rules that are not functions take no arguments, so nothing further to check
_matches_any_arguments(rule) if not rule.head.args

# a function definition only catches everything when every argument is a
# distinct variable. `f("a") := 1` has no body, but applies only to the argument
# "a", so it leaves the other definitions in the set doing real work. Repeating
# a name — `f(x, x)` — is a constraint too, so the names must be distinct
_matches_any_arguments(rule) if {
	every arg in rule.head.args {
		arg.type == "var"
	}

	count({arg.value | some arg in rule.head.args}) == count(rule.head.args)
}
