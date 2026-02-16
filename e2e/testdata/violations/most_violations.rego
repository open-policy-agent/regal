# METADATA
# description: |
#   This file is used for e2e tests of most of the provided linter rules. All violations can't
#   be tested in a single file, as some are mutually exclusive (i.e. implicit-future-keywords and
#   rule-named-if).
package all_violations

import rego.v1

# creates a circular import
import data.circular_import

# Imports

# pointless-import
import data.all_violations

# avoid-importing-input
import input

# import-shadows-import
import data.foo
import data.foo

# confusing-alias
import data.foo as bar

import data.redundant.alias as alias

# redundant-data-import
import data

# import-shadows-builtin
import data.http

# prefer-package-imports
# aggregate rule, so will only fail if more than one file is linted
import data.circular_import.x

rule := "here"

import data.after.rule

### Bugs ###

constant_condition if {
	1 == 1
}

duplicate_rule if {
	input.foo
}

duplicate_rule if {
	input.foo
}

leaked_internal_reference := data.foo._bar

# METADATA
# invalid-metadata-attribute: true
should := "fail"

not_equals_in_loop if {
	"foo" != input.bar[_]
}

sprintf_arguments_mismatch := sprintf("%v", [1, 2])

# rule-shadows-builtin
abs := true

top_level_iteration := input[_]

unassigned_return_value if indexof("foo", "o")

zero_arity_function() := true

inconsistent_args(a, b) if {
	a == b
}

inconsistent_args(b, a) if {
	b == a
}

if_empty_object if {}

redundant_existence_check if {
	input.foo
	startswith(input.foo, "bar")
}

redundant_loop_count if {
	count(input.coll) > 0
	some x in input.coll
}

deprecated_builtin := all([true])

partial contains "foo"

impossible_not if {
	not partial
}

time_now_ns_twice := time.now_ns() - now if {
	now := time.now_ns()
	numbers.range(1, 10)
}

argument_always_wildcard(_) if true

argument_always_wildcard(_) if true

# title: annotation without metadata
some_rule := true

var_shadows_builtin if http := true

unused_output_variable if {
	some x
	input[x]
}

default rule_assigns_default := false

rule_assigns_default := false if {
	input.yes
}

### Idiomatic ###

custom_has_key_construct(map, key) if {
	_ = map[key]
}

custom_in_construct(coll, item) if {
	item == coll[_]
}

use_some_for_output_vars if {
	input.foo[output_var]
}

non_raw_regex_pattern := regex.match("[0-9]", "1")

use_in_operator if {
	"item" == input.coll[_]
}

in_wildcard_key if {
	some _, val in input.coll
}

prefer_set_or_object_rule := {x | some x in input; x == "violation"}

equals_pattern_matching(x) := x == "x"

boolean_assignment := 1 < input.two

prefer_equals_comparison if input.enabled = true

# METADATA
# description: ambiguous scope
incremental if input.x

incremental if input.y

prefer_strings_count := count(indexof_n("foobarbaz", "a"))

use_object_keys := {k | some k; input.object[k]}

use_array_flatten if array.concat([1], array.concat([2], [3]))

use_object_union_n if object.union(a, object.union(b, c))

### Style ###

# avoid-get-and-list-prefix
get_foo(foo) := foo

# METADATA
# description: detached-metadata

annotation := "detached"

external_reference(_) if {
	data.foo
}

function_arg_return if {
	indexof("foo", "o", i)
	i == 1
}

line_length := "should be no longer than 120 characters but this line is really long and will exceed that limit which is kinda bad"

#no-whitespace-comment

opa_fmt := "fail"

preferSnakeCase := "fail"

# todo-comment

x := y if {
	y := 1
}

use_assignment = "oparator"

rule_length if {
	input.x1
	input.x2
	input.x3
	input.x4
	input.x5
	input.x6
	input.x7
	input.x8
	input.x9
	input.x10
	input.x11
	input.x12
	input.x13
	input.x14
	input.x15
	input.x16
	input.x17
	input.x18
	input.x19
	input.x20
	input.x21
	input.x22
	input.x23
	input.x24
	input.x25
	input.x26
	input.x27
	input.x28
	input.x29
	input.x30
	input.x31
}

default_over_else := 1 if {
	input.x
} else := 3

default_over_not := input.foo

default_over_not := "foo" if not input.foo

unnecessary_some if {
	some "x" in ["x"]
}

yoda_condition if {
	"foo" == input.bar
}

comprehension_term_assignment := [x |
	some y in input
	x := y.x
]

pointless_reassignment := yoda_condition

mixed_iteration if some x in input[_]

### Testing ###

# this will also trigger the test-outside-test-package rule
test_identically_named_tests := true

test_identically_named_tests := false

todo_test_bad if {
	input.bad
}

print_or_trace_call if {
	print("forbidden!")
}

abc = true

# metasyntactic variable
foo := "bar"

# messy rule
abc if false

# trailing default rule
default abc := true

# dubious print sprintf
y if {
	print(sprintf("name is: %s domain is: %s", [input.name, input.domain]))
}

# double negation
not_fine := true

fine if not not_fine

# rule name repeats package name
all_violations := true

### Performance

with_outside_test if {
	foo with input as {}
}

defer_assignment if {
	x := 1
	input.foo == "bar"
	x == 1
}

walk_no_path if {
	walk(input, [path, value])
	value == "violation"
}

non_loop_expression if {
	some role in input.roles
	endswith(user.email, "example.com")
	role == "admin"
}
