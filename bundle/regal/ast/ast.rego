# METADATA
# description: |
#   the 'ast' package provides the base functionality for working
#   with OPA's AST, more recently in the form of RoAST
package regal.ast

import data.regal.config
import data.regal.util

# METADATA
# description: set of Rego's scalar type
scalar_types := {"boolean", "null", "number", "string"}

# METADATA
# description: set containing names of all built-in functions counting as operators
operators := {
	"and",
	"assign",
	"div",
	"eq",
	"equal",
	"gt",
	"gte",
	"internal.member_2",
	"internal.member_3",
	"lt",
	"lte",
	"minus",
	"mul",
	"neq",
	"or",
	"plus",
	"rem",
}

# METADATA
# description: |
#   returns true if provided term is either a scalar or a collection of ground values
# scope: document
is_constant(term) if term.type in scalar_types

is_constant(term) if {
	term.type in {"array", "object", "set"}
	not has_term_var(term.value)
}

# METADATA
# description: true if provided term represents a wildcard (`_`) variable
is_wildcard(term) if {
	term.type == "var"
	startswith(term.value, "$")
}

default builtin_names := set()

# METADATA
# description: set containing the name of all built-in functions (given the active capabilities)
# scope: document
builtin_names := object.keys(config.capabilities.builtins)

# METADATA
# description: |
#   set containing the namespaces of all built-in functions (given the active capabilities),
#   like "http" in `http.send` or "sum" in `sum``
builtin_namespaces contains regex.replace(name, `\..*`, "") if some name in builtin_names

# METADATA
# description: |
#   provides the package path values (strings) as an array starting _from_ "data":
#   package foo.bar -> ["foo", "bar"]
package_path := [path.value |
	some i, path in input.package.path
	i > 0
]

# METADATA
# description: |
#   provide the package name / path as originally declared in the
#   input policy, so "package foo.bar" would return "foo.bar"
package_name := concat(".", package_path)

# METADATA
# description: provides the full package name including the "data." prefix,
package_name_full := concat("", ["data.", package_name])

# METADATA
# description: provides all static string values from ref
named_refs(ref) := [term |
	some i, term in ref
	_is_name(term.type, i)
]

_is_name("var", 0)
_is_name("string", pos) if pos > 0

# METADATA
# description: all the rules (excluding functions) in the input AST
rules := [rule |
	some rule in input.rules

	not rule.head.args
]

# METADATA
# description: all the test rules in the input AST
tests := [rule |
	some rule in rules

	startswith(rule.head.ref[0].value, "test_")
]

# METADATA
# description: all the functions declared in the input AST
functions := [rule |
	some rule in input.rules
	rule.head.args
]

# METADATA
# description: |
#   all rules and functions in the input AST not denoted as private, i.e. excluding
#   any rule/function with a `_` prefix. it's not unthinkable that more ways to denote
#   private rules (or even packages), so using this rule should be preferred over
#   manually checking for this using the rule ref
public_rules_and_functions := [rule |
	some rule in input.rules

	count([part |
		some i, part in rule.head.ref

		_private_rule(i, part)
	]) == 0
]

_private_rule(0, part) if startswith(part.value, "_")

_private_rule(i, part) if {
	i > 0
	part.type == "string"
	startswith(part.value, "_")
}

# METADATA
# description: a list of the argument names for the given rule (if function)
function_arg_names(rule) := [arg.value | some arg in rule.head.args]

# METADATA
# description: all the rule and function names in the input AST
rule_and_function_names contains ref_static_to_string(rule.head.ref) if some rule in input.rules

# METADATA
# description: all identifiers in the input AST (rule and function names, plus imported names)
identifiers := rule_and_function_names | imported_identifiers

# METADATA
# description: all rule names in the input AST (excluding functions)
rule_names contains ref_static_to_string(rule.head.ref) if some rule in rules

