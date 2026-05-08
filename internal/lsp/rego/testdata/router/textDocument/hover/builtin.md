# textDocument/hover

## Given

A policy that contains a call to a built-in function:

#### policy.rego

```rego
package policy

foo := endswith("foobar", "bar")
```

## When

The client requests hover information at the position of the `endswith` call:

#### input.json

```json
{
  "textDocument": {
    "uri": "file:///workspace/policy.rego"
  },
  "position": {
    "line": 2,
    "character": 10
  }
}
```

## Then

The server provides hover information for the `endswith` built-in function.

#### output.json

````json
{
  "contents": {
    "kind": "markdown",
    "value": "### [endswith](https://www.openpolicyagent.org/docs/policy-reference/#builtin-strings-endswith)\n\n```rego\nresult := endswith(search, base)\n```\n\nReturns true if the search string ends with the base string.\n\n#### Arguments\n\n- `search` string — search string\n- `base` string — base string\n\n#### Returns `result` of type `boolean`: result of the suffix check"
  },
  "range": {
    "end": {
      "character": 15,
      "line": 2
    },
    "start": {
      "character": 7,
      "line": 2
    }
  }
}
````
