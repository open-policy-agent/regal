package regal.rules.style["line-length_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.style["line-length"] as rule

test_fail_line_too_long if {
	r := rule.report with input as ast.with_rego_v1(`allow if {
foo == bar; bar == baz; [a, b, c, d, e, f] := [1, 2, 3, 4, 5, 6]; qux := [q | some q in input.nonsense]
	}`)
		with config.rules as {"style": {"line-length": {"max-line-length": 80}}}

	r == {{
		"category": "style",
		"description": "Line too long",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/line-length",
		}],
		"title": "line-length",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 6,
			"text": `foo == bar; bar == baz; [a, b, c, d, e, f] := [1, 2, 3, 4, 5, 6]; qux := [q | some q in input.nonsense]`,
			"end": {
				"col": 103,
				"row": 6,
			},
		},
		"level": "error",
	}}
}

test_success_line_too_long_but_non_breakable_word if {
	r := rule.report with input as ast.policy(`

	# Long url: https://www.example.com/this/is/a/very/long/url/that/cannot/be/shortened
	allow := true
	`)
		with config.rules as {"style": {"line-length": {"max-line-length": 40, "non-breakable-word-threshold": 50}}}

	r == set()
}

test_fail_line_too_long_but_below_breakable_word_threshold if {
	r := rule.report with input as ast.with_rego_v1(`

	# Long url: https://www.example.com/this/is/a/very/long
	allow := true
	`)
		with config.rules as {"style": {"line-length": {"max-line-length": 40, "non-breakable-word-threshold": 60}}}

	r == {{
		"category": "style",
		"description": "Line too long",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 7,
			"text": "\t# Long url: https://www.example.com/this/is/a/very/long",
			"end": {
				"col": 56,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/line-length",
		}],
		"title": "line-length",
	}}
}

test_fail_line_exceeds_120_characters_even_if_not_in_config if {
	# regal ignore:line-length
	r := rule.report with input as ast.with_rego_v1(`# Long url: https://www.example.com/this/is/a/very/long/url/that/cannot/be/shortened/and/should/trigger/an/error/anyway/so/that/it/can/be/shortened
	allow := true
	`)

	r == {{
		"category": "style",
		"description": "Line too long",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			# regal ignore:line-length
			"text": "# Long url: https://www.example.com/this/is/a/very/long/url/that/cannot/be/shortened/and/should/trigger/an/error/anyway/so/that/it/can/be/shortened",
			"end": {
				"col": 147,
				"row": 5,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/line-length",
		}],
		"title": "line-length",
	}}
}

test_success_line_not_too_long if {
	r := rule.report with input as ast.policy(`allow if { "foo" == "bar" }`)
		with config.rules as {"style": {"line-length": {"max-line-length": 80}}}

	r == set()
}