# METADATA
# description: |
#   determine if var in var (e.g. `x` in `input[x]`) is used as input or output
#   input = variable value set elsewhere in the policy
#   output =  variable value set in this location (unification)
# scope: document
is_output_var(rule, var) if {
	# test the cheap and common case first, and 'else' only when it's not
	is_wildcard(var)
} else if {
	not var.value in (rule_names | imported_identifiers)

	num_above := count([1 |
		some above in find_vars_in_local_scope(rule, var.location)
		above.value == var.value
	])
	num_some := count([1 |
		some name in find_some_decl_names_in_scope(rule, var.location)
		name == var.value
	])

	# only the first ref variable in scope can be an output! meaning that:
	# allow if {
	#     some x
	#     input[x]    # <--- output
	#     data.bar[x] # <--- input
	# }
	num_above - num_some == 0
}

# METADATA
# description: |
#   returns an array of all rule indices, as strings. this will be needed until
#   https://github.com/open-policy-agent/opa/issues/6736 is fixed
rule_index_strings := [sprintf("%d", [i]) | some i, _ in _rules]

# METADATA
# description: |
#   a map containing all function calls (built-in and custom) in the input AST
#   keyed by rule index
function_calls[rule_index] contains call if {
	some rule_index in rule_index_strings
	some ref in found.calls[rule_index]

	name := ref_to_string(ref[0].value)
	args := [arg |
		some i, arg in array.slice(ref, 1, 100)

		not _exclude_arg(name, i, arg.type)
	]

	call := {
		"name": name,
		"location": ref[0].location,
		"args": args,
	}
}

# exclude arg if:
# - first "arg" of assign is the variable to assign to
# - call - covered elsewhere
_exclude_arg(_, _, "call")
_exclude_arg("assign", 0, _)

# METADATA
# description: |
#   true if both ref values (or "paths") are equal in type
#   and value for each path component, ignoring locations
ref_value_equal(v1, v2) if {
	count(v1) == count(v2)

	every i, part in v1 {
		part.type == v2[i].type
		part.value == v2[i].value
	}
}

# METADATA
# description: |
#   returns a new ref value made by extending terms1 with terms2,
#   where the first term of terms2 transformed to string
extend_ref_terms(terms1, terms2) := array.concat(
	terms1,
	array.concat(
		[object.union(terms2[0], {"type": "string"})],
		array.slice(terms2, 1, 100),
	),
)

# METADATA
# description: |
#   true if all terms in `terms1` are also present in `terms2`
#   regardless of length, and ignoring locations
is_terms_subset(terms1, terms2) if {
	count(terms1) <= count(terms2)

	every i, term in terms1 {
		term.type == terms2[i].type
		term.value == terms2[i].value
	}
}

# METADATA
# description: returns the "path" string of any given ref value
ref_to_string(ref) := ref[0].value if {
	count(ref) == 1
} else := concat("", [_ref_part_to_string(i, part) | some i, part in ref])

_ref_part_to_string(0, part) := part.value
_ref_part_to_string(i, part) := _format_part(part) if i > 0

_format_part(part) := concat("", [".", part.value]) if {
	part.type == "string"
	regex.match(`^[a-zA-Z_][a-zA-Z1-9_]*$`, part.value)
} else := sprintf(`["%v"]`, [part.value]) if {
	part.type == "string"
} else := sprintf(`[%v]`, [part.value]) if {
	# for now, only allow printing the static parts of refs containing
	# template strings, as we don't likely want to rebuild the template string
	# from its parts here. if we need the string representation anywhere, we
	# can revisit this later, and perhaps simply grab the original text from its
	# location.
	part.type != "templatestring"
}

# METADATA
# description: |
#   returns the string representation of a ref up until its first
#   non-static (i.e. variable) value, if any:
#   foo.bar -> foo.bar
#   foo.bar[baz] -> foo.bar
ref_static_to_string(ref) := ref_to_string(array.slice(ref, 0, first_non_static)) if {
	first_non_static := [i |
		some i, part in ref
		i > 0
		part.type in {"call", "var", "ref", "templatestring"}
	][0]
} else := ref_to_string(ref)

# METADATA
# description: true if ref contains only static parts
static_ref(ref) if not _non_static_ref(ref)

