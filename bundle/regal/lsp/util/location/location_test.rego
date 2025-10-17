package regal.lsp.util.location_test

import data.regal.lsp.util.location

test_within_range_same_line if {
	pos := {"line": 3, "character": 7}
	rng := {
		"start": {"line": 3, "character": 6},
		"end": {"line": 3, "character": 10},
	}

	location.within_range(pos, rng)
}

test_within_range_different_lines if {
	pos := {"line": 4, "character": 0}
	rng := {
		"start": {"line": 3, "character": 6},
		"end": {"line": 5, "character": 20},
	}

	location.within_range(pos, rng)
}
