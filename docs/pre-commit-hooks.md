# Pre-Commit Hooks

[Pre-Commit](https://pre-commit.com) is a framework for managing and maintaining multi-language pre-commit hooks.
This allows running Regal automatically whenever (and as the name implied before) a Rego file is about to be committed.

To use Regal with pre-commit, add this to your `.pre-commit-config.yaml`

```yaml
- repo: https://github.com/open-policy-agent/regal
  rev: v0.7.0 # Use the ref you want to point at
  hooks:
    - id: regal-lint
  # -   id: ...
```

## Hooks Available

### `regal-lint`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Runs Regal against all staged `.rego` files, aborting the commit if any fail.

- requires the `go` build chain is installed and available on `$PATH`
- will build and install the tagged version of Regal in an isolated `GOPATH`
- ensures compatibility between versions

### `regal-lint-use-path`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Runs Regal against all staged `.rego` files, aborting the commit if any fail.

- requires the `regal` package is already installed and available on `$PATH`.

### `regal-download`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Runs Regal against all staged `.rego` files, aborting the commit if any fail.

- Downloads the `regal` binary from GitHub. The downloaded version follows the
  `rev` you pin in your `.pre-commit-config.yaml` when it points at a Regal
  release (e.g. `rev: v0.26.0` downloads the `v0.26.0` binary), so pinning the
  hook pins the Regal version too. If `rev` points at a branch or commit rather
  than a release, the latest release is downloaded instead.
- Set the `REGAL_VERSION` environment variable (e.g. `v0.26.0`) to override the
  version regardless of the pinned `rev`.

### `regal-fix`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Runs `regal fix` against all staged `.rego` files, applying any
auto-fixable rule violations in place. Use this alongside `regal-lint` when you
want the hook to repair style-level issues automatically rather than asking the
contributor to re-run `regal fix` themselves.

- requires the `go` build chain is installed and available on `$PATH`
- will build and install the tagged version of Regal in an isolated `GOPATH`
- ensures compatibility between versions

### `regal-fix-use-path`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Same as `regal-fix`, but uses the `regal` binary already on `$PATH`.

- requires the `regal` package is already installed and available on `$PATH`.

### `regal-fix-download`

![commit-msg hook](https://img.shields.io/badge/hook-pre--commit-informational?logo=git)

Same as `regal-fix`, but downloads the `regal` binary from GitHub instead of building or relying on `$PATH`.

- The downloaded version follows the `rev` you pin in your
  `.pre-commit-config.yaml` when it points at a Regal release, so pinning the
  hook pins the Regal version too. Falls back to the latest release otherwise.
- Set the `REGAL_VERSION` environment variable to override the version.
