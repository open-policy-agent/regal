# METADATA
# description: functions to render markdown documentation from templates
package regal.lsp.template

# METADATA
# description: renders compact markdown documentation for the provided built-in function
render_for_builtin(builtin) := content if {
	category := _category(builtin)
	builtin_safe := _to_safe_builtin(builtin)

	content := replace(
		strings.render_template(_builtin_template, {
			"builtin": builtin_safe,
			"category": category,
			"link": _docs_link(builtin_safe, category),
			"snippet": _example_snippet(builtin_safe),
		}),
		"<bt>", "`",
	)
}

_docs_link(builtin, category) := link if {
	count(builtin.categories) > 0

	link := [substring(bc, 4, -1) |
		some bc in builtin.categories
		startswith(bc, "url=")
	][0]
} else := sprintf("https://www.openpolicyagent.org/docs/policy-reference/#builtin-%s-%s", [
	category,
	replace(builtin.name, ".", "-"),
])

_builtin_template := `### [{{ .builtin.name }}]({{ .link }})

{{ .snippet }}

{{ .builtin.description }}

{{ if (gt (len .builtin.decl.args) 0) -}}
#### Arguments
{{ range $arg := .builtin.decl.args }}
- <bt>{{ $arg.name }}<bt> {{ $arg.type }}: {{ $arg.description -}}
{{ end }}
{{ end }}
Returns <bt>{{ .builtin.decl.result.name }}<bt> of type <bt>{{ .builtin.decl.result.type }}<bt>: {{
   .builtin.decl.result.description
}}
`

_category(builtin) := builtin.categories[0] if {
	count(builtin.categories) > 0
} else := substring(builtin.name, 0, i) if {
	count(builtin.categories) == 0
	i := indexof(builtin.name, ".")
	i != -1
} else := builtin.name

_example_snippet(builtin) := snippet if {
	args := [arg.name | some arg in builtin.decl.args]
	snippet := sprintf("```rego\n%s := %s(%s)\n```", [
		builtin.decl.result.name,
		builtin.name,
		concat(", ", args),
	])
}

# here to work around the **extremely** annoying behavior of strings.render_template
# where missing keys are treated as fatal errors instead of giving template authors a
# chance to handle this: https://github.com/open-policy-agent/opa/issues/7931
_to_safe_builtin(builtin) := safe if {
	safe_attributes := {
		"description": "(no description)",
		"categories": [],
		"decl": {
			"args": [],
			"result": {
				"name": "output",
				"description": "",
			},
			"type": "function",
		},
	}

	merged := object.union(safe_attributes, builtin)
	safe := object.union(merged, {"decl": {"args": _safe_args(merged.decl.args)}})
}

_safe_args(args) := [_to_safe_arg(i, arg) | some i, arg in args]

_to_safe_arg(i, arg) := arg if {
	_safe_arg(arg)
} else := object.union(
	{
		"name": ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"][i],
		"description": "(no description)",
	},
	arg,
)

_safe_arg(arg) if {
	arg.name
	arg.description
}
