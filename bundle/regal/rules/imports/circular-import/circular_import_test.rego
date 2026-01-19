package regal.rules.imports["circular-import_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.imports["circular-import"] as rule

test_aggregate_rule_empty_if_no_refs if {
	aggregate := rule.aggregate with input as ast.policy("allow := true")

	aggregate == set()
}

test_aggregate_rule_empty_if_no_static_refs if {
	aggregate := rule.aggregate with input as ast.policy("allow := data[foo]")

	aggregate == set()
}

test_aggregate_rule_contains_single_self_ref if {
	aggregate := rule.aggregate with input as ast.policy("import data.example")
		with ast.package_name_full as "data.policy"

	aggregate == {{"refs": {"data.example": {"3:8:3:20"}}, "package_name": "data.policy"}}
}

test_aggregate_rule_surfaces_refs if {
	aggregate := rule.aggregate with ast.package_name_full as "data.policy.foo"
		with input as regal.parse_module("example.rego", `
    package policy.foo

    import future.keywords

    import data.foo.bar

    allow := data.baz.qux

    deny contains message if {
      data.config.deny.enabled
      message := "deny"
    }
    `)

	aggregate == {{
		"refs": {
			"data.config.deny.enabled": {"11:7:11:31"},
			"data.foo.bar": {"6:12:6:24"},
			"data.baz.qux": {"8:14:8:26"},
		},
		"package_name": "data.policy.foo",
	}}
}

test_import_graph if {
	r := rule._import_graph with input.aggregates_internal as {
		"a.rego": {"imports/circular-import": {{
			"refs": {"data.policy.b": {"3:12:3:12"}},
			"package_name": "data.policy.a",
		}}},
		"b.rego": {"imports/circular-import": {{
			"refs": {"data.policy.c": {"3:12:3:12"}},
			"package_name": "data.policy.b",
		}}},
		"c.rego": {"imports/circular-import": {{
			"refs": {"data.policy.a": {"3:12:3:12"}},
			"package_name": "data.policy.c",
		}}},
	}

	r == {"data.policy.a": {"data.policy.b"}, "data.policy.b": {"data.policy.c"}, "data.policy.c": {"data.policy.a"}}
}

test_import_graph_self_import if {
	r := rule._import_graph with input.aggregates_internal as {"example.rego": {"imports/circular-import": {{
		"refs": {"data.example": {"4:12:4:12"}},
		"package_name": "data.example",
	}}}}

	r == {"data.example": {"data.example"}}
}

test_groups if {
	r := rule._groups with rule._import_graph as {
		"data.policy.a": {"data.policy.b"},
		"data.policy.b": {"data.policy.c"},
		"data.policy.c": {"data.policy.a"},
		"data.policy.d": {"data.policy.e"},
		"data.policy.e": {"data.policy.f"},
		"data.policy.f": {"data.policy.d"},
		"data.policy.g": {"data.policy.g"},
	}

	r == {
		{"data.policy.a", "data.policy.b", "data.policy.c"},
		{"data.policy.d", "data.policy.e", "data.policy.f"},
		{"data.policy.g"},
	}
}

test_groups_empty_graph if {
	r := rule._groups with rule._import_graph as {"data.policy.a": {}}

	r == set()
}

test_package_locations if {
	r := rule._package_locations with input.aggregates_internal as {
		"a.rego": {"imports/circular-import": {{
			"refs": {"data.policy.b": {"3:12:3:12"}},
			"package_name": "data.policy.a",
		}}},
		"b.rego": {"imports/circular-import": {{
			"refs": {"data.policy.c": {"3:12:3:12"}},
			"package_name": "data.policy.b",
		}}},
		"c.rego": {"imports/circular-import": {{
			"refs": {"data.policy.a": {"3:12:3:12"}},
			"package_name": "data.policy.c",
		}}},
	}

	r == {
		"data.policy.a": {"data.policy.c": ["c.rego", "3:12:3:12"]},
		"data.policy.b": {"data.policy.a": ["a.rego", "3:12:3:12"]},
		"data.policy.c": {"data.policy.b": ["b.rego", "3:12:3:12"]},
	}
}

test_aggregate_report_fails_when_cycle_present if {
	r := rule.aggregate_report with input.aggregates_internal as {
		"a.rego": {"imports/circular-import": {{
			"refs": {"data.policy.b": {"3:12:3:12"}},
			"package_name": "data.policy.a",
		}}},
		"b.rego": {"imports/circular-import": {{
			"refs": {"data.policy.a": {"2:0:2:0"}},
			"package_name": "data.policy.b",
		}}},
	}

	r == {{
		"category": "imports",
		"description": "Circular import detected in: data.policy.a, data.policy.b",
		"level": "error",
		"location": {
			"col": 0,
			"file": "b.rego",
			"row": 2,
			"end": {
				"col": 0,
				"row": 2,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/circular-import", "imports"),
		}],
		"title": "circular-import",
	}}
}

test_aggregate_report_fails_when_cycle_present_in_1_package if {
	r := rule.aggregate_report with input.aggregates_internal as {"a.rego": {"imports/circular-import": {{
		"refs": {"data.policy.a": {"3:12:3:12"}},
		"package_name": "data.policy.a",
	}}}}

	r == {{
		"category": "imports",
		"description": "Circular self-dependency in: data.policy.a",
		"level": "error",
		"location": {
			"col": 12,
			"end": {
				"col": 12,
				"row": 3,
			},
			"file": "a.rego", "row": 3,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/circular-import", "imports"),
		}],
		"title": "circular-import",
	}}
}

test_aggregate_report_fails_when_cycle_present_in_n_packages if {
	r := rule.aggregate_report with input.aggregates_internal as {
		"a.rego": {"imports/circular-import": {{
			"refs": {"data.policy.b": {"3:12:3:12"}},
			"package_name": "data.policy.a",
		}}},
		"b.rego": {"imports/circular-import": {{
			"refs": {"data.policy.c": {"3:12:3:12"}},
			"package_name": "data.policy.b",
		}}},
		"c.rego": {"imports/circular-import": {{
			"refs": {"data.policy.a": {"3:12:3:12"}},
			"package_name": "data.policy.c",
		}}},
	}

	r == {{
		"category": "imports",
		"description": "Circular import detected in: data.policy.a, data.policy.b, data.policy.c",
		"level": "error",
		"location": {
			"col": 12,
			"file": "c.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/circular-import", "imports"),
		}],
		"title": "circular-import",
	}}
}
