package regal.rules.style["no-whitespace-comment_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.style["no-whitespace-comment"] as rule

test_fail_no_leading_whitespace if {
	r := rule.report with input as ast.policy(`#foo`)

	r == {{
		"category": "style",
		"description": "Comment should start with whitespace",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/no-whitespace-comment", "style"),
		}],
		"title": "no-whitespace-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 5,
				"row": 3,
			},
			"text": "#foo",
		},
		"level": "error",
	}}
}

test_fail_comments_after_shebang if {
	# Adding a shebang should not excempt any other comments from linting.

	r := rule.report with input as regal.parse_module("policy.rego", "#!/usr/bin/env foo\n\npackage policy\n\n#foo")

	r == {{
		"category": "style",
		"description": "Comment should start with whitespace",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/no-whitespace-comment", "style"),
		}],
		"title": "no-whitespace-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 5,
				"row": 5,
			},
			"text": "#foo",
		},
		"level": "error",
	}}
}

test_fail_shebang_in_body if {
	# Shebang-style comments not on the first line of the package should not be
	# exempted from linting.

	r := rule.report with input as regal.parse_module("policy.rego", "package policy\n\n#!/usr/bin/env foo")

	r == {{
		"category": "style",
		"description": "Comment should start with whitespace",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/no-whitespace-comment", "style"),
		}],
		"title": "no-whitespace-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 19,
				"row": 3,
			},
			"text": "#!/usr/bin/env foo",
		},
		"level": "error",
	}}
}

test_fail_no_leading_whitespace_multiple_hashes if {
	r := rule.report with input as ast.policy(`##foo`)

	r == {{
		"category": "style",
		"description": "Comment should start with whitespace",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/no-whitespace-comment", "style"),
		}],
		"title": "no-whitespace-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 6,
				"row": 3,
			},
			"text": "##foo",
		},
		"level": "error",
	}}
}

test_success_excepted_pattern if {
	r := rule.report with input as ast.policy(`#-- foo`)
		with config.rules as {"style": {"no-whitespace-comment": {"except-pattern": "^--"}}}

	r == set()
}

test_success_leading_whitespace if {
	r := rule.report with input as ast.policy(`# foo`)

	r == set()
}

test_success_leading_whitespace_double_hash if {
	r := rule.report with input as ast.policy(`## foo`)

	r == set()
}

test_success_lonely_hash if {
	r := rule.report with input as ast.policy(`#`)

	r == set()
}

test_success_comment_with_newline if {
	r := rule.report with input as ast.policy(`
	# foo
	#
	# bar`)

	r == set()
}

test_success_multiple_hash_comment if {
	r := rule.report with input as ast.policy(`
	##########
	# Foobar #
	##########`)

	r == set()
}

test_success_no_leading_whitespace_shebang if {
	# shebangs should be exempted from the leading whitespace lint.

	r := rule.report with input as regal.parse_module("policy.rego", "#!/usr/bin/env foo\n\npackage policy")

	r == set()
}
