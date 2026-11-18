package regal.prepared_test

import data.regal.prepared

test_validate_known_categories if {
	report := prepared.prepare.validate.unknown
		with data.eval.params.enable_category as ["foo"]
		with data.eval.params.disable_category as ["bar"]
		with data.regal.config.rules as {"baz": {}}
		with data.internal.rules as {"foo": {}, "bar": {}, "baz": {}}

	report == set()
}

test_validate_known_rules if {
	report := prepared.prepare.validate.unknown
		with data.eval.params.enable as ["foo"]
		with data.eval.params.disable as ["bar"]
		with data.regal.config.rules as {"bugs": {"baz": {}}}
		with data.internal.rules as {"bugs": {"foo": {}, "bar": {}, "baz": {}}}

	report == set()
}

test_validate_unknown_categories if {
	report := prepared.prepare.validate.unknown
		with data.eval.params.enable_category as ["foo"]
		with data.eval.params.disable_category as ["bar"]
		with data.regal.config.rules as {"baz": {}, "bugs": {}}
		with data.internal.rules as {"bugs": {}}

	report == {
		"unknown category: foo",
		"unknown category: bar",
		"unknown category: baz",
	}
}

test_validate_unknown_rules if {
	report := prepared.prepare.validate.unknown
		with data.eval.params.enable as ["foo"]
		with data.eval.params.disable as ["bar"]
		with data.regal.config.rules as {"bugs": {"baz": {}}}
		with data.internal.rules as {"bugs": {"qux": {}}} # qux is the only known rule

	report == {
		"unknown rule: foo",
		"unknown rule: bar",
		"unknown rule: baz",
	}
}
