# textDocument/semanticTokens/full: full policy

## Given

A policy a "full" policy:

#### policy.rego

```rego
package regal.woo

test_function(param1) := result if {
    calc1 := param1 * 2
    calc2 := param2 + 10
    result := calc1 + calc2

    calc3 := 1
    calc3 == param1
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

The server provides all semantic tokens for the document:

#### output.json

```json
{
  "data": [
    0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 2, 14, 6, 1, 1, 1, 13, 6, 1, 4, 5, 13, 6, 1,
    4
  ]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
