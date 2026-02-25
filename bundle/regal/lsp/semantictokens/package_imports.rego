package regal.lsp.semantictokens

# METADATA
# description: Extract import tokens - return only last term of the path
import_tokens contains last_term if {
	some import_statement in module.imports
	import_path := import_statement.path.value

	last_term := import_path[count(import_path) - 1]
}

# METADATA
# description: Extract package tokens - return full package path
package_tokens := module.package.path
