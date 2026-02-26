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
	comprehensions := [c |
		c := ast.found.comprehensions[_][_]
	]
	some comprehension in comprehensions
	comp_vars := _comprehension_vars(comprehension)
	some var in comp_vars
}

# METADATA
# description: Extract comprehension variable references in the output
result.reference contains var if {
	comprehensions := [c |
		c := ast.found.comprehensions[_][_]
	]
	some comprehension in comprehensions
	comp_vars := _comprehension_vars(comprehension)

	output_vars := array.flatten([
		_get_comprehension_key(comprehension),
		_get_comprehension_value(comprehension),
		_get_comprehension_vars(comprehension.value.body),
	])
	some var in output_vars
	var.type == "var"
	var.value in {v.value | some v in comp_vars}
}

# Helper function to get declared variables in a comprehension
_comprehension_vars(comprehension) := comp_vars if {
	comprehension.type in ["arraycomprehension", "setcomprehension", "objectcomprehension"]
	comp_vars := {v |
		some term in comprehension.value.body
		term.terms.symbols
		some symbol in term.terms.symbols
		some v in array.slice(symbol.value, 1, count(symbol.value) - 1)
		v.type == "var"
	}
}

# Helper to get variables from differing comprehensions
_get_comprehension_value(comprehension) := value if {
	comprehension.type in ["arraycomprehension", "setcomprehension"]
	value := comprehension.value.term
} else := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.value
} else := set()

# Helper to get variables from differing comprehensions
_get_comprehension_key(comprehension) := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.key
} else := set()

# Helper to get variables from differing comprehensions
_get_comprehension_vars(body) := [value |
	some expr in body
	not expr.terms.symbols
	some value in expr.terms
	value.type == "var"
]
