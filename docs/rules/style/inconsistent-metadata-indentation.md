# inconsistent-metadata-indentation

**Summary**: Inconsistent indentation of multi-line metadata

**Category**: Style

**Avoid**
```rego
package authz

# METADATA
# description: |
#   allowed to read everything
read if input.admin

# METADATA
# description: |
#  allowed to write everything
write if input.admin
```

**Prefer**
```rego
package authz

# METADATA
# description: |
#   allowed to read everything
read if input.admin

# METADATA
# description: |
#   allowed to write everything
write if input.admin
```

## Rationale

The content of a multi-line metadata string — the lines below a `description: |` entry — is indented to a depth of your
choosing. What reads well is less important than it being the same everywhere, as a file where one annotation indents
its string content with two spaces and the next one with three looks like a mistake rather than a choice.

Two spaces is the depth the rule expects, and one file uses it per project, so the default is the same everywhere.
Projects that indent metadata content differently can say so with `indent-num-spaces`. Nested structures, like the
items of `related_resources`, are indented a level deeper by definition, so the step of a nested string is measured
from the key it belongs to, and not from the strings at the top of the block.

## Configuration Options

This linter rule provides the following [configuration options](https://www.openpolicyagent.org/projects/regal/configuration):

```yaml title=".regal/config.yaml or .regal.yaml"
rules:
  style:
    inconsistent-metadata-indentation:
      # one of "error", "warning", "ignore"
      level: error
      # spaces a multi-line metadata string is indented by, relative to its key
      indent-num-spaces: 2
```

## Related Resources

- OPA Docs: [Annotations](https://www.openpolicyagent.org/docs/policy-language/#annotations)
- GitHub: [Source Code](https://github.com/open-policy-agent/regal/blob/main/bundle/regal/rules/style/inconsistent-metadata-indentation/inconsistent_metadata_indentation.rego)
