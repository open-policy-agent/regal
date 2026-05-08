# completionItem/resolve: input

## Given

A client that supports resolving completion items:

#### data.json

```json
{
  "client": {
    "capabilities": {
      "textDocument": {
        "completion": {
          "resolveSupport": {
            "properties": ["documentation", "detail"]
          }
        }
      }
    }
  }
}
```

## When

The client requests a completion item for `input` to be resolved:

#### input.json

```json
{
  "label": "input",
  "kind": 14,
  "detail": "input document",
  "textEdit": {
    "range": {
      "start": { "line": 10, "character": 20 },
      "end": { "line": 10, "character": 30 }
    },
    "newText": "input"
  },
  "data": "input"
}
```

## Then

The server resolves the completion item using the `data` field to provide markdown documentation:

#### output.json

```json
{
  "data": "input",
  "detail": "input document",
  "documentation": {
    "kind": "markdown",
    "value": "# input\n\n'input' refers to the input document being evaluated.\nIt is a special keyword that allows you to access the data sent to OPA at evaluation time.\n\nTo see more examples of how to use 'input', check out the\n[policy language documentation](https://www.openpolicyagent.org/docs/policy-language/).\n\nYou can also experiment with input in the [Rego Playground](https://play.openpolicyagent.org/).\n"
  },
  "kind": 14,
  "label": "input",
  "textEdit": {
    "newText": "input",
    "range": {
      "end": {
        "character": 30,
        "line": 10
      },
      "start": {
        "character": 20,
        "line": 10
      }
    }
  }
}
```
