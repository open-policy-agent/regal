# textDocument/semanticTokens/full: some

## Given

A policy with `some` iteration:

#### policy.rego

```rego
package regal.woo

some_two_vars_construct if {
    some i, item in input.array
    i < 10
    item > 0
}

some_one_var_construct if {
    some i in input.array
    i < 10
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

The server provides semantic tokens for those:

#### output.json

```json
{
  "data": [
    0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 3, 9, 1, 1, 2, 0, 3, 4, 1, 2, 1, 4, 1, 1, 4,
    1, 4, 4, 1, 4, 4, 9, 1, 1, 2, 1, 4, 1, 1, 4
  ]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
