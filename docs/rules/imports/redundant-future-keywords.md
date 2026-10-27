# redundant-future-keywords

**Summary**: Remove future keyword imports that are redundant in Rego v1

**Category**: Imports

**Automatically fixable**: No

## Avoid

```rego
package policy

import future.keywords.every

allow if {
    every role in input.roles {
        role == "reader"
    }
}
```

## Prefer

```rego
package policy

allow if {
    every role in input.roles {
        role == "reader"
    }
}
```

## Rationale

Rego v1 includes `contains`, `every`, `if`, and `in` by default, while `not`, `or`, and `and` remain available through
explicit or wildcard imports.

## Configuration Options

```yaml title=".regal/config.yaml or .regal.yaml"
rules:
  imports:
    redundant-future-keywords:
      # one of "error", "warning", "ignore"
      level: error

```

## Related Resources

- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/imports/redundant-future-keywords/redundant_future_keywords.rego)
- Regal: [Configure capabilities](https://www.openpolicyagent.org/projects/regal#capabilities)
