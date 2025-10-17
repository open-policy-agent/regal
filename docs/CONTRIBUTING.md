# Development

If you'd like to contribute to Regal, here are some pointers to help get you started.

Before you start, the [architecture](./architecture) guide provides a useful overview of how Regal works, so you might
want to read that before diving into the code!

## Prerequisites

The following tools are required to build, test and lint Regal:

- The latest version of [Go](https://go.dev/doc/install)
- The [golangci-lint](https://golangci-lint.run/usage/install/#local-installation) linter
- [Node](https://nodejs.org/en) in order to run the following tools:
  - The [markdownlint](https://github.com/DavidAnson/markdownlint) linter
  - [dprint](https://github.com/dprint/dprint) for checking JSON, YAML, etc
- The [rq](https://git.sr.ht/~charles/rq) tool. This is used for automating and simplifying many of the tasks outlined
  in this document, and is (ab)used as a Rego-based replacement for Make in this project. Check out the
  [do.rq](https://github.com/open-policy-agent/regal/blob/main/build/do.rq) file to see what that looks like, and for
  documentation on the available tasks. Note that this is a
  [tool dependency](https://www.jvt.me/posts/2025/01/27/go-tools-124/) of Regal and does not need to be installed
  separately.

## Contributing New Rules

If you'd like to contribute a new built-in rule, the simplest way to get started is to run the `regal new rule` command.
This should be done from the top-level directory of the Regal repository, and would look something like this:

```shell
regal new rule --type builtin --category naming --name foo-bar-baz
```

This will create two files in `bundle/regal/rules/naming` (since `naming` was the category) — one for the rule and one
for testing it. The code here will be a pretty simple template, but contains all the required components for a built-in
rule. A good idea for learning more about what's needed is to take a look at some previous PRs adding new rules to
Regal.

### Guiding principles for new built-in rules

- All rules should have succinct, descriptive names which are unique - even across categories
- A rule that misses a few cases is better than no rule at all, but it's good to document any known edge cases
- False positives should however always be avoided
- Add tests for as many cases as you can think of
- Any new rule should have an example violation added in `e2e/testdata/violations/most_violations.rego`
- All the steps for building, testing and linting in this document should pass

If you're struggling with any of the above points, or you're unsure of what to do, no worries! Just say so in your PR,
or ask for advice in the `#regal` channel in the OPA
[Slack](https://slack.openpolicyagent.org).

### Rules Development Workflow

Linter rules use a JSON representation of the AST as input. Use the `regal parse` command pointed at any Rego
file to inspect its AST in this format. It is often more convenient to direct this output to a file, like
`input.json` to browse it in your favorite editor, e.g. `regal parse policy.rego > input.json`.

If you're using VS Code and the [OPA VS Code extension](https://github.com/open-policy-agent/vscode-opa), you may
use the [Code Lens for Evaluation](https://openpolicyagent.org/projects/regal/language-server#code-lenses-evaluation) to directly
evaluate packages and rules using the `input.json` file as input, and see the result directly in your editor on the
line you clicked to evaluate.

As another convenience, any `.rego` file where the first comment in the policy is `# regal eval:use-as-input` will have
the evaluation feature automatically use the AST of the file as input. This allows building queries against the AST of
the policy you're working on, providing an extremely fast feedback loop for developing new rules!

![Use AST of file as input](./assets/lsp/eval_use_as_input.png)

## Contributing to the Language Server

Since the linter is also a core component of the language server, it's likely a good strategy to first understand how
that works, and the suggested rules development workflow described above. That said, working with a *server* is by
nature different from a one-off process. So what would a good workflow for language server development look like?

By default, Regal uses policies and data from its `bundle` directory, which it embeds **at compile time**. This means
that if you make a change in any Rego policy, you'll need to re-run `go build`/`go run` for the change to take effect.
And since you most likely will be testing the language server via an editor, that too will have to be either restarted,
or instructed to reload the language server after each compilation. This is frankly a pretty frustrating development
experience, and even more so when you're used to the fast feedback loop of linter rules development.

While changing any Go code in the language server still requires recompilation, it's now possible to tell the language
server to load its bundle (i.e. all Rego policies and JSON files that Regal depend on) from disk instead of using the
bundle embedded in the binary. By setting the `REGAL_BUNDLE_PATH` environment variable to the path of your Regal
repository's `bundle` directory, the language server will prefer to load its bundle from there instead, falling back to
the embedded bundle only when issues prevent the bundle on disk from being loaded.

When `REGAL_BUNDLE_PATH` is set, any change you make inside the target `bundle` directory will trigger an update of the
bundle, immediately propagating the change to the language server. While obviously less efficient than loading the
bundle only once, this provides a **very** nice language server development experience, where any change you make is
immediately reflected in your editor's UI. Try it out!

### Caveats

- The language server reloads the bundle only on file **save** events, so if you don't see your changes reflected, you
  probably forgot to save the file you changed.
- The `REGAL_BUNDLE_PATH` variable is only read on server initialization, and changing this while the language server
  is running has no effect until the server is restarted.

## Building

Build the `regal` executable simply by running `go build`, or with `rq` installed, by running `build/do.rq build`.

Occasionally you may want to run the `build/do.rq fetch` script. This will
populate the `data` directory with any data necessary for linting (such as the built-in function metadata from OPA).

## Testing

To run all the Rego unit tests, you can run the `regal test` command targeting the `bundle` directory:

```shell
regal test bundle
```

To run all tests — Go and Rego:

```shell
go test ./...
```

Or using `rq`:

```shell
build/do.rq test
```

### E2E tests

End-to-End (E2E) tests assert the behavior of the `regal` binary called with certain configs, and test files.
They expect a `regal` binary either in the top-level directory, or pointed at by `$REGAL_BIN`, and can be run
locally via

```shell
go test -tags e2e ./e2e
```

Alternatively, using `rq`:

```shell
build/do.rq e2e
```

## Linting

Regal uses [golangci-lint](https://golangci-lint.run/) with most linters enabled. In order to check your code, run:

```shell
golangci-lint run ./...
```

Note that for many issues reported, and in particular those related to style, it is often possible to run with the
`--fix` flag to have the issues automatically fixed:

```shell
golangci-lint run --fix ./...
```

In order to ensure consistent formatting in our markdown docs, we use the
[markdownlint](https://github.com/DavidAnson/markdownlint) tool in CI. To run it yourself before submitting a PR,
install it (`brew install markdownlint-cli`) and run:

```shell
markdownlint --config docs/.markdownlint.yaml README.md docs/
```

## Preparing a pull request

Using `rq`, run all the required steps with:

```shell
build/do.rq pr
```

This will run all the formatters, linters and tests. Make sure all of them pass before submitting your PR. If there's
anything you can't figure out, don't hesitate to ask for help in the `#regal` Slack channel (see `Community` below).

## Documentation

The tables on the [rules pages](https://openpolicyagent.org/projects/regal/rules) are
generated from the individual
[rule Markdown files](https://github.com/open-policy-agent/regal/tree/main/docs/rules)
under `docs/rules`.

## Wasm (Experimental)

Build with

```shell
GOOS=wasip1 GOARCH=wasm go build -o regal.wasm .
```

Run with wasmtime regal.wasm and the like:

```shell
$ curl https://wasmtime.dev/install.sh -sSf | bash
# ...
$ wasmtime --version
wasmtime-cli 13.0.0
$ wasmtime --dir $(pwd) regal -- lint bundle
90 files linted. No violations found.
```
