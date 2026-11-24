# METADATA
# description: Common functions for finding AST elements at a given position
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.util.find

import rego.v2

import data.regal.ast
import data.regal.lsp.location
import data.regal.lsp.util.range

# METADATA
# description: |
#   find the function argument at the given position, if any, either where it is declared
#   or where it is used in the rule
arg_at_position := [arg, rule_index] if {
	[var, rule_index] := var_at_position

	not startswith(var.value, "$")

	some arg in data.workspace.parsed[input.params.textDocument.uri].rules[rule_index].head.args

	arg.value == var.value
	arg.type == "var"
}

# METADATA
# description: find the variable at input.params.position, and the index of its rule, if any
var_at_position := [var, rule_index] if {
	[rule, rule_index] := rule_at_position

	walk(rule, [_, var])

	var.type == "var"
	range.contains_position(range.parse(var.location), input.params.position)
}

# METADATA
# description: find the variable declared by assignment at input.params.position, if any
assign_var_at_position := [var, rule_index] if {
	[var, rule_index] := var_at_position

	not startswith(var.value, "$")

	ast.found.vars[rule_index].assign[_].value == var.value
}

# METADATA
# description: find the variables in terms positions in expressions at position
terms_var_at_position := [term, rule_index] if {
	some rule_index
	terms := ast.found.expressions[rule_index][_].terms

	terms[0].value[0].value != "assign" or not is_array(terms)

	some term in array.slice(terms, 1, 100)

	term.type == "var"
	not startswith(term.value, "$")

	range.contains_position(range.parse(term.location), input.params.position)
}

# METADATA
# description: find the rule at input.params.position if any
rule_at_position := [rule, rule_index] if {
	some rule_index, rule in data.workspace.parsed[input.params.textDocument.uri].rules

	range.contains_position(range.parse(rule.location), input.params.position)
}

# METADATA
# description: find the package whose name is at input.params.position, if any
package_at_position := pkg if {
	pkg := data.workspace.parsed[input.params.textDocument.uri].package

	range.contains_position(range.spanning(pkg.path), input.params.position)
}

# METADATA
# description: find the import statement at input.params.position, if any
import_at_position := imp if {
	some imp in data.workspace.parsed[input.params.textDocument.uri].imports

	range.contains_position(range.parse(imp.path.location), input.params.position)
}

# METADATA
# description: |
#   find the import statement at input.params.position, if the position is on its alias.
#   The alias has no location in the AST, so it is found in the text following the import path.
import_alias_at_position := imp if {
	some imp in data.workspace.parsed[input.params.textDocument.uri].imports

	imp.alias

	path_end := range.parse(imp.path.location).end

	input.params.position.line == path_end.line

	rest := substring(input.regal.file.lines[path_end.line], path_end.character, -1)
	start := path_end.character + count(regex.find_n(`^\s+as\s+`, rest, 1)[0])

	input.params.position.character >= start
	input.params.position.character <= start + count(imp.alias)
}

# METADATA
# description: |
#   find the `some`-declared variable at the given position, if any.
some_var_at_position := [var, rule_index] if {
	text := input.regal.file.lines[input.params.position.line]

	# `+ 1` converts LSP 0-based char position to word_at's 1-based column
	word := location.word_at(text, input.params.position.character + 1)

	some kind in ["some", "somein"]
	some rule_index, vars in ast.found.vars
	some var in vars[kind]

	var.value == word.text

	var_pos := range.parse(var.location)
	input.params.position.line == var_pos.start.line
	input.params.position.character >= var_pos.start.character
	input.params.position.character <= var_pos.end.character
}

# METADATA
# description: |
#   find the `every`-declared variable at the given position, if any.
every_var_at_position := [var, every_terms] if {
	text := input.regal.file.lines[input.params.position.line]

	# `+ 1` converts LSP 0-based char position to word_at's 1-based column
	word := location.word_at(text, input.params.position.character + 1)

	some every_blocks in ast.found.every
	some every_terms in every_blocks
	some kind in ["key", "value"]

	var := every_terms[kind]
	var.type == "var"
	var.value == word.text

	var_pos := range.parse(var.location)
	input.params.position.line == var_pos.start.line
	input.params.position.character >= var_pos.start.character
	input.params.position.character <= var_pos.end.character
}

# METADATA
# description: |
#   find the variable declared inside a comprehension at the
#   given position, if any.
comprehension_var_at_position := [var, comp] if {
	text := input.regal.file.lines[input.params.position.line]

	# `+ 1` converts LSP 0-based char position to word_at's 1-based column
	word := location.word_at(text, input.params.position.character + 1)

	some comps in ast.found.comprehensions
	some comp in comps
	some var in _comp_declared_vars(comp.value.body)

	var.value == word.text

	var_pos := range.parse(var.location)
	input.params.position.line == var_pos.start.line
	input.params.position.character >= var_pos.start.character
	input.params.position.character <= var_pos.end.character
}

# METADATA
# description: |
#   find all references that the given position is inside of, if any.
refs_at_position := [refs, rule_index] if {
	[_, rule_index] := rule_at_position

	refs := [ref |
		some ref in ast.found.refs[rule_index]
		ref_range := range.parse(ref.location)

		ref_range.start.line == input.params.position.line
		ref_range.start.character <= input.params.position.character
		ref_range.end.character >= input.params.position.character
	]
}

# METADATA
# description: |
#   like refs_at_position, but only returns the reference closest to the position.
ref_at_position := [ref, rule_index] if {
	[refs, rule_index] := refs_at_position

	# position may be inside of multiple refs, e.g:
	# data.fooooo.baa[input.x].aaaar
	# |               |   ^
	# here we're only interested in the ref closest
	# to position, ie. input.x in the example above
	[_, ref] := min([[distance, ref] |
		some ref in refs

		ref_range := range.parse(ref.location)
		distance := input.params.position.character - ref_range.start.character
	])
}

# METADATA
# description: |
#   find the variable inside of a ref at the given position, if any.
ref_var_at_position := [term, rule_index] if {
	[ref, rule_index] := ref_at_position

	walk(array.slice(ref.value, 1, 100), [_, term])

	term.type == "var"

	term_range := range.parse(term.location)

	input.params.position.character >= term_range.start.character
	input.params.position.character <= term_range.end.character
}

_comp_declared_vars(body) := [v |
	some expr in body
	some symbol in expr.terms.symbols
	some v in array.slice(symbol.value, 1, 100)
	v.type == "var"
	not startswith(v.value, "$")
]
