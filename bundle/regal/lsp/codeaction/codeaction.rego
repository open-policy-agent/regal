# METADATA
# description: Handler for Code Actions
# related_resources:
#   - https://www.openpolicyagent.org/projects/regal/language-server#code-actions
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_codeAction
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.codeaction
package regal.lsp.codeaction

import data.regal.lsp.client

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

	some diagnostic in input.params.context.diagnostics

	[title, args] := rules[diagnostic.code]
	action := {
		"title": title,
		"kind": "quickfix",
		"diagnostics": [diagnostic],
		"isPreferred": true,
		"command": {
			"title": title,
			"command": $"regal.fix.{diagnostic.code}",
			"tooltip": title,
			"arguments": [json.marshal(object.filter(
				{
					"target": input.params.textDocument.uri,
					"diagnostic": diagnostic,
				},
				args,
			))],
		},
	}
}

# METADATA
# description: Generic code action to ignore any rule in config from diagnostics
actions contains action if {
	"quickfix" in only

	some diagnostic in input.params.context.diagnostics

	action := {
		"title": "Ignore this rule in config",
		"kind": "quickfix",
		"diagnostics": [diagnostic],
		"isPreferred": false,
		"command": {
			"title": "Ignore this rule in config",
			"command": "regal.config.disable-rule",
			"tooltip": "Ignore this rule in config",
			"arguments": [json.marshal({"diagnostic": diagnostic})],
		},
	}
}

# METADATA
# description: |
#  Code actions to show documentation for a linter rule. Note that this currently
#  only works for VSCode clients, via their `vscode.open` command. If we learn about
#  other clients that support this, we'll add them here.
actions contains action if {
	client.identifier == client.identifiers.vscode
	"quickfix" in only

	some diagnostic in input.params.context.diagnostics

	# always show the docs link
	title := $"Show documentation for {diagnostic.code}"
	action := {
		"title": title,
		"kind": "quickfix",
		"diagnostics": [diagnostic],
		"command": {
			"title": title,
			"command": "vscode.open",
			"tooltip": title,
			"arguments": [diagnostic.codeDescription.href],
		},
	}
}

# METADATA
# description: |
#   Code action to explore compiler stages for all LSP clients.
#   Invokes the regal.explorer command which handles client-specific behavior.
actions contains action if {
	strings.any_prefix_match("source.explore", only)

	action := {
		"title": "Explore compiler stages for this policy",
		"kind": "source.explore",
		"command": {
			"title": "Explore compiler stages for this policy",
			"command": "regal.explorer",
			"tooltip": "Explore compiler stages for this policy",
			"arguments": [{"target": input.params.textDocument.uri, "format": true}],
		},
	}
}

# METADATA
# description: |
#   Code action to create a test from the current rule evaluation state.
actions contains action if {
	strings.any_prefix_match("source.createTest", only)

	action := {
		"title": "Create tests for this file",
		"kind": "source.createTest",
		"command": {
			"title": "Create tests for this file",
			"command": "regal.createTest",
			"tooltip": "Create test cases for all rules in this file",
			"arguments": [{
				"target": input.params.textDocument.uri,
				"row": input.params.range.start.line,
			}],
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
	"prefer-equals-comparison": ["Replace = with == in comparison", ["target", "diagnostic"]],
	"constant-condition": ["Remove constant condition", ["target", "diagnostic"]],
	"redundant-existence-check": ["Remove redundant existence check", ["target", "diagnostic"]],
}

# METADATA
# description: |
#   Any code action kinds to filter by, if provided in input. A kind may
#   be hierarchical — if only contains "source" it matches all source actions,
#   while "source.foo" matches only source actions with a "foo" prefix.
# scope: document
default only := ["quickfix", "source.explore", "source.createTest"]

only := input.params.context.only if input.params.context.only != []
