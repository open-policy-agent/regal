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
      "label": "indexof",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "indexof_n",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "intersection",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode_verify",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign_raw",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_eddsa",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es256",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es384",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es512",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs256",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs384",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs512",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps256",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps384",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps512",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs256",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs384",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs512",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_array",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_boolean",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_null",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_number",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_object",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_set",
      "labelDetails": {
        "description": "built-in function"
      }
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_string",
      "labelDetails": {
        "description": "built-in function"
      }
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
