# textDocument/semanticTokens/full: comprehensions

## Given

A policy with every comprehension type:

#### policy.rego

```rego
package regal.woo

array_comprehensions := [x |
    some i, x in [1, 2, 3]
    i == 2
]

set_comprehensions := {x |
    some i, x in [1, 2, 3]
    i == 2
}

object_comprehensions := {k: v |
    some k, v in [1, 2, 3]
    v == 2
}
```

## When

The client requests semantic tokens for the document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  }
}
```

## Then

The server provides semantic tokens for the comprehensions:

#### output.json

```json
{
  "data": [
    0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 2, 25, 1, 1, 4, 1, 9, 1, 1, 2, 0, 3, 1, 1, 2,
    1, 4, 1, 1, 4, 3, 23, 1, 1, 4, 1, 9, 1, 1, 2, 0, 3, 1, 1, 2, 1, 4, 1, 1, 4,
    3, 26, 1, 1, 4, 0, 3, 1, 1, 4, 1, 9, 1, 1, 2, 0, 3, 1, 1, 2, 1, 4, 1, 1, 4
  ]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
