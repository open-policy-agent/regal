# textDocument/inlayHint

## Given

A policy that contains a call to a built-in function:

#### policy.rego

```rego
package policy

foo := startswith("foobar", "foo")
```

## When

The client requests inlay hints for the document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "range": {
    "start": {
      "line": 0,
      "character": 0
    },
    "end": {
      "line": 2,
      "character": 34
    }
  }
}
```

## Then

The server provides inlay hints for the two arguments of the `startswith` call:

#### output.json

```json
[
  {
    "kind": 2,
    "label": "base:",
    "paddingRight": true,
    "position": {
      "character": 28,
      "line": 2
    },
    "tooltip": {
      "kind": "markdown",
      "value": "`base` — string: base string"
    }
  },
  {
    "kind": 2,
    "label": "search:",
    "paddingRight": true,
    "position": {
      "character": 18,
      "line": 2
    },
    "tooltip": {
      "kind": "markdown",
      "value": "`search` — string: search string"
    }
  }
]
```
