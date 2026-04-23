package regal.rules.idiomatic["directory-package-mismatch_test"]

import data.regal.config

import data.regal.rules.idiomatic["directory-package-mismatch"] as rule

test_success_directory_structure_package_path_match if {
	r := rule.report
		with input as regal.parse_module("foo/bar/baz/p.rego", "package foo.bar.baz")
		with config.rules as _default_config

	r == set()
}

test_fail_directory_structure_package_path_mismatch if {
	r := rule.report
		with input as regal.parse_module("foo/bar/baz/p.rego", "package foo.baz.bar")
		with config.rules as _default_config

	r == with_location({
		"col": 9,
		"file": "foo/bar/baz/p.rego",
		"row": 1,
		"end": {
			"col": 20,
			"row": 1,
		},
		"text": "package foo.baz.bar",
	})
}

test_success_directory_structure_package_path_match_longer_directory_path if {
	r := rule.report
		with input as regal.parse_module("system/directories/foo/bar/baz/p.rego", "package foo.bar.baz")
		with config.rules as _default_config

	r == set()
}

with_location(location) := {{
	"category": "idiomatic",
	"description": "Directory structure should mirror package",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/directory-package-mismatch",
	}],
	"title": "directory-package-mismatch",
}}

_default_config := {"idiomatic": {"directory-package-mismatch": {
	"level": "error",
	"exclude-test-suffix": true,
}}}
