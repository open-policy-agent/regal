# METADATA
# description: |
#   Returns location of variables to be highlighted via semantic tokens. Currently returns:
#     - declarations of function args in text documents
#     - variable references that are used in function calls
#     - variable references that are used in expressions
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# This is handling the case where the module from the parsed workspace is empty

default result.response := {}

# METADATA
# entrypoint: true
result.response := {
	"arg_tokens": arg_tokens,
	"package_tokens": package_tokens,
	"import_tokens": import_tokens,
	"comprehension_tokens": comprehension_tokens,
	"construct_tokens": construct_tokens,
}

# METADATA
# description: Extract import tokens - return only last term of the path
import_tokens contains last_term if {
	some import_statement in module.imports
	import_path := import_statement.path.value

	last_term := import_path[count(import_path) - 1]
}

# METADATA
# description: Extract function argument declarations
arg_tokens.declaration contains arg if {
	some rule in module.rules
	some arg in rule.head.args
	arg.type == "var"
}

# METADATA
# description: Extract variable references in function calls
arg_tokens.reference contains arg if {
	some rule in module.rules

	rule.head.args

	arg_names := {v.value | some v in rule.head.args}

	walk(rule.body, [_, expr])

	expr.terms[0].type == "ref"

	some arg in array.slice(expr.terms, 1, count(expr.terms))

	arg.type == "var"
	arg.value in arg_names
}

# METADATA
# description: Extract variable references in call expressions
arg_tokens.reference contains arg if {
	some rule in module.rules
	arg_names := {v.value | some v in rule.head.args}
	walk(rule.body, [_, expr])

	some term in expr.terms
	term.type == "call"

	some arg in term.value
	arg.type == "var"

	arg.value in arg_names
}

# METADATA
# description: Extract package tokens - return full package path
package_tokens := module.package.path

# METADATA
# description: Extract comprehension variable declarations from array/set/object comprehensions
comprehension_tokens.declaration contains var if {
	some rule in module.rules
	walk(rule.head.value, [_, comprehension])
	comp_vars := _comprehension_vars(comprehension)
	some var in comp_vars
}

# METADATA
# description: Extract comprehension variable references in the output
comprehension_tokens.reference contains var if {
	some rule in module.rules
	walk(rule, [_, comprehension])

	comprehension.type in ["arraycomprehension", "setcomprehension", "objectcomprehension"]
	comp_vars := _comprehension_vars(comprehension)

	output_vars := {_get_comprehension_key(comprehension), _get_comprehension_value(comprehension)}
	some var in output_vars
	var.type == "var"
	var.value in {v.value | some v in comp_vars}
}

# METADATA
# description: Extract comprehension variable references in the body
comprehension_tokens.reference contains var if {
	some rule in module.rules
	walk(rule, [_, comprehension])

	comprehension.type in ["arraycomprehension", "setcomprehension", "objectcomprehension"]
	comp_vars := _comprehension_vars(comprehension)

	walk(comprehension.value.body, [_, expr])
	some var in expr.terms
	var.type == "var"
	var.value in {v.value | some v in comp_vars}
}

# METADATA
# description: Extract variable declarations from every and some constructs
construct_tokens.declaration contains var if {
	some rule in module.rules
	walk(rule.body, [_, term])

	declared_vars := _get_construct_vars(term.terms)

	some var in declared_vars
}

# METADATA
# description: Extract variable references in every and some constructs
construct_tokens.reference contains var if {
	some rule in module.rules

	walk(rule.body, [_, declare_term])
	declared_vars := _get_construct_vars(declare_term)
	declared_vars != set()
	declared_var_names := {v.value | some v in declared_vars}

	walk(rule.body, [_, ref_term])
	values := _get_construct_reference_context(ref_term)
	values != []

	walk(values, [_, expr])
	some var in expr.terms
	var.type == "var"
	var.value in declared_var_names
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

# Helper to get variables from every/some constructs
_get_construct_vars(terms) := vars if {
	terms.domain
	terms.key
	terms.value
	vars := {terms.key, terms.value}
} else := vars if {
	terms.symbols
	not terms.body
	vars := {v |
		some symbol in terms.symbols
		some v in array.slice(symbol.value, 1, count(symbol.value) - 1)
		v.type == "var"
	}
} else := vars if {
	terms.domain
	terms.key == null
	terms.value
	vars := {terms.value}
} else := set()

# Helper to get the reference context based on construct type
_get_construct_reference_context(construct_terms) := values if {
	values := construct_terms.body
} else := values if {
	not construct_terms.symbols
	is_array(construct_terms.terms)
	values := [{"terms": construct_terms.terms}]
} else := []

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
