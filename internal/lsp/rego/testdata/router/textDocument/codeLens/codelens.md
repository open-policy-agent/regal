# textDocument/codeLens

## Given

A policy with a rule.

#### policy.rego

```rego
package policy

allow if {
    "admin" in input.user.roles
}
```

## When

The client requests code lenses for the document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  }
}
```

## Then

The server responds with Evaluate code lenses for both the package and the rule:

#### output.json

```json
[
  {
    "command": {
      "arguments": [
        "{\"path\":\"data.policy\",\"row\":1,\"target\":\"file:///workspace/policy.rego\"}"
      ],
      "command": "regal.eval",
      "title": "Evaluate"
    },
    "range": {
      "end": {
        "character": 7,
        "line": 0
      },
      "start": {
        "character": 0,
        "line": 0
      }
    }
  },
  {
    "command": {
      "arguments": [
        "{\"path\":\"data.policy.allow\",\"row\":3,\"target\":\"file:///workspace/policy.rego\"}"
      ],
      "command": "regal.eval",
      "title": "Evaluate"
    },
    "range": {
      "end": {
        "character": 1,
        "line": 4
      },
      "start": {
        "character": 0,
        "line": 2
      }
    }
  }
]
```
