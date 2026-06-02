package regal.aggregators_test

import data.regal.aggregators

test_aggregate_collects_imports_with_location if {
	r := aggregators.imports with input as regal.parse_module("p.rego", `
	package a

	import data.b
	import data.c.d`)

	r == [
		[["b"], "4:2:4:8"],
		[["c", "d"], "5:2:5:8"],
	]
}

test_aggregate_ignores_imports_of_regal_in_custom_rule if {
	r := aggregators.imports with input as regal.parse_module("p.rego", `
	package custom.regal.rules.foo.bar

	import data.regal.ast

	import data.a.b.c
	`)

	r == [[["a", "b", "c"], "6:2:6:8"]]
}
