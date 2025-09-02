# METADATA
# description: Handler for Code Actions
# related_resources:
#   - https://docs.styra.com/regal/language-server#code-actions
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_codeAction
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.codeaction
package regal.lsp.codeaction

import data.regal.lsp.clients

# METADATA
# entrypoint: true
result["response"] := actions

# METADATA
# description: A set of all code actions applicable in the current document
# scope: document

# METADATA
# description: Code actions for fixing reported diagnostics
actions contains action if {
	"quickfix" in only

	some diag in input.params.context.diagnostics

	[title, args] := rules[diag.code]
	action := {
		"title": title,
		"kind": "quickfix",
		"diagnostics": [diag],
		"isPreferred": true,
		"command": {
			"title": title,
			"command": concat("", ["regal.fix.", diag.code]),
			"tooltip": title,
			"arguments": [json.marshal(object.filter(
				{
					"target": input.params.textDocument.uri,
					"diagnostic": diag,
				},
				args,
			))],
		},
	}
}

# METADATA
# description: |
#  Code actions to show documentation for a linter rule. Note that this currently
#  only works for VSCode clients, via their `vscode.open` command. If we learn about
#  other clients that support this, we'll add them here.
actions contains action if {
	input.regal.client.identifier == clients.vscode
	"quickfix" in only

	some diag in input.params.context.diagnostics

	# always show the docs link
	title := concat("", ["Show documentation for ", diag.code])
	action := {
		"title": title,
		"kind": "quickfix",
		"diagnostics": [diag],
		"command": {
			"title": title,
			"command": "vscode.open",
			"tooltip": title,
			"arguments": [diag.codeDescription.href],
		},
	}
}

# METADATA
# description: |
#   Code action to explore the compiler stages for a policy. This is *source action*,
#   unrelated to diagnostics. Depends on the "vscode.open" command being available, and
#   therefore currently only works in VSCode clients.
actions contains action if {
	input.regal.client.identifier == clients.vscode

	strings.any_prefix_match("source.explore", only)

	document := trim_prefix(input.params.textDocument.uri, input.regal.environment.workspace_root_uri)
	explorer_url := concat("", [input.regal.environment.web_server_base_uri, "/explorer", document])
	action := {
		"title": "Explore compiler stages for this policy",
		"kind": "source.explore",
		"command": {
			"title": "Explore compiler stages for this policy",
			"command": "vscode.open",
			"tooltip": "Explore compiler stages for this policy",
			"arguments": [explorer_url],
		},
	}
}

# METADATA
# description: All code actions for fixing reported diagnostics
rules := {
	"opa-fmt": ["Format using opa-fmt", ["target"]],
	"use-rego-v1": ["Format for Rego v1 using opa fmt", ["target"]],
	"use-assignment-operator": ["Replace = with := in assignment", ["target", "diagnostic"]],
	"no-whitespace-comment": ["Format comment to have leading whitespace", ["target", "diagnostic"]],
	"non-raw-regex-pattern": ["Replace \" with ` in regex pattern", ["target", "diagnostic"]],
	"directory-package-mismatch": [
		"Move file so that directory structure mirrors package path",
		["target", "diagnostic"],
	],
}

# METADATA
# description: |
#   Any code action kinds to filter by, if provided in input. A kind may
#   be hierarchical — if only contains "source" it matches all source actions,
#   while "source.foo" matches only source actions with a "foo" prefix.
# scope: document
default only := [
	"quickfix",
	"source.explore",
]

only := input.params.context.only if count(input.params.context.only) > 0
