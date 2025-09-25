# METADATA
# description: provides completion suggestions for rules in the workspace
package regal.lsp.completion.providers.rulerefs

import data.regal.ast

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

_ref_is_internal(ref) if contains(ref, "._")

_line := input.regal.file.lines[input.params.position.line]

_word := location.ref_at(_line, input.params.position.character + 1)

_workspace_rule_refs contains ref if {
	some refs in data.workspace.defined_refs
	some ref in refs
}

_parsed_current_file := data.workspace.parsed[input.params.textDocument.uri]

_current_file_package := ast.ref_to_string(_parsed_current_file.package.path)

_current_file_imports contains ref if {
	some imp in _parsed_current_file.imports

	ref := ast.ref_to_string(imp.path.value)
}

_current_package_refs contains ref if {
	some ref in _workspace_rule_refs

	startswith(ref, _current_file_package)
}

_imported_package_refs contains ref if {
	some ref in _workspace_rule_refs

	not _ref_is_internal(ref)

	strings.any_prefix_match(ref, _current_file_imports)
}

_other_package_refs contains ref if {
	some ref in _workspace_rule_refs

	not ref in _imported_package_refs
	not ref in _current_package_refs

	not _ref_is_internal(ref)
}

# from the current package
_rule_ref_suggestions contains pkg_ref if {
	some ref in _current_package_refs

	pkg_ref := trim_prefix(ref, concat("", [_current_file_package, "."]))
}

# from imported packages
_rule_ref_suggestions contains pkg_ref if {
	some ref in _imported_package_refs
	some imported_package in _current_file_imports

	startswith(ref, imported_package)

	prefix := regex.replace(imported_package, `\.[^\.]+$`, "")
	pkg_ref := trim_prefix(ref, concat("", [prefix, "."]))
}

# from any other package
_rule_ref_suggestions contains ref if some ref in _other_package_refs

# also suggest the unimported packages themselves
# e.g. data.foo.rule will also generate data.foo as a suggestion
_rule_ref_suggestions contains regex.replace(ref, `\.[^\.]+$`, "") if some ref in _other_package_refs

_matching_rule_ref_suggestions contains ref if {
	_line != ""
	location.in_rule_body(_line)

	# \W is used here to match ( in the case of func() := ..., as well as the space in the case of rule := ...
	first_word := regex.split(`\W+`, trim_space(_line))[0]

	some ref in _rule_ref_suggestions

	startswith(ref, _word.text)

	# this is to avoid suggesting a recursive rule, e.g. rule := rule, or func() := func()
	ref != first_word
}

# METADATA
# description: set of completion suggestions for references to rules
items contains item if {
	some ref in _matching_rule_ref_suggestions

	item := {
		"label": ref,
		"kind": kind.variable,
		"detail": "reference",
		"textEdit": {
			"range": location.word_range(_word, input.params.position),
			"newText": ref,
		},
	}
}
