package regal.lsp.testlocations_test

import data.regal.ast
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
			"kind": "package_test",
			"package": "data.foo_test",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo_test.rego",
				"text": "package foo_test",
			},
		},
		{
			"kind": "file_test",
			"package": "data.foo_test",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo_test.rego",
				"text": "package foo_test",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo_test",
			"test": "test_1",
			"location": {
				"col": 1,
				"row": 3,
				"end": {"col": 7, "row": 3},
				"file": "file://foo_test.rego",
				"text": "test_1 if {",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo_test",
			"test": "test_2",
			"location": {
				"col": 1,
				"row": 7,
				"end": {"col": 7, "row": 7},
				"file": "file://foo_test.rego",
				"text": "test_2 if true",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo_test",
			"test": "test_3",
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

	{
		{
			"kind": "file_test",
			"package": "data.foo",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo.rego",
				"text": "package foo",
			},
		},
		{
			"kind": "package_test",
			"package": "data.foo",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo.rego",
				"text": "package foo",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo",
			"test": "test_foo",
			"location": {
				"col": 1,
				"row": 3,
				"end": {"col": 9, "row": 3},
				"file": "file://foo.rego",
				"text": "test_foo if true",
			},
		},
	} == result
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
			"kind": "file_test",
			"package": "data.foo",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo.rego",
				"text": "package foo",
			},
		},
		{
			"kind": "package_test",
			"package": "data.foo",
			"location": {
				"col": 1,
				"row": 1,
				"end": {"col": 8, "row": 1},
				"file": "file://foo.rego",
				"text": "package foo",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo",
			"test": "foo.bar.test_me",
			"location": {
				"col": 1,
				"row": 3,
				"end": {"col": 16, "row": 3},
				"file": "file://foo.rego",
				"text": "foo.bar.test_me if {",
			},
		},
		{
			"kind": "single_test",
			"package": "data.foo",
			"test": "foo.bar.test_me.baz",
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
