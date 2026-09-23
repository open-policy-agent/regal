package regal.rules.idiomatic["prefer-string-interpolation_test"]

import data.regal.ast

import data.regal.rules.idiomatic["prefer-string-interpolation"] as rule

test_sprintf_should_use_interpolation_mixed if {
	r := rule.report with input as ast.policy(`r := sprintf("hello %s %d %v", [x, y, z])`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r := sprintf(\"hello %s %d %v\", [x, y, z])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_sprintf_should_use_interpolation_strings if {
	r := rule.report with input as ast.policy(`r := sprintf("%s-%s", [x, y])`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r := sprintf(\"%s-%s\", [x, y])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_sprintf_simple_var_ref if {
	r := rule.report with input as ast.policy(`r if sprintf("%s", [x])`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r if sprintf(\"%s\", [x])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_sprintf_simple_var_ref_with_same_letter_suffix if {
	r := rule.report with input as ast.policy(`r if sprintf("%ss", [x])`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r if sprintf(\"%ss\", [x])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_sprintf_reference_in_argument_and_config_set_to_include if {
	r := rule.report
		with input as ast.policy(`r if sprintf("%s", [input.foo])`)
		with data.regal.config as {
			"rules": {
				"idiomatic": {
					"prefer-string-interpolation": {
						"include-non-var-args": true,
					},
				},
			},
		}

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r if sprintf(\"%s\", [input.foo])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_sprintf_reference_in_argument_and_config_not_set_to_include if {
	r := rule.report
		with input as ast.policy(`r if sprintf("%s", [input.foo])`)
		with data.regal.config as {
			"rules": {
				"idiomatic": {
					"prefer-string-interpolation": {
						"include-non-var-args": false,
					},
				},
			},
		}

	r == set()
}

test_sprintf_no_interpolation_escaped if {
	r := rule.report with input as ast.policy(`r := sprintf("hello %%s", [x])`)

	r == set()
}

test_sprintf_no_interpolation_float if {
	r := rule.report with input as ast.policy(`r := sprintf("hello %s %f", [x, y])`)

	r == set()
}

test_sprintf_no_interpolation_padding if {
	r := rule.report with input as ast.policy(`r := sprintf("%-*s", [10, x])`)

	r == set()
}

# NOTE(anders):
# we could add non-violation cases ad-nauseam, but since this rule covers only the
# most basic cases (and likely can't be extended to cover many more), we probably
# don't # need to be more exhaustive.

test_concat_should_use_interpolation if {
	r := rule.report with input as ast.policy(`r := concat("", [x, y])`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r := concat(\"\", [x, y])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_concat_reference_in_argument_and_config_set_to_include if {
	r := rule.report
		with input as ast.policy(`r if concat("", [input.x, input.y])`)
		with data.regal.config as {
			"rules": {
				"idiomatic": {
					"prefer-string-interpolation": {
						"include-non-var-args": true,
					},
				},
			},
		}

	r == {{
		"category": "idiomatic",
		"description": "Prefer string interpolation",
		"level": "error",
		"location": {
			"col": 6,
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "r if concat(\"\", [input.x, input.y])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation",
		}],
		"title": "prefer-string-interpolation",
	}}
}

test_concat_reference_in_argument_and_config_not_set_to_include if {
	r := rule.report
		with input as ast.policy(`r if concat("", [input.x, input.y])`)
		with data.regal.config as {
			"rules": {
				"idiomatic": {
					"prefer-string-interpolation": {
						"include-non-var-args": false,
					},
				},
			},
		}

	r == set()
}

test_concat_static_strings_exception if {
	r := rule.report with input as ast.policy(`r := concat("\n", [
		"pretend",
		"these",
		"strings",
		"are",
		"really",
		"long",
	])`)

	r == set()
}
