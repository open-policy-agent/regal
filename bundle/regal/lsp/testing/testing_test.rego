package regal.lsp.testing_test

import data.regal.lsp.testing

test_at_cursor_in_marks_the_cursor_with_an_at_sign if {
	[tdp, modules] := testing.at_cursor_in({"file:///a.rego": `package p

r := input.a@`})

	tdp.params.position == {"line": 2, "character": 11}
	count(modules["file:///a.rego"].rules) == 1
}

test_at_cursor_in_allows_pipes_in_the_policy if {
	[tdp, modules] := testing.at_cursor_in({"file:///a.rego": `package p

r := [x | some x in input.a; x@ > 1]`})

	tdp.params.position == {"line": 2, "character": 29}
	count(modules["file:///a.rego"].rules) == 1
}
