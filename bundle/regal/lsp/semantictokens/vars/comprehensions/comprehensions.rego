# METADATA
# description: |
#   Helper package for semantictokens that returns variable references and declarations in comprehensions
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.comprehensions

import data.regal.ast

# METADATA
# description: Extract comprehension variable declarations from array/set/object comprehensions
result.declaration contains var if {
	# regal ignore:prefer-some-in-iteration
	comprehensions := ast.found.comprehensions[_]
	some comprehension in comprehensions

	comp_vars := {v |
		some term in comprehension.value.body
		term.terms.symbols
		some symbol in term.terms.symbols
		some v in array.slice(symbol.value, 1, count(symbol.value) - 1)
		v.type == "var"
	}
	some var in comp_vars
}

# METADATA
# description: Extract comprehension variable references in the output
result.reference contains var if {
	# regal ignore:prefer-some-in-iteration
	comprehensions := ast.found.comprehensions[_]
	some comprehension in comprehensions

	output_vars := array.flatten([
		_comprehension_key(comprehension),
		_comprehension_value(comprehension),
		_comprehension_body_vars(comprehension.value.body),
	])
	some var in output_vars
	var.type == "var"
	var.value in {v.value | some v in result.declaration}
}

default _comprehension_value(_) := set()

# Helper to get variables from differing comprehensions
_comprehension_value(comprehension) := value if {
	comprehension.type in ["arraycomprehension", "setcomprehension"]
	value := comprehension.value.term
} else := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.value
}

default _comprehension_key(_) := set()

# Helper to get variables from differing comprehensions
_comprehension_key(comprehension) := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.key
}

# Helper to get variables from differing comprehensions
_comprehension_body_vars(body) := [value |
	some expr in body
	not expr.terms.symbols
	some value in expr.terms
	value.type == "var"
]
