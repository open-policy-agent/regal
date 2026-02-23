package regal.ast

import data.regal.util

# METADATA
# description: all comments in the input AST with their `text` attribute base64 decoded
comments_decoded := [decoded |
	some comment in input.comments

	text_decoded := base64.decode(comment.text)
	decoded := object.union_n([
		comment,
		{"text": text_decoded},
		{"location": util.to_location_no_text(comment.location)},
		{"location": {"text": $"#{text_decoded}"}},
	])
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
	"related_resources",
	"authors",
	"organizations",
	"schemas",
	"entrypoint",
	"custom",
}

# METADATA
# description: true if comment matches a metadata annotation attribute
comments["annotation_match"](str) if regex.match(
	`^\s*(scope|title|description|related_resources|authors|organizations|schemas|entrypoint|custom)\s*:`,
	str,
)

# METADATA
# description: |
#   map of all ignore directive comments, like ("# regal ignore:line-length")
#   found in input AST, indexed by the row they're at
ignore_directives[row] := rules if {
	some comment in comments_decoded

	contains(comment.text, "regal ignore:")

	row := comment.location.row + 1

	rules := regex.split(`,\s*`, trim_space(regex.replace(comment.text, `^.*regal ignore:\s*(\S+)`, "$1")))
}

# METADATA
# description: |
#   returns an array of partitions, i.e. arrays containing all comments
#   grouped by their "blocks". only comments on the same column as the
#   one before is considered to be part of a block.
comment_blocks(comments_decoded) := blocks if {
	row_partitions := [partition |
		rows := [comment.location.row | some comment in comments_decoded]
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
			some comment in row_partition

			col := comment.location.col # regal ignore:comprehension-term-assignment
			partition := [c |
				some c in row_partition
				c.location.col == col
			]
		}
	]
}

_splits(xs) := array.flatten([
	# -1 ++ [ all indices where there's a step larger than one ] ++ length of xs
	# the -1 is because we're adding +1 in array.slice
	-1,
	[i |
		some i in numbers.range(0, n - 1)
		xs[i + 1] != xs[i] + 1
	],
	n,
]) if {
	n := count(xs)
}
