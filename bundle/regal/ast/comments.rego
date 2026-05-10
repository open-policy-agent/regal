package regal.ast

# METADATA
# description: all comments in the input AST
comments_decoded := [decoded |
	some location in _comments

	[row_str, col_str, end_row_str, end_col_str] := split(location, ":")
	row := to_number(row_str)
	col := to_number(col_str)
	text := substring(input.regal.file.lines[row - 1], col - 1, -1)

	decoded := {
		"row": row,
		"col": col,
		"end": {
			"row": to_number(end_row_str),
			"col": to_number(end_col_str),
		},
		"text": text,
	}
]

# METADATA
# description: |
#   an array of partitions, i.e. arrays containing all comments grouped by their "blocks"
comments["blocks"] := comment_blocks(comments_decoded)

# METADATA
# description: set of all the standard metadata attribute names, as provided by OPA
comments["metadata_attributes"] := {
	"scope",
	"title",
	"description",
	"labels",
	"related_resources",
	"authors",
	"organizations",
	"schemas",
	"entrypoint",
	"custom",
	"compile",
}

# METADATA
# description: true if comment matches a metadata annotation attribute
comments["annotation_match"](str) if regex.match(
	`^#?\s*(scope|title|description|labels|related_resources|authors|organizations|schemas|entrypoint|custom|compile)\s*:`,
	str,
)

# METADATA
# description: |
#   map of all ignore directive comments, like ("# regal ignore:line-length")
#   found in input AST, indexed by the row they're at
ignore_directives[row] := rules if {
	some location in comments_decoded

	contains(location.text, "regal ignore:")

	row := location.row + 1

	rules := regex.split(`,\s*`, trim_space(regex.replace(location.text, `^.*regal ignore:\s*(\S+)`, "$1")))
}

# METADATA
# description: |
#   returns an array of partitions, i.e. arrays containing all comments
#   grouped by their "blocks". only comments on the same column as the
#   one before is considered to be part of a block.
comment_blocks(comments_decoded) := blocks if {
	row_partitions := [partition |
		rows := [location.row | some location in comments_decoded]
		breaks := _splits(rows)

		some j, k in breaks
		partition := array.slice(
			comments_decoded,
			breaks[j - 1] + 1,
			k + 1,
		)
	]

	blocks := [block |
		some row_partition in row_partitions
		some block in {col: partition |
			some location in row_partition

			col := location.col # regal ignore:comprehension-term-assignment
			partition := [partition_location |
				some partition_location in row_partition
				partition_location.col == col
			]
		}
	]
}

# see _rules for information about this hack and why we need it for now
_comments := input.comments

_comments := data.workspace.parsed[u].comments if {
	not input.comments

	u := object.get(input, ["params", "textDocument", "uri"], null)
}

_splits(rows) := array.flatten([
	# -1 ++ [ all indices where there's a step larger than one ] ++ length of xs
	# the -1 is because we're adding +1 in array.slice
	-1,
	[i |
		some i in numbers.range(0, n - 1)
		rows[i + 1] != rows[i] + 1
	],
	n,
]) if {
	n := count(rows)
}
