# messy-rule

**Summary**: Messy incremental rule

**Category**: Style

**Avoid**

```rego
package policy

allow if something

unrelated_rule if { # <--- this rule is breaking up allow
    # ...
}

allow if something_else # <--- should be with the first allow
```

**Prefer**

```rego
package policy

allow if something

allow if something_else

unrelated_rule if {
    # ...
}
```

## Rationale

In Rego, rules can be formed of many 'rule heads', partial definitions covering specific cases which together make up
the behaviour of the whole rule. Rules that are defined incrementally should have their definitions grouped together, as
this makes the code easier to follow. While this is mostly a style preference, having incremental rules grouped also
allows editors like VS Code to "know" that the rules belong together, allowing them to be smarter when displaying the
symbols of a workspace.

## Configuration Options

This linter rule provides the following [configuration options](https://www.openpolicyagent.org/projects/regal/configuration):

```yaml title=".regal/config.yaml or .regal.yaml"
rules:
  style:
    messy-rule:
      # one of "error", "warning", "ignore"
      level: error
      # whether to allow default rules to be placed apart from
      # the other definitions of the same rule, for example to
      # have all default rules grouped at the top of the policy
      allow-separate-defaults: false
```

With `allow-separate-defaults` set to `true`, the following policy is not flagged:

```rego
package policy

default allow := false

default reason := ""

allow if input.user.admin

reason := "user is admin" if input.user.admin
```

Note that default rules still break up the grouping of other rules, so a default rule placed between two definitions of
another rule will still have those definitions reported.

## Related Resources

- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/style/messy-rule/messy_rule.rego)
