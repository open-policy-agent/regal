package regal.rules.{{.Category}}{{.NameTest}}

import data.regal.ast

import data.regal.rules.{{.Category}}{{.Name}} as rule

# Example test, replace with your own
test_aggregate_reports_violation if {
	agg := rule.aggregate with input as ast.policy("foo := true")

	r := rule.aggregate_report with input.aggregate as agg

	# Use print(r) here to see the report. Great for development!

	count(r) > 0
}
