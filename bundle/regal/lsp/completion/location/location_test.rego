package regal.lsp.completion.location_test

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.lsp.completion.location

test_find_rule_from_location if {
	policy := `package p

import rego.v1

rule1 if {
	x := 1
}

rule2 if {
	y := 2
}

rule3 if {
	z := 3
}
`
	lines := split(policy, "\n")

	module := regal.parse_module("p.rego", policy)

	not location.find_rule(module.rules, 2) with input.regal.file.lines as lines

	r1 := location.find_rule(module.rules, 5) with input.regal.file.lines as lines

	ast.ref_to_string(r1.head.ref) == "rule1"

	r2 := location.find_rule(module.rules, 9) with input.regal.file.lines as lines
	ast.ref_to_string(r2.head.ref) == "rule2"

	r3 := location.find_rule(module.rules, 15) with input.regal.file.lines as lines
	ast.ref_to_string(r3.head.ref) == "rule3"
}

test_find_locals_at_location[loc] if {
	policy := `package p

import rego.v1

rule if {
	x := 1
}

function(a, b) if {
	c := 3
}

another if {
	some x, y in collection
	z := x + y
}
`
	module := regal.parse_module("p.rego", policy)
	lines := split(policy, "\n")

	some [loc, want] in {
		[{"row": 6, "col": 1}, set()],
		[{"row": 6, "col": 10}, {"x"}],
		[{"row": 10, "col": 1}, {"a", "b"}],
		[{"row": 10, "col": 6}, {"a", "b", "c"}],
		[{"row": 15, "col": 1}, {"x", "y"}],
		[{"row": 16, "col": 1}, {"x", "y", "z"}],
	}

	r := location.find_locals(module.rules, loc) with input as module
		with input.regal.file.lines as lines
		with config.capabilities as capabilities.provided

	r == want
}

test_word_at if {
	line := "foo bar here baz qux"
	word := location.word_at(line, 12) # col after 'r' in 'here'

	word == {
		"text_before": "foo bar ",
		"text_after": " baz qux",
		"offset_before": 3,
		"offset_after": 1,
		"text": "here",
	}
}

test_word_with_number_at if {
	line := "foo bar here2 baz qux"
	word := location.word_at(line, 12) # col after 'r' in 'here2'

	word == {
		"text_before": "foo bar ",
		"text_after": " baz qux",
		"offset_before": 3,
		"offset_after": 2,
		"text": "here2",
	}
}

test_ref_at if {
	ref := location.ref_at("allow := neo4j.q", 17) # at end of 'neo4j.q'

	ref == {
		"offset_before": 7,
		"offset_after": 0,
		"text": "neo4j.q",
	}
}
