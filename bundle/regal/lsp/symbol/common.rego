package regal.lsp.symbol

import rego.v2

import data.regal.ast

# METADATA
# description: |
#   Returns the kind based on the type of the rule (function, constant, etc.)
# scope: document
default rule_kind(_) := 13 # kind.variable but can't use ref in default rule

rule_kind(rule) := kind.function if {
	rule.head.args
} else := kind.constant if ast.is_constant_rule(rule)

# METADATA
# description: |
#   Returns the detail string for the given rule, including its type and return type.
# scope: document
default rule_detail(_) := ""

rule_detail(rule) := $`{_rule_type(rule)} ({return_type})` if {
	not rule.head.key
	rule.head.value
	return_type := _rule_return_type(rule.head)
	return_type != ""
} else := _rule_type(rule)

default _rule_type(_) := ""

_rule_type(rule) := "multi-value rule" if rule.head.key and not rule.head.value
_rule_type(rule) := "single-value rule" if not rule.head.args and not rule.head.key
_rule_type(rule) := "single-value map rule" if rule.head.key and rule.head.value
_rule_type(rule) := $"function({concat(", ", _format_args(rule.head.args))})" if rule.head.args

_format_args(args) := [_format_value(term) | some term in args]

_format_value(term) := "_" if {
	term.type == "var" and startswith(term.value, "$")
} else := $`"{term.value}"` if {
	term.type == "string"
} else := term.value

_rule_return_type(head) := type if {
	head.value == true or head.value.type == "boolean"
	type := {
		false: "boolean",
		true: "test",
	}[_is_test(head.ref)]
} else := _simplify_type(builtin.decl.result.type) if {
	head.value.type == "call"

	name := ast.ref_static_to_string(head.value.value[0].value)
	builtin := data.workspace.builtins[name]
} else := "string" if {
	head.value.type == "templatestring"
} else := trim_suffix(head.value.type, "comprehension") if {
	head.value.type in {
		"number",
		"string",
		"null",
		"array",
		"object",
		"set",
		"arraycomprehension",
		"objectcomprehension",
		"setcomprehension",
	}
} else := ""

default _is_test(_) := false

_is_test(head_ref) if startswith(head_ref[_].value, "test_")

_simplify_type(name) := simplified if {
	cutpos := [i |
		some i in [indexof(name, "<"), indexof(name, "[")]
		i != -1
	]
	simplified := substring(name, 0, min(cutpos))
} else := name
