package regal.rules.idiomatic["no-defined-entrypoint_test"]

import data.regal.rules.idiomatic["no-defined-entrypoint"] as rule

test_aggregate_entrypoints if {
	module := regal.parse_module("policy.rego", `
# METADATA
# entrypoint: true
package p

# METADATA
# entrypoint: true
allow := false`)

	aggregate := rule.aggregate with input as module
	aggregate == {
		{"entrypoint": {
			"col": 1,
			"row": 2,
			"end": {
				"col": 19,
				"row": 3,
			},
			"text": "# METADATA\n# entrypoint: true",
		}},
		{"entrypoint": {
			"col": 1,
			"row": 6,
			"end": {
				"col": 19,
				"row": 7,
			},
			"text": "# METADATA\n# entrypoint: true",
		}},
	}
}

test_fail_no_entrypoint_defined if {
	r := rule.aggregate_report with input as {"aggregate": set()}
	r == {{
		"category": "idiomatic",
		"description": "Missing entrypoint annotation",
		"level": "error",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/no-defined-entrypoint",
		}],
		"title": "no-defined-entrypoint",
	}}
}

test_success_single_entrypoint_defined if {
	a := {"p.rego": {"idiomatic/no-defined-entrypoint": {{"entrypoint": {"col": 1, "file": "policy.rego", "row": 2}}}}}
	r := rule.aggregate_report with input.aggregates_internal as a

	r == set()
}

test_success_multiple_entrypoints_defined if {
	r := rule.aggregate_report with input.aggregates_internal as {"p.rego": {"idiomatic/no-defined-entrypoint": [
		{"entrypoint": {"col": 1, "file": "policy.rego", "row": 2}},
		{"entrypoint": {"col": 1, "file": "policy.rego", "row": 6}},
	]}}

	r == set()
}

# Test the fallback pattern where aggregates come from data.workspace.aggregates
# instead of input.aggregates_internal - expects success (no violations)
test_success_entrypoint_from_data_workspace_aggregates if {
	# Using with data.workspace.aggregates instead of with input.aggregates_internal
	r := rule.aggregate_report with data.workspace.aggregates as {"p.rego": {"idiomatic/no-defined-entrypoint": [
		{"entrypoint": {"col": 1, "file": "policy.rego", "row": 2}},
		{"entrypoint": {"col": 1, "file": "policy.rego", "row": 6}},
	]}}

	r == set()
}

# Test the fallback actually works - expects failure (violation) when no entrypoint is defined
test_fail_no_entrypoint_with_data_workspace if {
	# Explicitly ensure input.aggregates_internal is undefined by using an empty input
	# No entrypoint defined, so should fail
	r := rule.aggregate_report with input as {}
		with data.workspace.aggregates as {"p.rego": {"idiomatic/no-defined-entrypoint": []}}

	r == {{
		"category": "idiomatic",
		"description": "Missing entrypoint annotation",
		"level": "error",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/no-defined-entrypoint",
		}],
		"title": "no-defined-entrypoint",
	}}
}
