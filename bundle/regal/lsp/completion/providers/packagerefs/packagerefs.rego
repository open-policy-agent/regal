# METADATA
# description: Completion suggestions for importing available packages
package regal.lsp.completion.providers.packagerefs

import data.regal.ast

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: suggest packages matching typed import ref
items contains item if {
	position := location.to_position(input.regal.context.location)
	line := input.regal.file.lines[position.line]

	startswith(line, "import ")

	ref := location.ref_at(line, input.regal.context.location.col)

	startswith(ref.text, "d")

	some i, path in _paths_sorted

	startswith(path, ref.text)

	item := {
		"label": path,
		"kind": kind.module,
		"detail": "package",
		"textEdit": {
			"range": location.word_range(ref, position),
			"newText": path,
		},
		# tell clients to sort paths first by the number of path components (shortest first),
		# and only then alphabetically (done in _paths_by_num_parts and _paths_sorted)
		"sortText": sprintf("%03d", [i]),
	}
}

_package_paths contains str if {
	some uri
	path := data.workspace.parsed[uri].package.path

	uri != input.regal.file.uri # don't suggest the package of the current file
	not endswith(regal.last(path).value, "_test") # importing tests makes no sense

	str := ast.ref_to_string(path)
}

# regal ignore:prefer-set-or-object-rule
_paths_by_num_parts := {num_parts: sort(paths) |
	some i
	num_parts := strings.count(_package_paths[i], ".")
	paths := [path |
		some j
		strings.count(_package_paths[j], ".") == num_parts
		path := _package_paths[j]
	]
}

_paths_sorted := [path |
	some i in sort(object.keys(_paths_by_num_parts))
	some path in _paths_by_num_parts[i]
]
