# {{.NameOriginal}}

**Summary**: ADD DESCRIPTION HERE

**Category**: {{.Category | ToUpper}}

**Automatically fixable**: [Yes](https://www.openpolicyagent.org/projects/regal/fixing) / No

**Avoid**
```rego
package policy

# ... ADD CODE TO AVOID HERE
```

**Prefer**
```rego
package policy

# ... ADD CODE TO PREFER HERE
```

## Rationale

ADD RATIONALE HERE

## Configuration Options

This linter rule provides the following configuration options:

```yaml
rules:
  {{.Category}}:
    {{.NameOriginal}}:
      # one of "error", "warning", "ignore"
      level: error
```
