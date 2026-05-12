# textDocument/signatureHelp

## Given

A policy that contains a call to the built-in `regex.match` function:

#### policy.rego

```rego
package policy

allow if regex.match(`foo`, "bar")
```

## When

The client requests signature help at the position of the first argument of the `regex.match` call:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 2,
    "character": 21
  }
}
```

## Then

The server provides signature help information for the `regex.match` function (arg names and types):

#### output.json

```json
{
  "activeParameter": 0,
  "activeSignature": 0,
  "signatures": [
    {
      "activeParameter": 0,
      "documentation": "Matches a string against a regular expression.",
      "label": "regex.match(pattern: string, value: string) -\u003e boolean",
      "parameters": [
        {
          "documentation": "(pattern: string): regular expression",
          "label": "pattern: string"
        },
        {
          "documentation": "(value: string): value to match against `pattern`",
          "label": "value: string"
        }
      ]
    }
  ]
}
```
