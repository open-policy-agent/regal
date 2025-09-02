# METADATA
# description: |
#   the code lens provider decides where code lenses should be placed in the given input file
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.completion
package regal.lsp.codelens

import data.regal.ast
import data.regal.result
import data.regal.util

import data.regal.lsp.util.location

# METADATA
# entrypoint: true
default result["response"] := []

result["response"] := lenses if count(input.regal.file.parse_errors) == 0

result["response"] := lenses if {
	count(input.regal.file.parse_errors) > 0
	count(input.regal.file.lines) == input.regal.file.successful_parse_count
}

# code lenses are displayed in the order they come back in the returned
# array, and 'evaluate' somehow feels better to the left of 'debug'
# METADATA
# description: contains code lenses determined for module
lenses := array.concat(
	[l | some l in _eval_lenses],
	[l | some l in _debug_lenses],
)

# METADATA
# description: Debug lens included in response only when client supports it
debug_supported if input.regal.client.init_options.enableDebugCodelens == true

_module := data.workspace.parsed[input.params.textDocument.uri]

_eval_lenses contains {
	"range": location.to_range(result.location(_module.package).location),
	"command": {
		"title": "Evaluate",
		"command": "regal.eval",
		"arguments": [json.marshal({
			"target": input.params.textDocument.uri,
			"path": ast.ref_to_string(_module.package.path),
			"row": util.to_location_object(_module.package.location).row,
		})],
	},
}

_eval_lenses contains _rule_lens(input.params.textDocument.uri, rule, "regal.eval", "Evaluate") if {
	some rule in _module.rules

	# can't evaluate functions
	not rule.head.args
}

_debug_lenses contains lens if {
	debug_supported

	lens := {
		"range": location.to_range(result.location(_module.package).location),
		"command": {
			"title": "Debug",
			"command": "regal.debug",
			"arguments": [json.marshal({
				"target": input.params.textDocument.uri,
				"path": ast.ref_to_string(_module.package.path),
				"row": util.to_location_object(_module.package.location).row,
			})],
		},
	}
}

_debug_lenses contains lens if {
	debug_supported

	some rule in _module.rules

	# can't debug functions
	not rule.head.args

	# no need to add a debug lens for a rule like `pi := 3.14`
	not _unconditional_constant(rule)

	lens := _rule_lens(input.params.textDocument.uri, rule, "regal.debug", "Debug")
}

_rule_lens(file_uri, rule, command, title) := {
	"range": location.to_range(result.location(rule).location),
	"command": {
		"title": title,
		"command": command,
		"arguments": [json.marshal({
			"target": file_uri,
			"path": concat(".", [
				ast.ref_to_string(_module.package.path),
				ast.ref_static_to_string(rule.head.ref),
			]),
			"row": util.to_location_object(rule.head.location).row,
		})],
	},
}

_unconditional_constant(rule) if {
	not rule.body
	ast.is_constant(rule.head.value)
}
