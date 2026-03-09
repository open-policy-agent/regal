# Naming Convention

This document outlines the recommended naming conventions when writing Rego for Regal, custom linter rules, or really
any Rego targeting OPA's AST or Regal's RoAST format (which is derived from the former). The purpose of these
conventions is reducing ambiguity by sticking to a consistent set of names, which _always refer to the same thing_.

Regal contributors are encouraged to follow these conventions, but **not** required to. The conventions are here to
support us in writing more consistent Rego for the context of Regal, not act as an obstacle for contributing.

## Compliance Check

You can run `regal lint --enable naming-convention bundle` to check the Regal bundle for compliance against
these naming conventions. This check is not automated, but can be run occasionally to catch and correct inconsistencies
in naming. The list of "allowed" words can and will of course also change over time.

An alternative that we **may** consider for the future is to run this check in CI configured at `warn` level. But while
this could be a friendly nudge, it also risks to clutter our build logs with messages that are mostly of the "notice"
kind.

## Rules and Functions

- Use an underscore prefix for any rule that is internal to a package (e.g., `_count_vars`).

## Variables

- We commonly use three letter abbreviations for variable names related to Rego and AST types, such as `val`, `num`,
  or `ref`. The plural form of such identifiers is naturally the name + `s`, e.g. `vals`, `nums`, `refs`.
- Four letter words, like `rule`, `body`, `term`, `...`, are never abbreviated.
  Consistently sticking to this convention makes for concise but easily readable code, that aligns nicely.
- Use `<name>` + `other` for comparisons and equality checks.
- Use `i`, `j`, `k`, `...` for indices.
- Use `n` for counts, or lengths.
- `_` may be used as a prefix for local variables when (and only when) a name shadows or otherwise conflicts with
  another identifier in the same scope, and there's no better name to use.
- Three or four letter variable names should not be used for anything but known names and types.
- Longer variable names are allowed to describe shapes and state outside of the Rego / AST domai.

### Longer Variable Names

- Use longer and more descriptive names (with words separated by underscores) whenever the "default" names are
  insufficient. All names of length 5 and above are excepted from our naming convention policy, but should still be
  carefully chosen.
- When using longer variable names, it is often preferable to use full words instead of abbreviations, e.g.
  `empty_arrays` over `empty_arrs`.
- When using longer variable names, try to see if any existing name describes the same thing, and if so, use that.

Future versions of this document may include commonly used longer variable names.

### AST Nodes

The following names should be used consistently across Regal for variables representing AST nodes of the given type.

- `node` - The "any" type for AST elements whose type is not specified or known at the place of use.
- `pkg` - An argument or variable representing a package node.
- `imp` - An argument or variable representing an import node.
- `rule` - An argument or variable representing a rule node.
- `ref` - An argument or variable representing a ref node.
- `fun` - An argument or variable representing a function node.
- `head` - An argument or variable representing a rule head node, or occasionally the head of a ref.
- `body` - An argument or variable representing a body node.
- `expr` - An argument or variable representing an expression.
- `args` - The array of terms representing the arguments of a function.
- `arg` - A function argument term. May also be referenced as `term` when that is more appropriate to the context.
- `term` - An argument or variable representing a term node.
- `terms` - An array of terms, as found in for example the `value` of a `ref`.

### JSON Types

- `str` - String
- `num` - Number
- `arr` - Array
- `obj` - Object (where `key`/ `val` are used for object keys and values).
- `set` - Set

(Boolean values and null are almost never stored in variables.)

### AST Types

- `term` - Term
- `rule` - Rule
- `body` - Body
- `call` - Call
- `comp` - Comprehension
- `ref` - Ref
- `set` - Set

### Collections

- `coll` - A variable representing any collection type. Prefer `arr`, `obj`, or `set` when the type is known.
- `seq` - Use to represent values that either arrays or sets.

### Location and Position

- `rows` - An array of rows (line numbers).
- `cols` - An array of columns.
- `text` - The source code text corresponding to a location.
- `line` - The text of a single line of source code.
- `loc` - A location string or object.
- `row` - The row number of a location.
- `col` - The column number of a location.
- `end` - The end position of a location.

### Regal-specific

- `cfg` - A variable representing a Regal configuration object.
- `aggs` - A collection of aggregates
- `agg` - An aggregated item

### Language Server

- `file` - A variable representing a file path.
- `word` - A variable representing a word, such as a variable name, function name, etc.
- `url` - A variable representing a URL.
- `uri` - A variable representing a URI.
- `dir` - A variable representing a directory path.

### Various

- `other` - A variable representing the other item in a comparison or equality check.
- `start` - The start of something.
- `name` - The name of something, such as a rule or variable.
- `link` - A variable representing a link, such as a URL or URI.
- `path` - A variable representing a file or URI path.
- `kind` - The kind of something, such as a rule vs. a type.
- `rest` - The rest of something, such as the remaining terms in a ref after the head.
- `last` - The last item in a collection.
- `next` - The next item in a collection.
- `diff` - A variable representing the difference between two items.
- `len` - The length of something.
- `pos` - A position, such as a cursor position.
- `sub` - A subset.
- `sup` - A superset.
- `lhs` - The left-hand side of a rule or expression.
- `rhs` - The right-hand side of a rule or expression.

## Avoid

- `parts`/`part` — When referring to terms in types like `ref`s or `array`s. Should most often be replaced by
  `terms`, `args`, or whatever is more specific.
- `item` — For anything but array or set items of unspecified type.
