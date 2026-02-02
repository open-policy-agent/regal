# METADATA
# description: Line too long
package regal.rules.style["line-length"]

import data.regal.config
import data.regal.result

report contains violation if {
	max_line_length := object.get(config.rules, ["style", "line-length", "max-line-length"], 120)

	some i, line in input.regal.file.lines

	line != ""

	line_length := count(line)
	line_length > max_line_length

	not _has_word_above_threshold(line)

	violation := result.fail(
		rego.metadata.chain(),
		{"location": {
			"file": input.regal.file.name,
			"row": i + 1,
			"col": 1,
			"text": line,
			"end": {
				"row": i + 1,
				"col": line_length,
			},
		}},
	)
}

_has_word_above_threshold(line) if {
	threshold := config.rules.style["line-length"]["non-breakable-word-threshold"]

	some word in split(line, " ")
	count(word) > threshold
}
