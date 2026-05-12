# textDocument/completion: item defaults

## Given

A package with a title annotation in metadata:

#### policy.rego

```rego
package policy

value := i
```

And a client that supports item defaults for edit ranges:

#### data.json

```json
{
  "client": {
    "capabilities": {
      "textDocument": {
        "completion": {
          "completionList": {
            "itemDefaults": ["editRange"]
          }
        }
      }
    }
  }
}
```

## When

The client requests completion suggestions at the position just after the `i`:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 2,
    "character": 10
  }
}
```

## Then

The response includes all possible completions at the given position, with the default edit range provided
only once, thus massively reducing the size of the completion response:

#### output.json

```json
{
  "isIncomplete": false,
  "itemDefaults": {
    "editRange": {
      "end": {
        "character": 10,
        "line": 2
      },
      "start": {
        "character": 9,
        "line": 2
      }
    }
  },
  "items": [
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "indexof"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "indexof_n"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "intersection"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode_verify"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign_raw"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_eddsa"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es256"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es384"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es512"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs256"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs384"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs512"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps256"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps384"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps512"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs256"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs384"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs512"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_array"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_boolean"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_null"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_number"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_object"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_set"
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_string"
    },
    {
      "data": "input",
      "detail": "input document",
      "kind": 14,
      "label": "input",
      "textEdit": {
        "newText": "input",
        "range": {
          "end": {
            "character": 10,
            "line": 2
          },
          "start": {
            "character": 9,
            "line": 2
          }
        }
      }
    }
  ]
}
```
