# textDocument/hover

## Given

A policy that contains an import.

#### policy.rego

```rego
package policy

import data.lib
```

## When

The client requests hover information at the position of the `import` keyword:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 2,
    "character": 3
  }
}
```

## Then

The server provides hover information with a link pointing to example usage of the `import` keyword.

#### output.json

```json
{
  "contents": {
    "kind": "markdown",
    "value": "[View Usage Examples](https://www.openpolicyagent.org/docs/policy-reference//keywords/import)\n\n"
  },
  "range": {
    "end": {
      "character": 5,
      "line": 2
    },
    "start": {
      "character": 0,
      "line": 2
    }
  }
}
```
