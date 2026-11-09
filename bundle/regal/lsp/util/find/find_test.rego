package regal.lsp.util.find_test

import data.regal.ast
import data.regal.lsp.testing
import data.regal.lsp.util.find

test_rule_at_position[tdp.params.position] if {
	policy := `package p

rule1 if {
	true
}

rule2 if {
	false
}

default rule3 := false`
	module := regal.parse_module("p.rego", policy)

	some [tdp, exp_name, exp_index] in [
		[text_document_position(2, 3), "rule1", 0],
		[text_document_position(2, 7), "rule1", 0],
		[text_document_position(6, 0), "rule2", 1],
		[text_document_position(6, 10), "rule2", 1],
		[text_document_position(7, 5), "rule2", 1],
		[text_document_position(10, 9), "rule3", 2],
	]

	[rule, i] := find.rule_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	i == exp_index
	rule.head.ref[0].value == exp_name
}

test_arg_at_position[name] if {
	some name, snippet in {
		"declaration": "fn(x|) := x.y",
		"just after the declaration": "fn(x)| := x.y",
		"use in the head value": "fn(x) := x|.y",
		"use in the body": "fn(x) := y if y := x|.y",
		"use in the else branch": "fn(x) := 1 if false else := x|",
	}

	[tdp, module] := testing.at_cursor(snippet)

	[arg, rule_index] := find.arg_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	arg.value == "x"
	rule_index == 0
}

test_arg_at_position_not_an_arg[name] if {
	some name, snippet in {
		"wildcard": "fn(_|) := 1",
		"second wildcard": "fn(_, _|) := 1",
		"field with the same name as the argument": "fn(x) := input.x|",
		"array argument, not supported": "fn([a|, b]) := a",
	}

	[tdp, module] := testing.at_cursor(snippet)

	not find.arg_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module
}

test_import_at_position[tdp.params.position] if {
	policy := `package p
import data.foo.bar
import data.baz.qux
import data.alpha["be-ta"].gamma`

	module := regal.parse_module("p.rego", policy)

	some [tdp, exp_path] in [
		[text_document_position(1, 10), "data.foo.bar"],
		[text_document_position(2, 10), "data.baz.qux"],
		[text_document_position(3, 15), "data.alpha[\"be-ta\"].gamma"],
	]

	imp := find.import_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	ast.ref_static_to_string(imp.path.value) == exp_path
}

test_import_alias_at_position[name] if {
	some name, snippet in {
		"in the alias": "import data.foo.bar as b|az",
		"just after the alias": "import data.foo.bar as baz |",
	}

	[tdp, module] := testing.at_cursor(snippet)

	imp := find.import_alias_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	imp.alias == "baz"
}

test_import_alias_at_position_not_on_alias[name] if {
	some name, snippet in {
		"in the path": "import data.fo|o.bar as baz",
		"on the as keyword": "import data.foo.bar a|s baz",
	}

	[tdp, module] := testing.at_cursor(snippet)

	not find.import_alias_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module
}

test_assign_var_at_position[name] if {
	some name, snippet in {
		"declaration": `allow if {
	foo| := 1
	foo == 1
}`,
		"use": `allow if {
	foo := 1
	foo| == 1
}`,
		"destructured declaration": `allow if {
	[foo|, bar] := [1, 2]
	foo == bar
}`,
	}

	[tdp, module] := testing.at_cursor(snippet)

	[var, rule_index] := find.assign_var_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module

	var.value == "foo"
	rule_index == 0
}

test_assign_var_at_position_not_an_assign_var[name] if {
	some name, snippet in {
		"input": `allow if input|.user == 1`,
		"some variable": `allow if {
	some x| in input.xs
	x == 1
}`,
		"argument": `fn(x|) := x`,
		"variable that is never assigned": `allow if {
	foo := 1
	input.foo| == 1
}`,
	}

	[tdp, module] := testing.at_cursor(snippet)

	not find.assign_var_at_position
		with input as tdp
		with data.workspace.parsed["file:///p.rego"] as module
}

text_document_position(line, character) := {"params": {
	"textDocument": {"uri": "file:///p.rego"},
	"position": {
		"line": line,
		"character": character,
	},
}}
