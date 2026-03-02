package regal.lsp.template_test

import data.regal.lsp.template

test_template_common_builtin_function if {
	builtin := {
		"categories": ["strings"],
		"decl": {
			"args": [
				{
					"description": "search string",
					"name": "search",
					"type": "string",
				},
				{
					"description": "base string",
					"name": "base",
					"type": "string",
				},
			],
			"result": {
				"description": "result of the prefix check",
				"name": "result",
				"type": "boolean",
			},
			"type": "function",
		},
		"description": "Returns true if the search string begins with the base string.",
		"name": "startswith",
	}
	rendered := template.render_for_builtin(builtin)

	rendered == concat("\n", [
		`### [startswith](https://www.openpolicyagent.org/docs/policy-reference/#builtin-strings-startswith)`,
		"",
		"```rego",
		"result := startswith(search, base)",
		"```",
		"",
		"Returns true if the search string begins with the base string.",
		"",
		"#### Arguments",
		"",
		"- `search` string: search string",
		"- `base` string: base string",
		"",
		"Returns `result` of type `boolean`: result of the prefix check",
		"",
	])
}

test_template_eopa_builtin if {
	builtin := {
		"name": "neo4j.query",
		"description": "Returns results for the given neo4j query.",
		"categories": ["url=https://github.com/open-policy-agent/eopa/blob/main/docs/eopa/reference/built-in-functions/neo4j.md"],
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
