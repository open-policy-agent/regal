package regal.rules.bugs["unconditional-with-conditions_test"]

import data.regal.ast

import data.regal.rules.bugs["unconditional-with-conditions"] as rule

# The shape of the violation, asserted once in full. The remaining tests assert
# which definition was flagged, which is the part that can actually regress.
test_fail_unconditional_definition_with_conflicting_value if {
	r := rule.report with input as ast.policy(`
	allow := true if input.ok

	allow := false
	`)

	r == {{
		"category": "bugs",
		"description": "Rule defined unconditionally alongside conditional definitions",
		"level": "error",
		"location": {
			"col": 2,
			"end": {
				"col": 16,
				"row": 6,
			},
			"file": "policy.rego",
			"row": 6,
			"text": "\tallow := false",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/unconditional-with-conditions",
		}],
		"title": "unconditional-with-conditions",
	}}
}

# The unconditional definition agrees with the conditional one, so evaluation
# does not fail. The conditional definition is dead code, which is still worth
# reporting.
test_fail_unconditional_definition_with_same_value if {
	r := rule.report with input as ast.policy(`
	deny := true if input.bad

	deny := true
	`)

	_flagged(r) == {"\tdeny := true"}
}

# The function form, which is the one that turns up in practice.
test_fail_unconditional_function_with_wildcard_argument if {
	r := rule.report with input as ast.policy(`
	f(x) if x > 10

	f(_) := false
	`)

	_flagged(r) == {"\tf(_) := false"}
}

# A named argument that the body does not constrain is as unconditional as a
# wildcard, and this is how the shape is usually written.
test_fail_unconditional_function_with_named_argument if {
	r := rule.report with input as ast.policy(`
	f(x) := true if x > 10

	f(x) := false
	`)

	_flagged(r) == {"\tf(x) := false"}
}

test_fail_unconditional_definition_alongside_default if {
	r := rule.report with input as ast.policy(`
	default f(_) := false

	f(x) if x > 10

	f(_) := false
	`)

	_flagged(r) == {"\tf(_) := false"}
}

test_fail_unconditional_definition_before_the_conditional_one if {
	r := rule.report with input as ast.policy(`
	allow := false

	allow := true if input.ok
	`)

	_flagged(r) == {"\tallow := false"}
}

test_fail_unconditional_definition_with_constant_composite_value if {
	r := rule.report with input as ast.policy(`
	config := {"mode": "strict"} if input.strict

	config := {"mode": "loose"}
	`)

	_flagged(r) == {"\tconfig := {\"mode\": \"loose\"}"}
}

test_fail_two_unconditional_definitions_are_both_reported if {
	r := rule.report with input as ast.policy(`
	allow := true if input.ok

	allow := false

	allow := true
	`)

	_flagged(r) == {"\tallow := false", "\tallow := true"}
}

test_fail_ref_head_with_static_string if {
	r := rule.report with input as ast.policy(`
	config.mode := "strict" if input.strict

	config.mode := "loose"
	`)

	_flagged(r) == {"\tconfig.mode := \"loose\""}
}

# A number is as static a ref part as a string is, so these two definitions
# address the same document and conflict in the same way.
test_fail_ref_head_with_static_number if {
	r := rule.report with input as ast.policy(`
	config[1] := "strict" if input.strict

	config[1] := "loose"
	`)

	_flagged(r) == {"\tconfig[1] := \"loose\""}
}

# --- cases that must not be reported ----------------------------------------

# `contains` definitions union rather than conflict, so an unconditional one
# says nothing about whether the others matter.
test_success_multi_value_rule if {
	r := rule.report with input as ast.policy(`
	deny contains "always"

	deny contains msg if {
		msg := "sometimes"
	}
	`)

	r == set()
}

# `f("a")` has no conditions but applies only to that one argument, so the
# other definitions in the set still do the work for every other argument.
test_success_function_with_constant_argument if {
	r := rule.report with input as ast.policy(`
	f("a") := 1

	f(x) := 2 if x == "b"
	`)

	r == set()
}

# Repeating an argument name is a constraint too: this definition applies only
# to calls where both arguments are equal.
test_success_function_with_repeated_argument_name if {
	r := rule.report with input as ast.policy(`
	f(x, x) := 1

	f(x, y) := 2 if x != y
	`)

	r == set()
}

# The definition without conditions can itself be undefined, so the conditional
# one is what answers in that case. Reporting this would be a false positive.
test_success_unconditional_value_is_not_constant if {
	r := rule.report with input as ast.policy(`
	allow := input.override

	allow := false if not input.override
	`)

	r == set()
}

test_success_default_and_one_conditional_definition if {
	r := rule.report with input as ast.policy(`
	default allow := false

	allow := true if input.ok
	`)

	r == set()
}

test_success_only_unconditional_definitions if {
	r := rule.report with input as ast.policy(`
	allow := false
	`)

	r == set()
}

# Two definitions that both lack conditions may well conflict, but no condition
# is made pointless by it, which is what this rule reports. `allow := true`
# beside `allow := false` conflicts on every evaluation and is left to a rule of
# its own; `duplicate-rule` does not catch it either, as it compares rule text.
test_success_two_definitions_without_conditions if {
	r := rule.report with input as ast.policy(`
	allow := true

	allow := false
	`)

	r == set()
}

test_success_definition_without_conditions_beside_a_non_constant_one if {
	r := rule.report with input as ast.policy(`
	allow := input.override

	allow := true
	`)

	r == set()
}

test_success_all_definitions_conditional if {
	r := rule.report with input as ast.policy(`
	allow := true if input.ok

	allow := false if input.blocked
	`)

	r == set()
}

# `else` gives one rule with an ordered chain rather than a set of definitions,
# so the earlier conditions do decide the answer and there is nothing to report.
# It is one of several ways to write a fallback, `default` being another.
test_success_else_fallback if {
	r := rule.report with input as ast.policy(`
	f(x) := 1 if {
		x > 10
	} else := 2
	`)

	r == set()
}

# Whether two dynamic ref heads ever address the same document depends on what
# the key evaluates to, so they are left alone. A variable in the ref, rather
# than a computed key, would be unsafe in a definition without a body, so this
# is the form the exclusion has to cover.
test_success_dynamic_ref_head if {
	r := rule.report with input as ast.policy(`
	config[input.key] := "loose"

	config[input.key] := "strict" if input.strict
	`)

	r == set()
}

test_success_different_rules_do_not_pair_up if {
	r := rule.report with input as ast.policy(`
	allow := true if input.ok

	deny := false
	`)

	r == set()
}

_flagged(report) := {violation.location.text | some violation in report}
