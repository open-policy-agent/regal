package regal.config_test

import data.regal.config

cases := {
	"p.rego": {
		"p.rego": true,
		"dir/p.rego": true,
		"dir1/dir2/p.rego": true,
		"q.rego": false,
	},
	"**/p.rego": {
		"p.rego": true,
		"dir/p.rego": true,
		"dir1/dir2/p.rego": true,
		"q.rego": false,
	},
	"dir/**/p.rego": {
		"dir/sub/p.rego": true,
		"dir2/sub/p.rego": false,
		"dir/p.rego": true,
		"dir2/p.rego": false,
		"dir/sub/sub1/p.rego": true,
		"dir/sub/q.rego": false,
	},
	"*_test.rego": {
		"p_test.rego": true,
		"q_test.rego": true,
		"dir/p_test.rego": true,
		"p.rego": false,
		"ptest.rego": false,
	},
	"outer/inner/": {
		"outer/inner/file": true,
		"outer/inner": false,
	},
	"/p.rego": {
		"p.rego": true,
		"dir/p.rego": false,
		"p.rego/file": true,
	},
}

test_all_cases_are_as_expected if {
	not_exp := {pattern: res |
		some pattern, subcases in cases
		res := {file |
			some file, exp in subcases
			act := exclude(pattern, file)
			exp != act
		}
		count(res) > 0
	}

	count(not_exp) == 0
}

exclude(pattern, file) if {
	some p in config.patterns_compiler([pattern])
	glob.match(p, ["/"], file)
}

rules_config_error := {"rules": {"test": {"test-case": {"level": "error"}}}}

rules_config_ignore_delta := {"rules": {"test": {"test-case": {"ignore": {"files": ["p.rego"]}}}}}

config_ignore := {"ignore": {"files": ["p.rego"]}}

test_excluded_file_default if {
	not config.excluded_file("test", "test-case", "p.rego") with data.eval.params as params({})
		with config.merged_config as rules_config_error
}

test_excluded_file_with_ignore if {
	compiled := config.patterns_compiler(config_ignore.ignore.files) with config.merged_config as object.union(
		rules_config_error,
		rules_config_ignore_delta,
	)

	config.excluded_file("test", "test-case", "p.rego") with data.regal.util as "obnoxious formatter"
		with data.internal.prepared.ignore_patterns.files.test["test-case"] as compiled
}

test_excluded_file_cli_overrides_config if {
	not config.excluded_file("test", "test-case", "p.rego") with config.merged_config as config_ignore
		with data.eval.params as params({"ignore_files": [""]})
}

test_trailing_slash[pattern] if {
	some [pattern, want] in [
		["foo/**/bar", {"foo/**/bar", "foo/**/bar/**"}],
		["foo", {"foo", "foo/**"}],
		["foo/**", {"foo/**"}],
	]

	config._trailing_slash(pattern) == want
}
