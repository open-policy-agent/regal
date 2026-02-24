package regal.lsp.testlocations_test

import data.regal.lsp.testlocations

test_multiple_test_rules if {
	policy := `package foo_test

test_1 if {
	1 == 2
}

test_2 if true

  test_3 if {
	1 == 2
	2 == 1
}
`

	result := testlocations.result with input as regal.parse_module("file://foo_test.rego", policy)

	{
		{
			"package": "data.foo_test",
			"name": "test_1",
			"location": {
				"col": 1,
				"row": 3,
				"end": {"col": 7, "row": 3},
				"file": "file://foo_test.rego",
				"text": "test_1 if {",
			},
		},
		{
			"package": "data.foo_test",
			"name": "test_2",
			"location": {
				"col": 1,
				"row": 7,
				"end": {"col": 7, "row": 7},
				"file": "file://foo_test.rego",
				"text": "test_2 if true",
			},
		},
		{
			"package": "data.foo_test",
			"name": "test_3",
			"location": {
				"col": 3,
				"row": 9,
				"end": {"col": 9, "row": 9},
				"file": "file://foo_test.rego",
				"text": "  test_3 if {",
			},
		},
	} == result
}

test_no_test_rules if {
	policy := `package foo_test`

	result := testlocations.result with input as regal.parse_module("file://foo_test.rego", policy)

	set() == result
}

test_non_test_package if {
	policy := `package foo

test_foo if true
`

	result := testlocations.result with input as regal.parse_module("file://foo.rego", policy)

	{{
		"package": "data.foo",
		"name": "test_foo",
		"location": {
			"col": 1,
			"row": 3,
			"end": {"col": 9, "row": 3},
			"file": "file://foo.rego",
			"text": "test_foo if true",
		},
	}} == result
}

test_funny_test_names_package if {
	policy := `package foo

foo.bar.test_me if {
	false
}

foo.bar.test_me.baz if {
	false
}
`

	result := testlocations.result with input as regal.parse_module("file://foo.rego", policy)

	{
		{
			"package": "data.foo",
			"name": "foo.bar.test_me",
			"location": {
				"col": 1,
				"row": 3,
				"end": {"col": 16, "row": 3},
				"file": "file://foo.rego",
				"text": "foo.bar.test_me if {",
			},
		},
		{
			"package": "data.foo",
			"name": "foo.bar.test_me.baz",
			"location": {
				"col": 1,
				"row": 7,
				"end": {"col": 20, "row": 7},
				"file": "file://foo.rego",
				"text": "foo.bar.test_me.baz if {",
			},
		},
	} == result
}
