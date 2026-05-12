# completionItem/resolve: built-in function

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

The client requests a completion item for a built-in function to be resolved:

#### input.json

```json
{
  "label": "startswith",
  "kind": 3,
  "detail": "built-in function",
  "textEdit": {
    "range": {
      "start": { "line": 10, "character": 20 },
      "end": { "line": 10, "character": 30 }
    },
    "newText": "startswith"
  },
  "data": "builtins"
}
```

## Then

The server resolves the completion item using the `data` field to provide markdown documentation:

#### output.json

````json
{
  "detail": "built-in function",
  "documentation": {
    "kind": "markdown",
    "value": "### [startswith](https://www.openpolicyagent.org/docs/policy-reference/#builtin-strings-startswith)\n\n```rego\nresult := startswith(search, base)\n```\n\nReturns true if the search string begins with the base string.\n\n#### Arguments\n\n- `search` string: search string\n- `base` string: base string\n\nReturns `result` of type `boolean`: result of the prefix check\n"
  },
  "kind": 3,
  "label": "startswith",
  "textEdit": {
    "newText": "startswith",
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
````
