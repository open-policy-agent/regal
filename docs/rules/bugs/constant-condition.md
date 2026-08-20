# constant-condition

**Summary**: Constant condition

**Category**: Bugs

**Automatically fixable**: [Yes](https://www.openpolicyagent.org/projects/regal/fixing)

**Avoid**
```rego
package policy

allow if {
    1 == 1
}
```

**Prefer**
```rego
package policy

allow := true
```

## Rationale

While most often a mistake, constant conditions are sometimes used as placeholders, or "TODO logic". While this is
harmless, it has no place in production policy, and should be replaced or removed before deployment.

Note that an operand of an `and`/`or` expression is only reported when the *whole* expression is constant, as in
`1 or 2`. A constant operand of an otherwise meaningful expression — like the `1` in `input.a and 1` — is left alone,
since removing it would leave the enclosing expression without an operand.

## Configuration Options

This linter rule provides the following configuration options:

```yaml
rules:
  bugs:
    constant-condition:
      # one of "error", "warning", "ignore"
      level: error
```

## Related Resources

- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/bugs/constant-condition/constant_condition.rego)