# optimized inverse of static_ref benefitting from early exit
# 128 is used only as a reasonable (well...) upper limit for a ref, but the
# slice will be capped at the length of the ref anyway (avoids count)
# regal ignore:narrow-argument
_non_static_ref(ref) if array.slice(ref.value, 1, 128)[_].type in {"call", "var", "ref", "templatestring"}

# METADATA
# description: provides a set of names of all built-in functions called in the input policy
builtin_functions_called contains name if {
	name := function_calls[_][_].name
	name in builtin_names
}

# METADATA
# description: |
#   Returns custom functions declared in input policy in the same format as builtin capabilities
function_decls[name] := {"decl": {"args": [{"type": "any"} | head.args[_]], "result": {"type": "any"}}} if {
	head := functions[_].head
	name := ref_to_string(head.ref)
}

# METADATA
# description: returns the args for function past the expected number of args
function_ret_args(fn_name, terms) := array.slice(terms, count(all_functions[fn_name].decl.args) + 1, 100)

# METADATA
# description: true if last argument of function is a return assignment
function_ret_in_args(fn_name, terms) if {
	# special case: print does not have a last argument as it's variadic
	fn_name != "print"

	rest := array.slice(terms, 1, 100)

	# for now, bail out of nested calls
	not "call" in {term.type | some term in rest}

	count(rest) > count(all_functions[fn_name].decl.args)
}

# METADATA
# description: answers if provided rule is implicitly assigned boolean true, i.e. allow if { .. } or not
# custom:
#   deprecated: |
#     this is now handled by the roast package, and there's little
#     point in using this over a `rule.head.value.location` check directly
# scope: document
implicit_boolean_assignment(rule) if not rule.head.value.location

# METADATA
# description: |
#   object containing all available built-in and custom functions in the
#   scope of the input AST, keyed by function name
all_functions := object.union(config.capabilities.builtins, function_decls)

# METADATA
# description: |
#   set containing all available built-in and custom function names in the
#   scope of the input AST
all_function_names := object.keys(all_functions)

# METADATA
# description: set containing the namespaces of all functions, built-in and custom
all_function_namespaces := builtin_namespaces | custom_function_namespaces

# METADATA
# description: set containing the namespaces of all custom functions
custom_function_namespaces contains regex.replace(name, `\..*`, "") if some name in object.keys(function_decls)

# METADATA
# description: |
#   true if rule head contains no identifier, but is a chained rule body immediately following the previous one:
#   foo {
#       input.bar
#   } {	# <-- chained rule body
#       input.baz
#   }
is_chained_rule_body(rule, lines) if {
	head_loc := util.to_location_object(rule.head.location)

	row_text := lines[head_loc.row - 1]
	col_text := substring(row_text, head_loc.col - 1, -1)

	startswith(col_text, "{")
}

# METADATA
# description: answers whether variable of `name` is found anywhere in provided rule `head`
# scope: document
var_in_head(head, name) if {
	has_named_var(head.value, name)
} else if {
	has_named_var(head.key, name)
} else if {
	some i, part in head.ref
	i > 0
	has_named_var(part, name)
}

# METADATA
# description: |
#   true if var of `name` is referenced in any `calls` (likely,
#   `ast.function_calls`) in the rule of given `rule_index`
# scope: document
var_in_call(calls, rule_index, name) if has_named_var(calls[rule_index][_].args[_], name)

# METADATA
# description: answers wether provided expression is an assignment (using `:=`)
is_assignment(term) if {
	term.type == "ref"
	term.value[0].type == "var"
	term.value[0].value == "assign"
}

# METADATA
# description: returns the terms in an assignment (`:=`) expression, or undefined if not assignment
assignment_terms(terms) := [terms[1], terms[2]] if is_assignment(terms[0])

# METADATA
# description: |
#   For a given rule head name, this rule contains a list of locations where
#   there is a rule head with that name.
rule_head_locations[name] contains {"row": loc.row, "col": loc.col} if {
	some rule in input.rules

	name := concat(".", ["data", package_name, ref_static_to_string(rule.head.ref)])
	loc := util.to_location_object(rule.head.location)
}
