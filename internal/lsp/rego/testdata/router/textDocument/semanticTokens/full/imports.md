# textDocument/semanticTokens/full: imports

## Given

A policy that contains imports:

#### policy.rego

```rego
package regal.woo

import data.regal.ast
import data.regal.util as uuuuutil
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

The server provides semantic tokens for the imports:

#### output.json

```json
{
  "data": [
    0, 0, 7, 3, 0, 0, 14, 3, 0, 0, 2, 0, 6, 3, 0, 0, 18, 3, 2, 0, 1, 0, 6, 3, 0,
    0, 26, 8, 2, 0
  ]
}
```

See the language server specification for details about the semantic tokens
[response format](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens).
