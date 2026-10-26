package regal.rules.imports["redundant-future-keywords_test"]

import data.regal.capabilities
import data.regal.config
import data.regal.rules.imports["redundant-future-keywords"] as rule

test_fail_obsolete_future_keyword_import if {
	module := regal.parse_module("policy.rego", `package policy
import future.keywords.every`)

	violations := rule.report with input as module
		with config.capabilities as capabilities.provided

	count(violations) == 1
	violations[_].title == "redundant-future-keywords"
	violations[_].location.text == "import future.keywords.every"
}

test_success_remaining_future_keyword_imports if {
	module := regal.parse_module("policy.rego", `package policy
import future.keywords.not
import future.keywords.or
import future.keywords.and
import future.keywords`)

	violations := rule.report with input as module
		with config.capabilities as capabilities.provided

	violations == set()
}

test_success_when_opa_v1_is_not_targeted if {
	module := regal.parse_module("policy.rego", `package policy
import future.keywords.every`)

	violations := rule.report with input as module
		with config.capabilities as {"features": []}

	violations == set()
}
