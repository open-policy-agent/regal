# textDocument/documentHighlight: metadata

## Given

A package with a title annotation in metadata:

#### policy.rego

```rego
# METADATA
# title: p
package p

```

## When

The client requests document highlights for the URI of the policy document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 0,
    "character": 4
  }
}
```

## Then

The response includes a document highlight for the range of `METADATA` and one for `title`:

#### output.json

```json
[
  {
    "kind": 1,
    "range": {
      "end": {
        "character": 7,
        "line": 1
      },
      "start": {
        "character": 2,
        "line": 1
      }
    }
  },
  {
    "kind": 1,
    "range": {
      "end": {
        "character": 10,
        "line": 0
      },
      "start": {
        "character": 2,
        "line": 0
      }
    }
  }
]
```
