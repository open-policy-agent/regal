# METADATA
# description: Comment should start with whitespace
package regal.rules.style["no-whitespace-comment"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	some comment in ast.comments_decoded

	not regex.match(`^[\s#]*$|^#*[\s]+.*$`, comment.text)
	not _excepted(comment.text)

	loc := util.to_location_object(comment.location)
	not _excepted_shebang(loc.row, comment.text)

	violation := result.fail(rego.metadata.chain(), result.location(comment))
}

# UNIX-style shebangs on the first line of the file should be exempted from the
# no-whitespace-comment rule, as the format of the shebang is imposed by the
# OS.
_excepted_shebang(row, text) if {
	row == 1

	# Note that the # will already have been consumed while parsing the AST.
	regex.match(`^[!].*`, text)
}

_excepted(text) if regex.match(config.rules.style["no-whitespace-comment"]["except-pattern"], text)
