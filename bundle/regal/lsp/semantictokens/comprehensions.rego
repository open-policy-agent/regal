package regal.lsp.semantictokens

import data.regal.ast

# METADATA
# description: Extract comprehension variable declarations from array/set/object comprehensions
comprehension_tokens.declaration contains var if {
	some comprehension in ast.found.comprehensions[_]
	comp_vars := _comprehension_vars(comprehension)
	some var in comp_vars
}

# METADATA
# description: Extract comprehension variable references in the output
comprehension_tokens.reference contains var if {
	some comprehension in ast.found.comprehensions[_]
	comp_vars := _comprehension_vars(comprehension)

	output_vars := array.flatten([
		_get_comprehension_key(comprehension),
		_get_comprehension_value(comprehension),
		_get_comprehension_vars(comprehension),
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
_get_comprehension_vars(comprehension) := [value |
	some expr in comprehension.value.body
	not expr.terms.symbols
	some value in expr.terms
	value.type == "var"
]
