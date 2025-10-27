package regal.lsp.util.range_test

import data.regal.lsp.util.range

test_range_from_location if {
	location := {
		"row": 3,
		"col": 5,
		"end": {
			"row": 4,
			"col": 10,
		},
	}

	range.from_location(location) == {
		"start": {
			"line": 2,
			"character": 4,
		},
		"end": {
			"line": 3,
			"character": 9,
		},
	}
}

test_parse if range.parse("1:5:2:10") == {
	"start": {
		"line": 0,
		"character": 4,
	},
	"end": {
		"line": 1,
		"character": 9,
	},
}

test_range_contains_position_on_same_line if {
	rng := {
		"start": {"line": 3, "character": 6},
		"end": {"line": 3, "character": 10},
	}

	range.contains_position(rng, {"line": 3, "character": 7})
}

test_range_contains_position_multiple_lines if {
	rng := {
		"start": {"line": 3, "character": 6},
		"end": {"line": 5, "character": 20},
	}

	range.contains_position(rng, {"line": 4, "character": 0})
}

test_range_contains_position_last_line if {
	rng := {
		"start": {"line": 3, "character": 6},
		"end": {"line": 5, "character": 20},
	}

	range.contains_position(rng, {"line": 5, "character": 15})
}
