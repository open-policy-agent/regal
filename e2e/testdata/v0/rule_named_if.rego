package rule_named_if

allow := true if {
	input.foo == "bar"
}
