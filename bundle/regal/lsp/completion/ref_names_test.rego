package regal.lsp.completion_test

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.lsp.completion

test_ref_names if {
	module := ast.policy(`
	import data.imp
	import data.foo.bar as bb

	x := 1

	allow if {
		some x
		input.foo[x] == data.bar[x]
		startswith("hey", "h")

		imp.foo == data.x
	}`)

	ref_names := completion.ref_names
		with config.capabilities as capabilities.provided
		with input as module

	ref_names == {
		"imp",
		"bb",
		"input.foo",
		"data.bar",
		"imp.foo",
		"data.x",
	}
}
