# METADATA
# description: package for signature help provider policies.
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.signaturehelp
package regal.lsp.signaturehelp

import rego.v1

# METADATA
# entrypoint: true
result["response"] := signature

# METADATA
# description: The builtin signature help response
# scope: document
default signature := null

signature := s if {
	func_info := _function_at_position(input.regal.file.lines, input.params.position)
	builtin_info := data.workspace.builtins[func_info.name]

	s := {
		"signatures": [{
			"label": _build_function_label(builtin_info.decl, func_info.name),
			# some builtins had a space at the start
			"documentation": trim_space(builtin_info.description),
			"parameters": _build_parameters(builtin_info.decl.args),
			"activeParameter": func_info.active_param - 1,
		}],
		"activeSignature": 0,
		"activeParameter": func_info.active_param - 1,
	}
}

default _function_at_position(_, _) := {}

_function_at_position(lines, position) := func if {
	content := concat("\n", lines)
	text := _text_up_to_position(lines, content, position)

	# we want the last one specifically, so need to get all
	result := regex.find_all_string_submatch_n(`([a-zA-Z_][a-zA-Z0-9_.]*)\(([^)]*)$`, text, -1)
	last_match := regal.last(result)

	func := {"name": last_match[1], "active_param": strings.count(last_match[2], ",") + 1}
}

# when position is after the last line
_text_up_to_position(lines, content, position) := content if position.line >= count(lines)

# when char is off the last line
_text_up_to_position(lines, content, position) := content if {
	position.line < count(lines)
	current_line := lines[position.line]
	position.character >= count(current_line)
}

_text_up_to_position(lines, _, position) := concat("\n", all_lines) if {
	position.line < count(lines)

	current_line := lines[position.line]
	position.character < count(current_line)

	complete_lines := [line | some i, line in lines; i < position.line]
	partial_line := substring(current_line, 0, position.character)

	all_lines := array.concat(complete_lines, [partial_line])
}

_build_function_label(decl, func_name) := label if {
	param_labels := concat(", ", [_param_label(arg) | some arg in decl.args])
	label := sprintf("%s(%s) -> %s", [func_name, param_labels, decl.result.type])
}

_build_parameters(args) := [param |
	some arg in args
	label := _param_label(arg)
	param := {
		"label": label,
		"documentation": sprintf("(%s): %s", [label, arg.description]),
	}
]

_param_label(arg) := concat(": ", [arg.name, arg.type])
