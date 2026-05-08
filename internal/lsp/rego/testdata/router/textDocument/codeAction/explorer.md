# textDocument/codeAction: explorer

## Given

Any valid policy.

#### policy.rego

```rego
package policy

```

## When

The client requests code actions for the document:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "range": {
    "start": {
      "line": 0,
      "character": 4
    },
    "end": {
      "line": 0,
      "character": 4
    }
  }
}
```

## Then

The response includes a source action for the OPA explorer command:

#### output.json

```json
[
  {
    "command": {
      "arguments": [
        {
          "format": true,
          "target": "file:///workspace/policy.rego"
        }
      ],
      "command": "regal.explorer",
      "title": "Explore compiler stages for this policy",
      "tooltip": "Explore compiler stages for this policy"
    },
    "kind": "source.explore",
    "title": "Explore compiler stages for this policy"
  }
]
```
