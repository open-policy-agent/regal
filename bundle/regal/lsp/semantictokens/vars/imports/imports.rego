# METADATA
# description: |
#   Helper package for semantictokens that returns imports
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.semantictokens.vars.imports

import data.regal.util

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA:
# description: |
#   Extract import keywords, and the imported identifier (last term in path or alias)
result contains token if {
	some import_statement in module.imports

	keyword_location := util.to_location_object(import_statement.location)
	line := keyword_location.row - 1

	some token in [
		{"line": line, "col": keyword_location.col - 1, "length": 6, "type": 3}, # "import" keyword
		_identifer(import_statement, line), # identifier (last term in path or alias)
	]
}

_identifer(import_statement, line) := token if {
	not import_statement.alias

	identifier := regal.last(import_statement.path.value)
	identifier_location := util.to_location_object(identifier.location)

	token := {
		"line": line,
		"col": identifier_location.col - 1,
		"length": identifier_location.end.col - identifier_location.col,
		"type": 2,
	}
}

_identifer(import_statement, line) := token if {
	import_statement.alias

	token := {
		"line": line,
		"col": indexof(input.regal.file.lines[line], " as ") + 4,
		"length": count(import_statement.alias),
		"type": 2,
	}
}
