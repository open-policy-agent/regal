# METADATA
# description: |
#   Display inlay hints next to function call arguments, and possibly other places in the future.
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_inlayHint
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.inlayhint
package regal.lsp.inlayhint

import data.regal.ast
import data.regal.util

import data.regal.lsp.inlayhint_resolve

# METADATA
# entrypoint: true
# scope: document
default result["response"] := null

result["response"] := _hints if {
	# later might look into partial rendering of
	# hints up until the line before the error
	_parse_errors == []
}

default _parse_errors := []

_parse_errors := input.regal.file.parse_errors

_hints contains hint if {
	some call in _calls_in_range

	info := _args_info(ast.ref_static_to_string(call[0].value))

	some i, arg in array.slice(call, 1, 100)

	[row, col, _, _] = split(arg.location, ":")

	# check that the argument is within the requested range
	line := to_number(row) - 1
	line >= input.params.range.start.line
	line <= input.params.range.end.line
	char := to_number(col) - 1

	hint := _build_hint(info[i], line, char)
}

_build_hint(info, line, char) := hint if {
	_tooltip_resolve_supported

	hint := {
		"label": concat("", [info.name, ":"]),
		"position": {
			"line": line,
			"character": char,
		},
		# 1 = type, 2 = parameter
		"kind": 2,
		"paddingRight": true,
		# tooltip data rendered by resolver when requested
		"data": object.filter(info, ["name", "type", "description"]),
	}
}

_build_hint(info, line, char) := hint if {
	not _tooltip_resolve_supported

	hint := {
		"label": concat("", [info.name, ":"]),
		"position": {
			"line": line,
			"character": char,
		},
		"kind": 2,
		"paddingRight": true,
		"tooltip": {
			"kind": "markdown",
			"value": inlayhint_resolve.markdown(info),
		},
	}
}

_args_info(name) := builtin.decl.args if {
	builtin := data.workspace.builtins[name]
	# TBD: consider caching this
} else := ast.function_decls[name].decl.args

_tooltip_resolve_supported if {
	"tooltip" in input.regal.client.capabilities.textDocument.inlayHint.resolveSupport.properties
}

_calls_in_range contains call if {
	call := ast.found.calls[_][_]

	not call[0].value[0].value in _infix_and_internal

	line := to_number(util.substring_to(call[0].location, 0, ":")) - 1
	line >= input.params.range.start.line
	line <= input.params.range.end.line
}

_infix_and_internal := {
	"assign",
	"neq",
	"gt",
	"gte",
	"lt",
	"lte",
	"equal",
	"plus",
	"minus",
	"mul",
	"div",
	"rem",
	"and",
	"or",
	"internal",
}
