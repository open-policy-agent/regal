# METADATA
# description: |
#   the `packagename` providers suggests completions for package
#   name based on the directory structure whre the file is located
package regal.lsp.completion.providers.packagename

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: set of suggested package names
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	startswith(line, "package ")
	input.params.position.character > 7

	path_sep := input.regal.environment.path_separator

	abs_dir := _base(input.params.textDocument.uri)
	rel_dir := trim_prefix(abs_dir, input.regal.environment.workspace_root_path)
	fix_dir := replace(replace(trim_prefix(rel_dir, path_sep), ".", "_"), path_sep, ".")

	word := location.ref_at(line, input.params.position.character + 1)

	some suggestion in _suggestions(fix_dir, word.text)

	item := {
		"label": suggestion,
		"kind": kind.folder,
		"detail": "suggested package name based on directory structure",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": $"{suggestion}\n\n",
		},
	}
}

_base(uri) := str if {
	end := trim_prefix(uri, "file://")
	str := substring(end, 0, regal.last(indexof_n(end, input.regal.environment.path_separator)))
}

_suggestions(dir, text) := [str |
	parts := split(dir, ".")
	len_p := count(parts)

	some n in numbers.range(0, len_p)

	formatted_parts := [formatted |
		some index, str in array.slice(parts, n, len_p)
		formatted := _format_part(str, _needs_quoting(str))
	]

	str := concat("", [delimited |
		some index, str in formatted_parts
		delimited := _delimit_part(str, array.slice(formatted_parts, index + 1, index + 2))
	])

	str != ""

	# it's not valid Rego to have a hypenated first part
	not startswith(str, `["`)

	startswith(str, text)
]

# matches anything with a non alphanumeric character or underscore anywhere in
# the part. E.g. "foo@bar", "@foo-bar" etc.
_needs_quoting(str) := regex.match(`[^a-zA-Z0-9_]`, str)

_format_part(str, false) := str
_format_part(str, true) := $`["{str}"]`

_delimit_part(str, next) := $"{str}." if {
	next != []
	not startswith(next[0], "[")
}

_delimit_part(str, next) := str if {
	next != []
	startswith(next[0], "[")
}

_delimit_part(str, []) := str
