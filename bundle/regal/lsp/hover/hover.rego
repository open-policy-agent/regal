# METADATA
# description: |
#   Display tooltips on hover over elements like built-in functions and keywords.
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.hover

import data.regal.util

import data.regal.lsp.location

import data.regal.lsp.hover.example
import data.regal.lsp.hover.keywords

# METADATA
# entrypoint: true
# scope: document
default result["response"] := null

# METADATA
# description: Return hover information for built-in functions
# scope: rule
result["response"] := hover if {
	line := input.params.position.line
	char := input.params.position.character
	text := input.regal.file.lines[line]
	call := location.ref_at(text, char + 1)

	# ensure this is a call, as e.g. `contains` could refer either to the built-in function or the keyword
	substring(text, char + call.offset_after, 1) == "("

	hover := {
		"contents": {
			"kind": "markdown",
			"value": tooltip(data.workspace.builtins[call.text]),
		},
		"range": {
			"start": {
				"line": line,
				"character": char - call.offset_before,
			},
			"end": {
				"line": line,
				"character": char + call.offset_after,
			},
		},
	}
}

# METADATA
# description: Return hover information for (supported) keywords
# scope: rule
result["response"] := hover if {
	line := input.params.position.line
	char := input.params.position.character
	text := input.regal.file.lines[line]
	word := location.word_at(text, char + 1)

	# cheap check before more expensive AST keywords lookup
	word.text in {"if", "package", "import", "contains", "some", "every", "in"}

	keyword := _keyword(word.text, line + 1, char + 1)

	hover := {
		"contents": {
			"kind": "markdown",
			"value": example.keyword(keyword.name),
		},
		"range": {
			"start": {
				"line": line,
				"character": keyword.location.col - 1,
			},
			"end": {
				"line": line,
				"character": (keyword.location.col - 1) + count(word),
			},
		},
	}
}

_keyword(name, row, col) := keyword if {
	# REPORT THIS: looks like this is a bug in OPA, as without the is_number checks
	# the type checker fails to identify the types passed here, and reports the
	# following error at the callsite above:
	#
	# rego_type_error: data.regal.lsp.hover._keyword: invalid argument(s)
	#     have: (string, number, number, ???)
	#     want: (any, string, any, any)
	#
	is_number(row)
	is_number(col)

	# should always be exactly one, but some abundance of caution
	# to avoid eval conflict errors however unlikely
	keyword := [keyword |
		some keyword in keywords.by_row[row]
		keyword.location.col <= col
		keyword.location.col + count(name) > col
	][0]
}

# METADATA
# description: Render built-in function documentation as a markdown tooltip
tooltip(func) := trim_space($`### [{func.name}]({_doc_url(func)})

{example.builtin(func.name)}{_fenced_rego(func)}

{object.get(func, ["description"], "")}

{_arguments(func)}

{_returns(func)}
`)

default _fenced_rego(_) := ""

_fenced_rego(func) := $"```rego\n{rets} := {func.name}({args})\n```" if {
	func.name != "print"

	rets := object.get(func, ["decl", "result", "name"], "output")
	args := concat(", ", [arg.name | some arg in func.decl.args])
}

_fenced_rego(func) := $"```rego\n{func.name}(arg1, arg2, ...)\n```\n" if func.decl.variadic

_category(func) := func.categories[0] if {
	func.categories != []
} else := util.substring_to(func.name, 0, ".")

_doc_url(func) := override if {
	override := _doc_override(func)
} else := $"https://www.openpolicyagent.org/docs/policy-reference/#builtin-{cat}-{txt}" if {
	cat := _category(func)
	txt := replace(func.name, ".", "")
}

_doc_override(func) := "https://www.openpolicyagent.org/docs/policy-reference/builtins/opa#debugging" if {
	# annoying special case that we should fix in OPA
	# (there's no anchor for the print function)
	func.name == "print"
}

_doc_override(func) := substring(text, 4, 1000) if {
	# first see if there's an explicit link override in the categories
	# which is a hack EOPA did/does for custom links since 1.29.1
	some text in func.categories

	startswith(text, "url=")
}

default _arguments(_) := ""

# regal ignore:narrow-argument
_arguments(func) := text if {
	func.decl.args != []

	args := [$"- `{arg.name}` {arg.type} — {arg.description}" | some arg in func.decl.args]
	text := concat("\n", array.flatten(["#### Arguments\n", args]))
}

default _returns(_) := ""

_returns(func) := $"#### Returns `{name}` of type `{func.decl.result.type}`{desc}\n" if {
	name := object.get(func, ["decl", "result", "name"], "output")
	desc := $": {object.get(func, ["decl", "result", "description"], "")}"
}
