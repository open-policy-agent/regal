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

	ps := input.regal.environment.path_separator

	abs_dir := _base(input.params.textDocument.uri)
	rel_dir := trim_prefix(abs_dir, input.regal.environment.workspace_root_path)
	fix_dir := replace(replace(trim_prefix(rel_dir, ps), ".", "_"), ps, ".")

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

_base(uri) := base if {
	path := trim_prefix(uri, "file://")
	base := substring(path, 0, regal.last(indexof_n(path, input.regal.environment.path_separator)))
}

_suggestions(dir, text) := [path |
	parts := split(dir, ".")
	len_p := count(parts)

	some n in numbers.range(0, len_p)

	formatted_parts := [p |
		some index, part in array.slice(parts, n, len_p)
		p := _format_part(part, _needs_quoting(part))
	]

	path := concat("", [p |
		some index, part in formatted_parts
		p := _delimit_part(part, array.slice(formatted_parts, index + 1, index + 2))
	])

	path != ""

	# it's not valid Rego to have a hypenated first part
	not startswith(path, `["`)

	startswith(path, text)
]

# matches anything with a non alphanumeric character or underscore anywhere in
# the part. E.g. "foo@bar", "@foo-bar" etc.
_needs_quoting(part) := regex.match(`[^a-zA-Z0-9_]`, part)

_format_part(part, false) := part
_format_part(part, true) := $`["{part}"]`

_delimit_part(part, next_part) := $"{part}." if {
	next_part != []
	not startswith(next_part[0], "[")
}

_delimit_part(part, next_part) := part if {
	next_part != []
	startswith(next_part[0], "[")
}

_delimit_part(part, []) := part
