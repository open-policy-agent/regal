# textDocument/codeAction: diagnostic

## Given

A policy that isn't properly formatted:

#### policy.rego

```rego
           package policy
```

## When

The client requests code actions for the document, and includes diagnostics information in the request:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "range": {
    "start": {
      "line": 0,
      "character": 25
    },
    "end": {
      "line": 0,
      "character": 25
    }
  },
  "context": {
    "diagnostics": [
      {
        "code": "opa-fmt",
        "message": "Format using opa-fmt",
        "range": {
          "start": {
            "line": 0,
            "character": 25
          },
          "end": {
            "line": 0,
            "character": 25
          }
        }
      }
    ]
  }
}
```

## Then

The response includes:

- A quickfix action for ignoring the `opa-fmt` rule in Regal's config
- A quickfix action for applying the `opa-fmt` fixer
- A source action for exploring compiler stages (although not directly related to the diagnostic)

#### output.json

```json
[
  {
    "command": {
      "arguments": [
        "{\"diagnostic\":{\"code\":\"opa-fmt\",\"message\":\"Format using opa-fmt\",\"range\":{\"end\":{\"character\":25,\"line\":0},\"start\":{\"character\":25,\"line\":0}}}}"
      ],
      "command": "regal.config.disable-rule",
      "title": "Ignore this rule in config",
      "tooltip": "Ignore this rule in config"
    },
    "diagnostics": [
      {
        "code": "opa-fmt",
        "message": "Format using opa-fmt",
        "range": {
          "end": {
            "character": 25,
            "line": 0
          },
          "start": {
            "character": 25,
            "line": 0
          }
        }
      }
    ],
    "isPreferred": false,
    "kind": "quickfix",
    "title": "Ignore this rule in config"
  },
  {
    "command": {
      "arguments": ["{\"target\":\"file:///workspace/policy.rego\"}"],
      "command": "regal.fix.opa-fmt",
      "title": "Format using opa-fmt",
      "tooltip": "Format using opa-fmt"
    },
    "diagnostics": [
      {
        "code": "opa-fmt",
        "message": "Format using opa-fmt",
        "range": {
          "end": {
            "character": 25,
            "line": 0
          },
          "start": {
            "character": 25,
            "line": 0
          }
        }
      }
    ],
    "isPreferred": true,
    "kind": "quickfix",
    "title": "Format using opa-fmt"
  },
  {
    "command": {
      "arguments": [
        {
          "format": true,
          "target": "file:///workspace/policy.rego"
        }
      ],
      "command": "regal.explorer",
      "title": "Explore compiler stages for this policy",
      "tooltip": "Explore compiler stages for this policy"
    },
    "kind": "source.explore",
    "title": "Explore compiler stages for this policy"
  }
]
```
