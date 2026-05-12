# inlayHint/resolve

## Given

A client that supports inlay hints resolution:

#### data.json

```json
{
  "client": {
    "capabilities": {
      "textDocument": {
        "inlayHint": {
          "resolveSupport": {
            "properties": ["tooltip"]
          }
        }
      }
    }
  }
}
```

## When

The client requests an inlay hint to be resolved:

#### input.json

```json
{
  "label": "needle",
  "position": {
    "line": 10,
    "character": 28
  },
  "kind": 2,
  "paddingRight": true,
  "data": {
    "name": "needle",
    "type": "string",
    "description": "string to search"
  }
}
```

## Then

The server resolves the inlay hint using the `data` field to provide a tooltip:

#### output.json

```json
{
  "kind": 2,
  "label": "needle",
  "paddingRight": true,
  "position": {
    "character": 28,
    "line": 10
  },
  "tooltip": {
    "kind": "markdown",
    "value": "`needle` — string: string to search"
  }
}
```
