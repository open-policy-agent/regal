package regal.lsp.codeaction_test

import data.regal.lsp.client
import data.regal.lsp.codeaction

test_actions_reported_in_expected_format if {
	r := codeaction.actions
		with data.client.identifier as client.identifiers.generic
		with input.regal.environment.workspace_root_uri as "file:///irrelevant"
		with input.params.textDocument.uri as "policy.rego"
		with input.params.context.diagnostics as [_diagnostics["opa-fmt"], _diagnostics["use-assignment-operator"]]
		with input.params.context.only as ["quickfix"]

	r == {
		{
			"command": {
				"arguments": [json.marshal({
					"target": "policy.rego",
					"diagnostic": _diagnostics["use-assignment-operator"],
				})],
				"command": "regal.fix.use-assignment-operator",
				"title": "Replace = with := in assignment", "tooltip": "Replace = with := in assignment",
			},
			"diagnostics": [_diagnostics["use-assignment-operator"]],
			"isPreferred": true,
			"kind": "quickfix",
			"title": "Replace = with := in assignment",
		},
		{
			"command": {
				"arguments": ["{\"target\":\"policy.rego\"}"],
				"command": "regal.fix.opa-fmt",
				"title": "Format using opa-fmt", "tooltip": "Format using opa-fmt",
			},
			"diagnostics": [_diagnostics["opa-fmt"]],
			"isPreferred": true,
			"kind": "quickfix",
			"title": "Format using opa-fmt",
		},
		_ignore_rule(_diagnostics["use-assignment-operator"]),
		_ignore_rule(_diagnostics["opa-fmt"]),
	}
}

test_code_action_returned_for_every_linter[rule] if {
	some rule, _ in codeaction.rules

	r := codeaction.actions
		with data.client.identifier as client.identifiers.generic
		with input.regal.environment.workspace_root_uri as "file:///irrelevant"
		with input.params.textDocument.uri as "file:///irrelevant"
		with input.params.context.only as ["quickfix"]
		with input.params.context.diagnostics as [{
			"code": rule,
			"message": "irrelevant",
			"range": {},
		}]

	count(r) == 2
}

test_code_actions_specific_to_vscode_reported_on_client_match if {
	diagnostic := _diagnostics["use-assignment-operator"]

	r := codeaction.actions
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as [diagnostic]
		with input.params.context.only as ["quickfix"]
		with data.client.identifier as client.identifiers.vscode

	r == {
		{
			"title": "Replace = with := in assignment",
			"kind": "quickfix",
			"isPreferred": true,
			"command": {
				"arguments": [json.marshal({"target": "file:///workspace/policy.rego", "diagnostic": diagnostic})],
				"command": "regal.fix.use-assignment-operator",
				"title": "Replace = with := in assignment", "tooltip": "Replace = with := in assignment",
			},
			"diagnostics": [diagnostic],
		},
		{
			"title": "Show documentation for use-assignment-operator",
			"kind": "quickfix",
			"command": {
				"arguments": ["https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator"],
				"command": "vscode.open",
				"title": "Show documentation for use-assignment-operator",
				"tooltip": "Show documentation for use-assignment-operator",
			},
			"diagnostics": [diagnostic],
		},
		_ignore_rule(diagnostic),
	}
}

test_code_actions_only_quickfix if {
	diagnostic := _diagnostics["use-assignment-operator"]

	r := codeaction.actions
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as [diagnostic]
		with input.params.context.only as ["quickfix"]
		with data.client.identifier as client.identifiers.vscode

	r == {
		{
			"title": "Replace = with := in assignment",
			"kind": "quickfix",
			"isPreferred": true,
			"command": {
				"arguments": [json.marshal({"target": "file:///workspace/policy.rego", "diagnostic": diagnostic})],
				"command": "regal.fix.use-assignment-operator",
				"title": "Replace = with := in assignment", "tooltip": "Replace = with := in assignment",
			},
			"diagnostics": [diagnostic],
		},
		{
			"title": "Show documentation for use-assignment-operator",
			"kind": "quickfix",
			"command": {
				"arguments": ["https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator"],
				"command": "vscode.open",
				"title": "Show documentation for use-assignment-operator",
				"tooltip": "Show documentation for use-assignment-operator",
			},
			"diagnostics": [diagnostic],
		},
		_ignore_rule(diagnostic),
	}
}

test_code_actions_only_source if {
	r := codeaction.actions
		with data.client.identifier as client.identifiers.generic
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as []
		with input.params.context.only as ["source"]

	count(r) == 2

	some action in r

	action.title == "Explore compiler stages for this policy"
	action.kind == "source.explore"
	action.command.command == "regal.explorer"
	action.command.title == "Explore compiler stages for this policy"
	action.command.tooltip == "Explore compiler stages for this policy"
	count(action.command.arguments) == 1
	action.command.arguments[0].target == "file:///workspace/policy.rego"
}

test_code_actions_source_create_test if {
	r := codeaction.actions
		with data.client.identifier as client.identifiers.generic
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as []
		with input.params.context.only as ["source.createTest"]

	count(r) == 1

	some action in r

	action.title == "Create tests for this file"
	action.kind == "source.createTest"
	action.command.command == "regal.createTest"
	action.command.title == "Create tests for this file"
	action.command.tooltip == "Create test cases for all rules in this file"
	count(action.command.arguments) == 1
	action.command.arguments[0].target == "file:///workspace/policy.rego"
}

test_code_actions_source_explore_in_default if {
	r := codeaction.actions
		with data.client.identifier as client.identifiers.generic
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as []

	some action in r

	action.kind == "source.explore"
	action.command.command == "regal.explorer"
}

test_code_actions_empty_only_means_all if {
	r := codeaction.actions
		with input.regal.environment.workspace_root_uri as "file:///workspace"
		with input.params.textDocument.uri as "file:///workspace/policy.rego"
		with input.params.context.diagnostics as [_diagnostics["use-assignment-operator"]]
		with input.params.context.only as []
		with data.client.identifier as client.identifiers.vscode

	count(r) == 5
}

_diagnostics["opa-fmt"] := {
	"code": "opa-fmt",
	"message": "Use opa fmt to format this file",
	"range": {
		"start": {
			"line": 0,
			"character": 0,
		},
		"end": {
			"line": 0,
			"character": 1,
		},
	},
}

# Silly object.union only to appease the type checker, who for some reason thinks that
# this violates the schema — and only in the first test. We'll have to look into that later,
# as it does *not* do that. But given the schema is only checked by the test command, we can
# live with this workaround for now.
_diagnostics["use-assignment-operator"] := object.union(
	{
		"code": "use-assignment-operator",
		"message": "Use := instead of = for assignment",
		"range": {
			"start": {
				"line": 2,
				"character": 0,
			},
			"end": {
				"line": 2,
				"character": 1,
			},
		},
		"codeDescription": {"href": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator"},
	},
	{},
)

_ignore_rule(diagnostic) := {
	"title": "Ignore this rule in config",
	"kind": "quickfix",
	"isPreferred": false,
	"command": {
		"arguments": [json.marshal({"diagnostic": diagnostic})],
		"command": "regal.config.disable-rule",
		"title": "Ignore this rule in config",
		"tooltip": "Ignore this rule in config",
	},
	"diagnostics": [diagnostic],
}
