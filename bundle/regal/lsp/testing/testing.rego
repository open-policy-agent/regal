package regal.lsp.testing

# METADATA
# description: |
#   input and parsed module for a snippet, where the cursor is on the character before the "|" in the
#   snippet. A package declaration is added, and the module is available as file:///p.rego. Primarily
#   for testing.
at_cursor(snippet) := [tdp, modules["file:///p.rego"]] if {
	[tdp, modules] := at_cursor_in({"file:///p.rego": $"package p\n\n{snippet}"})
}

# METADATA
# description: |
#   input and parsed modules for policies keyed by file URI, where the cursor is on the character before
#   the "|" in one of them. Primarily for testing.
at_cursor_in(policies) := [tdp, modules] if {
	modules := {file: regal.parse_module(file, replace(policy, "|", "")) | some file, policy in policies}

	some name, text in policies
	contains(text, "|")

	lines_before := split(split(text, "|")[0], "\n")

	tdp := {
		"params": {
			"textDocument": {"uri": name},
			"position": {
				"line": count(lines_before) - 1,
				"character": count(regal.last(lines_before)) - 1,
			},
		},
		"regal": {"file": {"lines": split(replace(text, "|", ""), "\n")}},
	}
}
