# inconsistent-rule-type

**Summary**: Inconsistent rule head value types

**Category**: Bugs

**Automatically fixable**: No

**Avoid**
```rego
package policy

import rego.v1

default hello := false

hello := "world" if input.message == "world"
```

**Prefer**
```rego
package policy

import rego.v1

default hello := ""

hello := "world" if input.message == "world"
```

## Rationale

When multiple definitions of the same complete rule (including default rules) assign values of incompatible
types—specifically mixing boolean values with non-boolean values such as strings, numbers, arrays, or objects—it
usually indicates an authoring bug or misunderstanding of rule evaluation semantics.

## Configuration Options

This linter rule provides the following [configuration options](https://www.openpolicyagent.org/projects/regal/configuration):

```yaml title=".regal/config.yaml or .regal.yaml"
rules:
  bugs:
    inconsistent-rule-type:
      # one of "error", "warning", "ignore"
      level: error
```

## Related Resources

- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/bugs/inconsistent-rule-type/inconsistent_rule_type.rego)
