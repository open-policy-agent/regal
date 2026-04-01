package regal.rules.performance["non-loop-expression_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.performance["non-loop-expression"] as rule

test_loop_start_points_walks_are_loops if {
	sps := rule._loop_start_points with input as ast.policy(`
allow if {
	walk(foo, [path, value])
	value == "foo"
}
`)

	sps == {"0": {5: {
		{"location": "5:13:5:17", "type": "var", "value": "path"},
		{"location": "5:19:5:24", "type": "var", "value": "value"},
	}}}
}

test_loop_start_points_some_k_v if {
	sps := rule._loop_start_points with input as ast.policy(`
allow if {
	some k, v in input.foo
	k == 1
	some value in v
	v == 2
}
`)

	sps == {"0": {
		5: {
			{"location": "5:10:5:11", "type": "var", "value": "v"},
			{"location": "5:7:5:8", "type": "var", "value": "k"},
		},
		7: {{"location": "7:7:7:12", "type": "var", "value": "value"}},
	}}
}

test_loop_start_points_ignore_comps if {
	sps := rule._loop_start_points with input as ast.policy(`
allow if {
	some k, v in input.foo
	foo := {e | some e in v}
}
`)

	sps == {"0": {5: {
		{"location": "5:10:5:11", "type": "var", "value": "v"},
		{"location": "5:7:5:8", "type": "var", "value": "k"},
	}}}
}

test_loop_start_points_some if {
	sps := rule._loop_start_points with input as ast.policy(`
allow if {
	some user in data.users
	input.email := user.email
	foo := "bar"
}
`)

	sps == {"0": {5: {{"location": "5:7:5:11", "type": "var", "value": "user"}}}}
}

test_loop_start_points_wildcard if {
	sps := rule._loop_start_points with input as ast.policy(`
allow if {
	email := input.emails[_]
	email == "foo"
}
`)

	sps == {"0": {5: {{"location": "5:2:5:7", "type": "var", "value": "email"}}}}
}

test_assignment_index if {
	ai := rule._assignment_index with input as ast.policy(`
allow if {
	some user in data.users
	email := user.emails[_]
	email == input.email

	foo := "bar"
	some k, v in input.foo
	walk(foo, [path, value])
	baz := value[_]
}
`)

	ai == {"0": {
		"baz": {12},
		"email": {6},
		"foo": {9},
		"k": {10},
		"path": {11},
		"user": {5},
		"v": {10},
		"value": {11},
	}}
}

test_assignment_index_comprehension if {
	ai := rule._assignment_index with input as ast.policy(`
allow if {
	foos := {bar|
		bar := 1
	}

	some bar, baz in foos

	not bar
}`)

	ai == {"0": {
		"foos": {5},
		"bar": {9},
		"baz": {9},
	}}
}

test_ok_when_correct_order if {
	r := rule.report with input as ast.policy(`
allow if {
	endswith(input.email, "acmecorp.com")
	some user in data.users
	user.email == input.email
}`)

	r == set()
}

test_ok_loop_expression_is_print if {
	r := rule.report with input as ast.policy(`
allow if {
	some user in data.users
	print("hello")
}`)

	r == set()
}

test_ok_when_dependent_vars_are_also_in_loop if {
	r := rule.report with input as ast.policy(`
allow if {
	some user in data.users
	caps_name := upper(user.name)
	caps_name == input.name
}`)

	r == set()
}

test_ok_when_not_in_scope if {
	r := rule.report with input as ast.policy(`
allow if {
	foo := { f | some f in input.bar }
	endswith(input.email, "acmecorp.com")
}`)

	r == set()
}

test_fail_single_some_in if {
	r := rule.report with input as ast.policy(`
allow if {
	some user in data.users
	endswith(input.email, "acmecorp.com")
	user.email == input.email
}`)

	r == with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 39,
			"row": 6,
		},
		"text": "\tendswith(input.email, \"acmecorp.com\")",
	})
}

test_fail_single_some if {
	r := rule.report with input as ast.policy(`
allow if {
	some email
	user.emails[email] == input.email
	endswith(input.email, "acmecorp.com")
}`)

	r == with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 7,
		"end": {
			"col": 39,
			"row": 7,
		},
		"text": "\tendswith(input.email, \"acmecorp.com\")",
	})
}

test_fail_wildcard if {
	r := rule.report with input as ast.policy(`
allow if {
	foo := input.foo[_]
	endswith(input.email, "acmecorp.com")
	input.foo == foo
}`)

	r == with_location({
		"col": 2,
		"end": {
			"col": 39,
			"row": 6,
		},
		"file": "policy.rego",
		"row": 6,
		"text": "\tendswith(input.email, \"acmecorp.com\")",
	})
}

# this should be the basis for another rule, but for now it should be a non-error
test_ok_unrelated_loops if {
	r := rule.report with input as ast.policy(`
allow if {
	foo := input.foo[_]
	input.foo == foo

	some bar in data.bars
	input.bar == bar
}`)

	r == set()
}

test_fail_two_somes if {
	r := rule.report with input as ast.policy(`
allow if {
	some user in data.users
	some email in user.emails
	endswith(input.user_email, "acmecorp.com")
	email == input.user_email
}`)

	r == with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 7,
		"end": {
			"col": 44,
			"row": 7,
		},
		"text": "\tendswith(input.user_email, \"acmecorp.com\")",
	})
}

test_wildcard_assign if {
	r := rule.report with input as ast.policy(`
allow if {
	user := data.users[_]
	endswith(input.user_email, "acmecorp.com")
	user.email == input.user_email
}`)

	r == with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 44,
			"row": 6,
		},
		"text": "\tendswith(input.user_email, \"acmecorp.com\")",
	})
}

test_some_key_value if {
	r := rule.report with input as ast.policy(`
allow if {
	some userID, user in data.users
	endswith(input.user_email, "acmecorp.com")
	user.email == input.user_email
	userID != 0
}`)

	r == with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 44,
			"row": 6,
		},
		"text": "\tendswith(input.user_email, \"acmecorp.com\")",
	})
}

test_every if {
	r := rule.report with input as ast.policy(`
allow if {
    every role in data.required_roles {
        endswith(input.email, "acmecorp.com")
        role in input.roles
    }
}`)

	r == with_location({
		"col": 9,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 46,
			"row": 6,
		},
		"text": "        endswith(input.email, \"acmecorp.com\")",
	})
}

# https://github.com/open-policy-agent/regal/issues/1436
test_success_not_loop_unification if {
	r := rule.report with input as ast.policy(`
	allow if {
		x = ["accounts", _]
		y
	}`)

	r == set()
}

# confirm fix for: https://github.com/open-policy-agent/regal/issues/1443
test_ref_loop_output_var if {
	r := rule.report with input as ast.policy(`r if {
		input[a].d[_]
		a == 1
	}`)

	r == set()
}

# mentioned in: https://github.com/open-policy-agent/regal/issues/1443
test_non_ref_output_var if {
	r := rule.report with input as ast.policy(`r if {
		some petid
		input.path = ["pets", petid]
		input.user == input.owner
	}`)

	r == set()
}

with_location(location) := {{
	"category": "performance",
	"description": "Non-loop expression",
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/non-loop-expression", "performance"),
	}],
	"title": "non-loop-expression",
	"location": location,
	"level": "error",
}}
