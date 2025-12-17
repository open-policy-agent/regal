package regal.result_test

import data.regal.config
import data.regal.result

test_no_related_resources_in_result_fail_on_custom_rule_unless_provided if {
	chain := [
		{"path": ["custom", "regal", "rules", "category", "name", "report"]},
		{
			"annotations": {
				"scope": "package",
				"description": "This is a test",
			},
			"path": ["custom", "regal", "rules", "category", "name"],
		},
	]

	violation := result.fail(chain, {})

	violation == {
		"category": "category",
		"description": "This is a test",
		"level": "error",
		"title": "name",
	}
}

test_related_resources_in_result_fail_on_custom_rule_when_provided if {
	chain := [
		{"path": ["custom", "regal", "rules", "category", "name", "report"]},
		{
			"annotations": {
				"scope": "package",
				"description": "This is a test",
				"related_resources": [{
					"description": "documentation",
					"ref": "https://example.com",
				}],
			},
			"path": ["custom", "regal", "rules", "category", "name"],
		},
	]

	violation := result.fail(chain, {})

	violation == {
		"category": "category",
		"description": "This is a test",
		"level": "error",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://example.com",
		}],
		"title": "name",
	}
}

test_related_resources_generated_by_result_fail_for_builtin_rule if {
	chain := [
		{"path": ["regal", "rules", "category", "name", "report"]},
		{
			"annotations": {
				"scope": "package",
				"description": "This is a test",
			},
			"path": ["regal", "rules", "category", "name"],
		},
	]

	violation := result.fail(chain, {})

	violation == {
		"category": "category",
		"description": "This is a test",
		"level": "error",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/name", "category"),
		}],
		"title": "name",
	}
}

test_aggregate_function_builtin_rule if {
	chain := [
		{"path": ["regal", "rules", "testing", "aggregation", "report"]},
		{
			"annotations": {
				"scope": "package",
				"description": "Testing result.aggregate function",
			},
			"path": ["regal", "rules", "testing", "aggregation"],
		},
	]

	r := result.aggregate(chain, {"foo": "bar", "baz": [1, 2, 3]}) with input as {
		"regal": {"file": {"name": "policy.rego"}},
		"package": {"path": [{"value": "data"}, {"value": "a"}, {"value": "b"}, {"value": "c"}]},
	}
	r == {
		"aggregate_source": {"package_path": ["a", "b", "c"]},
		"aggregate_data": {
			"foo": "bar",
			"baz": [1, 2, 3],
		},
	}
}

test_aggregate_function_custom_rule if {
	chain := [
		{"path": ["custom", "regal", "rules", "testing", "aggregation", "report"]},
		{
			"annotations": {
				"scope": "package",
				"description": "Testing result.aggregate function",
			},
			"path": ["custom", "regal", "rules", "testing", "aggregation"],
		},
	]
	r := result.aggregate(chain, {"foo": "bar", "baz": [1, 2, 3]}) with input as {
		"regal": {"file": {"name": "policy.rego"}},
		"package": {"path": [{"value": "data"}, {"value": "a"}, {"value": "b"}, {"value": "c"}]},
	}
	r == {
		"aggregate_source": {"package_path": ["a", "b", "c"]},
		"aggregate_data": {
			"foo": "bar",
			"baz": [1, 2, 3],
		},
	}
}
