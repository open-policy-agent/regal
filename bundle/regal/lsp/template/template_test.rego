package regal.lsp.template_test

import data.regal.lsp.template

test_template_eopa_builtin if {
	builtin := {
		"name": "neo4j.query",
		"description": "Returns results for the given neo4j query.",
		"categories": ["url=https://docs.styra.com/enterprise-opa/reference/built-in-functions/neo4j"],
		"decl": {
			"args": [{
				"description": "query object",
				"dynamic": {
					"key": {"type": "string"},
					"value": {"type": "any"},
				},
				"name": "request",
				"type": "object",
			}],
			"result": {
				"description": "response object",
				"dynamic": {
					"key": {"type": "any"},
					"value": {"type": "any"},
				},
				"name": "response",
				"type": "object",
			},
			"type": "function",
		},
		"nondeterministic": true,
	}

	template.render_for_builtin(builtin)
}

test_template_eopa_old_builtin if {
	builtin := {
		"name": "neo4j.query",
		"decl": {
			"args": [{
				"dynamic": {
					"key": {"type": "string"},
					"value": {"type": "any"},
				},
				"type": "object",
			}],
			"result": {
				"dynamic": {
					"key": {"type": "any"},
					"value": {"type": "any"},
				},
				"type": "object",
			},
			"type": "function",
		},
		"nondeterministic": true,
	}

	template.render_for_builtin(builtin)
}

test_template_no_args_function if {
	builtin := {
		"name": "opa.runtime",
		"description": "Returns an object that describes the runtime environment where OPA is deployed.",
		"decl": {
			"result": {
				"description": "opa.runtime",
				"dynamic": {
					"key": {"type": "string"},
					"value": {"type": "any"},
				},
				"name": "output",
				"type": "object",
			},
			"type": "function",
		},
		"nondeterministic": true,
	}

	template.render_for_builtin(builtin)
}
