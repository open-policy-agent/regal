# impossible-default

**Summary**: Rule defined unconditionally alongside default definition

**Category**: Bugs

**Automatically fixable**: No

**Avoid**
```rego
package policy

default username := "guest"

username := "regal"
```

**Prefer**
```rego
package policy

default username := "guest"

username := "admin" if input.user.is_admin
```

## Rationale

A `default` rule definition in Rego provides a fallback value when no other
definition of the rule evaluates successfully.

When a rule has both a `default` definition and an unconditional definition
written without conditions, that unconditional definition answers for every input.
As a result, the default definition can never be reached:

- if the unconditional definition assigns a different value, evaluation either
  produces the unconditional value or fails with an evaluation conflict error
  if multiple definitions collide;
- if the unconditional definition assigns the same value, the default definition
  is completely redundant.

In both cases, the `default` definition is dead code. This often happens when an
author forgets to attach conditions (`if ...`) to a rule definition, or leaves
behind an obsolete `default` after converting a rule to an unconditional constant.

## Known limitations

- Only definitions **within one file** are compared. Rules split across
  multiple files in the same package are not detected, as each file is linted
  independently.
- The unconditional definition must assign a **constant** value. An assignment
  like `username := input.user` can itself be undefined, in which case the
  `default` definition genuinely serves as the fallback when `input.user` is
  missing:

  ```rego
  default username := "guest"

  username := input.user
  ```

- Multi-value rules (`deny contains msg if ...`) cannot declare a `default` in
  Rego and are therefore excluded.
- Function definitions whose arguments are not all distinct variables are not
  reported. `f("admin") := "boss"` has no body, but applies only when the argument
  is `"admin"`, leaving `default f(_) := "guest"` to handle all other inputs.
- Rules whose head contains a variable (`config[k] := "loose"`) are not
  reported.

## Configuration Options

This linter rule provides the following configuration options:

```yaml
rules:
  bugs:
    impossible-default:
      # one of "error", "warning", "ignore"
      level: error
```

## Related Resources

- OPA Docs: [Default Keyword](https://www.openpolicyagent.org/docs/policy-language/#default-keyword)
- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/bugs/impossible-default/impossible_default.rego)

## Community

If you think you've found a problem with this rule or its documentation, would like to suggest improvements, new rules,
or just talk about Regal in general, please join us in the `#regal` channel in the OPA
[Slack](https://slack.openpolicyagent.org)!
