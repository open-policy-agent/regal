# textDocument/documentLink: ignore directive

## Given

A policy that contains an inline ignore directive:

#### policy.rego

```rego
# regal ignore:prefer-snake-case
package policy
```

And configuration that includes the `prefer-snake-case` rule:

#### data.json

```json
{
  "workspace": {
    "config": {
      "rules": {
        "style": {
          "prefer-snake-case": {
            "level": "error"
          }
        }
      }
    }
  }
}
```

(the rule does not need to be enabled — the config is only used to backtrack which category the rule belongs to)

## When

The client requests document links for the URI of the policy document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  }
}
```

## Then

The server provides a document link to the Regal documentation for the `prefer-snake-case` rule:

#### output.json

```json
[
  {
    "range": {
      "start": {
        "line": 0,
        "character": 15
      },
      "end": {
        "line": 0,
        "character": 32
      }
    },
    "target": "https://www.openpolicyagent.org/projects/regal/rules/style/prefer-snake-case",
    "tooltip": "See documentation for prefer-snake-case"
  }
]
```
