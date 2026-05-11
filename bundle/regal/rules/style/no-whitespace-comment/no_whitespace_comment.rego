# METADATA
# description: Comment should start with whitespace
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/no-whitespace-comment
package regal.rules.style["no-whitespace-comment"]

import future.keywords.not

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	some location in ast.comments_decoded

	text := trim_prefix(location.text, "#")

	not regex.match(`^[\s#]*$|^#*[\s]+.*$`, text)
	not regex.match(config.rules.style["no-whitespace-comment"]["except-pattern"], text)
	not _excepted_shebang(location.row, text)

	violation := result.fail(rego.metadata.chain(), result.with_text(location))
}

# UNIX-style shebangs on the first line of the file should be exempted from the
# no-whitespace-comment rule, as the format of the shebang is imposed by the
# OS.
_excepted_shebang(row, text) if {
	row == 1

	# Note that the # will already have been consumed while parsing the AST.
	regex.match(`^[!].*`, text)
}
