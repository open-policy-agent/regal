# textDocument/semanticTokens/full: package

## Given

A policy that contains a package declaration:

#### policy.rego

```rego
package regal.woo
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

The server provides semantic tokens for the package declaration:

#### output.json

```json
{
  "data": [0, 0, 7, 3, 0, 0, 14, 3, 0, 0]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
