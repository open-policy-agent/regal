# METADATA
# description: various helpers to be used for testing completions providers
package regal.lsp.completion.providers.test_utils

# METADATA
# description: returns a map of all parsed modules in the workspace
parsed_modules(workspace) := {file_uri: parsed_module |
	some file_uri, contents in workspace
	parsed_module := regal.parse_module(file_uri, contents)
}

# METADATA
# description: adds location metadata to provided module, to be used as input
input_module_with_location(module, policy, location) := object.union(module, {
	"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"position": {"line": location.row - 1, "character": location.col - 1},
	},
	"regal": {"file": {
		"uri": "file:///p.rego",
		"lines": split(policy, "\n"),
	}},
})

# METADATA
# description: same as input_module_with_location, but accepts text content rather than a module
input_with_location(policy, location) := {
	"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"position": {"line": location.row - 1, "character": location.col - 1},
	},
	"regal": {"file": {"lines": split(policy, "\n")}},
}

# METADATA
# description: same as input_with_location but with option to set rego_version too
input_with_location_and_version(policy, location, rego_version) := {
	"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"position": {"line": location.row - 1, "character": location.col - 1},
	},
	"regal": {"file": {
		"lines": split(policy, "\n"),
		"rego_version": rego_version,
	}},
}
