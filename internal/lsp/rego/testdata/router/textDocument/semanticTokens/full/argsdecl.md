# textDocument/semanticTokens/full: function args declarations

## Given

A policy that contains a function declaration with arguments:

#### policy.rego

```rego
package regal.woo

test_function(param1, param2) := result if {
    true
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

The server provides semantic tokens for the function arguments declarations:

#### output.json

```json
{
  "data": [0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 2, 14, 6, 1, 1, 0, 8, 6, 1, 1]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
