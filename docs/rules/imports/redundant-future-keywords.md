# redundant-future-keywords

**Summary**: Remove future keyword imports that are redundant in Rego v1

**Category**: Imports

**Automatically fixable**: No

## Notice: Requires Rego v1 capabilities

This rule reports violations only when Regal is configured to target OPA v1 or later. Configure the target OPA version
through Regal's [capabilities setting](https://www.openpolicyagent.org/projects/regal#capabilities).

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

With Rego v1 capabilities, keywords such as `contains`, `every`, `if`, and `in` are part of the language by default, so
their individual `future.keywords` imports are redundant. The `not`, `or`, and `and` keywords still require imports.
This rule leaves the wildcard `import future.keywords` alone because it also imports those remaining keywords.

## Configuration Options

```yaml title=".regal/config.yaml or .regal.yaml"
rules:
  imports:
    redundant-future-keywords:
      # one of "error", "warning", "ignore"
      level: error

capabilities:
  from:
    engine: opa
    version: v1.0.0
```

## Related Resources

- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/imports/redundant-future-keywords/redundant_future_keywords.rego)
