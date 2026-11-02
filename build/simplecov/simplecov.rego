# METADATA
# description: |
#   transforms OPA's JSON test coverage format to equivalent
#   simplecov JSON, to be used for Codecov reports, et. al.
package build.simplecov

# METADATA
# entrypoint: true
from_opa := {"coverage": _coverage}

_coverage[file] := {"lines": _to_lines(report)} if some file, report in input.files

_covered_map(report) := cm if {
	covered := object.get(report, "covered", [])
	cm := {line: 1 |
		some item in covered
		some line in numbers.range(item.start.row, item.end.row)
	}
}

_not_covered_map(report) := ncm if {
	not_covered := object.get(report, "not_covered", [])
	ncm := {line: 0 |
		some item in not_covered
		some line in numbers.range(item.start.row, item.end.row)
	}
}

_to_lines(report) := lines if {
	cm := _covered_map(report)
	ncm := _not_covered_map(report)
	keys := sort([line | some line, _ in object.union(cm, ncm)])
	last := keys[count(keys) - 1]

	lines := [value |
		some i in numbers.range(1, last)
		value := _to_value(cm, ncm, i)
	]
}

_to_value(cm, _, line) := 1 if cm[line]

_to_value(_, ncm, line) := 0 if ncm[line]

_to_value(cm, ncm, line) := null if {
	not cm[line]
	not ncm[line]
}

# METADATA
# description: |
#   utility rule to evaluate when only the lines not covered are of interest
#   invoke like:
#   regal test --coverage bundle \
#   | opa eval -f pretty -I -d build/simplecov/simplecov.rego 'data.build.simplecov.not_covered'
# entrypoint: true
not_covered[file] := info.not_covered if some file, info in input.files
