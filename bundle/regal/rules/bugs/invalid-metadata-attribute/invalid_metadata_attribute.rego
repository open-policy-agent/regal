# METADATA
# description: Invalid attribute in metadata annotation
package regal.rules.bugs["invalid-metadata-attribute"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some block in ast.comments.blocks

	regex.match(`^\s*METADATA`, block[0].text)

	some attribute, _ in yaml.unmarshal(concat("\n", [entry.text | some entry in array.slice(block, 1, 100)]))

	not attribute in ast.comments.metadata_attributes

	violation := result.fail(rego.metadata.chain(), result.location([line |
		some line in block
		startswith(trim_space(line.text), $"{attribute}:")
	][0]))
}
