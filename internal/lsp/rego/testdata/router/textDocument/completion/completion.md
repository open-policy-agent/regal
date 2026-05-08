# textDocument/completion

## Given

A package with a title annotation in metadata:

#### policy.rego

```rego
package policy

value := i
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

The response includes all possible completions at the given position:

#### output.json

```json
{
  "isIncomplete": true,
  "items": [
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "indexof",
      "textEdit": {
        "newText": "indexof",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "indexof_n",
      "textEdit": {
        "newText": "indexof_n",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "intersection",
      "textEdit": {
        "newText": "intersection",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode",
      "textEdit": {
        "newText": "io.jwt.decode",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.decode_verify",
      "textEdit": {
        "newText": "io.jwt.decode_verify",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign",
      "textEdit": {
        "newText": "io.jwt.encode_sign",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.encode_sign_raw",
      "textEdit": {
        "newText": "io.jwt.encode_sign_raw",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_eddsa",
      "textEdit": {
        "newText": "io.jwt.verify_eddsa",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es256",
      "textEdit": {
        "newText": "io.jwt.verify_es256",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es384",
      "textEdit": {
        "newText": "io.jwt.verify_es384",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_es512",
      "textEdit": {
        "newText": "io.jwt.verify_es512",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs256",
      "textEdit": {
        "newText": "io.jwt.verify_hs256",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs384",
      "textEdit": {
        "newText": "io.jwt.verify_hs384",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_hs512",
      "textEdit": {
        "newText": "io.jwt.verify_hs512",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps256",
      "textEdit": {
        "newText": "io.jwt.verify_ps256",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps384",
      "textEdit": {
        "newText": "io.jwt.verify_ps384",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_ps512",
      "textEdit": {
        "newText": "io.jwt.verify_ps512",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs256",
      "textEdit": {
        "newText": "io.jwt.verify_rs256",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs384",
      "textEdit": {
        "newText": "io.jwt.verify_rs384",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "io.jwt.verify_rs512",
      "textEdit": {
        "newText": "io.jwt.verify_rs512",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_array",
      "textEdit": {
        "newText": "is_array",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_boolean",
      "textEdit": {
        "newText": "is_boolean",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_null",
      "textEdit": {
        "newText": "is_null",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_number",
      "textEdit": {
        "newText": "is_number",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_object",
      "textEdit": {
        "newText": "is_object",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_set",
      "textEdit": {
        "newText": "is_set",
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
    },
    {
      "data": "builtins",
      "detail": "built-in function",
      "kind": 3,
      "label": "is_string",
      "textEdit": {
        "newText": "is_string",
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
