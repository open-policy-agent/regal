# Rego Style Guide rules in Regal

Some notes, and a checklist for tracking, on implementing linter rules based on the
[Rego Style Guide](https://openpolicyagent.org/docs/style-guide).

Rules with a strikethrough are considered to be either impossible, or undesirable,
for Regal to check, and should be targeted by other means.

## General Advice

- [x] ~~Optimize for readability, not performance~~
- [x] Use `opa fmt`
- [ ] Use strict mode
- [ ] Use metadata annotations
- [x] ~~Get to know the built-in functions~~
- [ ] Consider using JSON schemas for type checking

### Notes

Both `opa fmt` and `strict mode` checks can be implemented by tapping into
the corresponding features of OPA, and simply report errors as linter violations.
Why not just use OPA for that? Mainly because we want a single command (and config)
for anything related to code quality in Rego. Rules can be disabled here in favor of
OPA if someone prefers that. Do note that we won't actually _format_ anything,
just as we won't do any other type of remediation — the rule should only check if
the files are formatted, similar to `opa fmt --diff --fail`.

Requiring the use of metadata annotations is doable, but a rule like that would
certainly have to be configurable to be usable. Same goes for JSON schemas.

## Style

- [x] Prefer snake_case for rule names and variables
- [x] Keep line length <= 120 characters

## Rules

- [ ] Use helper rules and functions
- [ ] Use negation to handle undefined
- [x] ~~Consider partial helper rules over comprehensions in rule bodies~~
- [x] Avoid prefixing rules and functions with `get_` or `list_`
- [x] Prefer unconditional assignment in rule head over rule body

### Notes

Three quite complex rules in the top here. While it's not going to be very
scientific, we could try to determine whether helper rules are used to a
satisfying degree by checking the dependencies of rules vs the number of
expressions, and come up with some (configurable) thresholds. "Use negation to
handle undefined" could be very hard to implement, and might be that we
shouldn't. "Consider" partial rules over comprehensions feels impossible to
determine whether the author has done, and both have valid use cases.

## Variables and Data Types

- [x] Use `in` to check for membership
- [x] Prefer some .. in for iteration
- [ ] Use every to express FOR ALL
- [ ] Don't use unification operator for assignment or comparison
- [x] Don't use undeclared variables
- [x] ~~Prefer sets over arrays (where applicable)~~

### Notes

Almost all of these should be doable, with some possibly being quite challenging.
One that could be very hard to implement is the `every` rule, as that would
require us to determine what **other** method was used and that `every` is a
suitable replacement. Except for "Use `in` to check for membership",
none of the above rules can be enforced using the AST alone.

## Functions

- [x] Prefer using arguments over input and data
- [x] Avoid using the last argument for the return value

## Regex

- [x] Use raw strings for regex patterns

### Notes

Can only be done by scanning the original code, as this is lost in the AST.

## Imports

- [x] Use explicit imports for future keywords
- [x] Prefer importing packages over rules and functions
- [x] Avoid importing input

### Notes

Checking for package imports requires a view of all modules. We may assume that
anything not found there are base documents to be provided at runtime.
