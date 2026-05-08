# textDocument/semanticTokens/full: variable references

## Given

A policy that contains a variable references:

#### policy.rego

```rego
package regal.woo

test_function(param1) := result if {
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

The server provides semantic tokens for the variable references:

#### output.json

```json
{
  "data": [0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 2, 14, 6, 1, 1, 2, 13, 6, 1, 4]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
