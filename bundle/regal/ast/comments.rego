package regal.ast

import data.regal.util

# METADATA
# description: all comments in the input AST with their `text` attribute base64 decoded
comments_decoded := [decoded |
	some comment in input.comments
	decoded := object.union(comment, {"text": base64.decode(comment.text)})
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

	loc := util.to_location_object(comment.location)
	row := loc.row + 1

	rules := regex.split(`,\s*`, trim_space(regex.replace(comment.text, `^.*regal ignore:\s*(\S+)`, "$1")))
}

# METADATA
# description: |
#   returns an array of partitions, i.e. arrays containing all comments
#   grouped by their "blocks". only comments on the same column as the
#   one before is considered to be part of a block.
comment_blocks(comments) := blocks if {
	row_partitions := [partition |
		rows := [row |
			some comment in comments
			row := util.to_location_object(comment.location).row
		]
		breaks := _splits(rows)

		some j, k in breaks
		partition := array.slice(
			comments,
			breaks[j - 1] + 1,
			k + 1,
		)
	]

	blocks := [block |
		some row_partition in row_partitions
		some block in {col: partition |
			some comment in row_partition
			col := util.to_location_object(comment.location).col

			partition := [c |
				some c in row_partition
				util.to_location_object(c.location).col == col
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
