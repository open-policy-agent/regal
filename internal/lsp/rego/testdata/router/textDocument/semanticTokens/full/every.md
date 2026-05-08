# textDocument/semanticTokens/full: every

## Given

A policy with every constructs:

#### policy.rego

```rego
package regal.woo

every_two_vars_construct if {
    every k, v in input.object {
        is_string(k)
        v > 0
    }
}

every_one_var_construct if {
    every k in input.object {
        is_string(k)
    }
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

The server provides semantic tokens for the every constructs:

#### output.json

```json
{
  "data": [
    0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 3, 10, 1, 1, 2, 0, 3, 1, 1, 2, 1, 18, 1, 1,
    4, 1, 8, 1, 1, 4, 5, 10, 1, 1, 2, 1, 18, 1, 1, 4
  ]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
