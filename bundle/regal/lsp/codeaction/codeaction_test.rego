package regal.lsp.codeaction_test

import data.regal.lsp.clients
import data.regal.lsp.codeaction

test_actions_reported_in_expected_format if {
	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": clients.generic},
			"environment": {
				"web_server_base_uri": "http://irrelevant",
				"workspace_root_uri": "file:///irrelevant",
			},
		},
		"params": {
			"textDocument": {"uri": "policy.rego"},
			"context": {"diagnostics": [_diagnostics["opa-fmt"], _diagnostics["use-assignment-operator"]]},
		},
	}

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
	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": 0},
			"environment": {
				"web_server_base_uri": "http://irrelevant",
				"workspace_root_uri": "file:///irrelevant",
			},
		},
		"params": {
			"textDocument": {"uri": "policy.rego"},
			"context": {"diagnostics": [{
				"code": rule,
				"message": "irrelevant",
				"range": {},
			}]},
		},
	}
	count(r) == 2
}

test_code_actions_specific_to_vscode_reported_on_client_match if {
	diagnostic := _diagnostics["use-assignment-operator"]

	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": clients.vscode},
			"environment": {
				"web_server_base_uri": "http://localhost:8000",
				"workspace_root_uri": "file:///workspace",
			},
		},
		"params": {
			"textDocument": {"uri": "file:///workspace/policy.rego"},
			"context": {"diagnostics": [diagnostic]},
		},
	}
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
		{
			"title": "Explore compiler stages for this policy",
			"kind": "source.explore",
			"command": {
				"arguments": ["http://localhost:8000/explorer/policy.rego"],
				"command": "vscode.open",
				"title": "Explore compiler stages for this policy",
				"tooltip": "Explore compiler stages for this policy",
			},
		},
	}
}

test_code_actions_only_quickfix if {
	diagnostic := _diagnostics["use-assignment-operator"]

	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": clients.vscode},
			"environment": {
				"web_server_base_uri": "http://localhost:8000",
				"workspace_root_uri": "file:///workspace",
			},
		},
		"params": {
			"textDocument": {"uri": "file:///workspace/policy.rego"},
			"context": {
				"diagnostics": [diagnostic],
				# this is the only field different from the previous test
				"only": ["quickfix"],
			},
		},
	}

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
	diagnostic := _diagnostics["use-assignment-operator"]
	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": clients.vscode},
			"environment": {
				"web_server_base_uri": "http://localhost:8000",
				"workspace_root_uri": "file:///workspace",
			},
		},
		"params": {
			"textDocument": {"uri": "file:///workspace/policy.rego"},
			"context": {
				"diagnostics": [diagnostic],
				# this is the only field different from the previous test
				"only": ["source"],
			},
		},
	}

	r == {{
		"title": "Explore compiler stages for this policy",
		"kind": "source.explore",
		"command": {
			"arguments": ["http://localhost:8000/explorer/policy.rego"],
			"command": "vscode.open",
			"title": "Explore compiler stages for this policy",
			"tooltip": "Explore compiler stages for this policy",
		},
	}}
}

test_code_actions_empty_only_means_all if {
	diagnostic := _diagnostics["use-assignment-operator"]
	r := codeaction.actions with input as {
		"regal": {
			"client": {"identifier": clients.vscode},
			"environment": {
				"web_server_base_uri": "http://localhost:8000",
				"workspace_root_uri": "file:///workspace",
			},
		},
		"params": {
			"textDocument": {"uri": "file:///workspace/policy.rego"},
			"context": {
				"diagnostics": [diagnostic],
				"only": [],
			},
		},
	}

	count(r) == 4
}

_diagnostics["opa-fmt"] := {
	"code": "opa-fmt",
	"message": "Use opa fmt to format this file",
	"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 1}},
}

# Silly object.union only to appease the type checker, who for some reason thinks that
# this violates the schema — and only in the first test. We'll have to look into that later,
# as it does *not* do that. But given the schema is only checked by the test command, we can
# live with this workaround for now.
_diagnostics["use-assignment-operator"] := object.union(
	{
		"code": "use-assignment-operator",
		"message": "Use := instead of = for assignment",
		"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 1}},
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
