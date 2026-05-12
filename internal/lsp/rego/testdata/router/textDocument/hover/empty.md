# textDocument/hover: empty response

## Given

A policy that contains nothing to hover over:

#### policy.rego

```rego
package policy
```

## When

The client requests hover information at the package name:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 0,
    "character": 10
  }
}
```

## Then

The server responds with `null`.

#### output.json

```json
null
```
