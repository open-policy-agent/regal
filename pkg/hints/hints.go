package hints

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

//nolint:lll
var patterns = map[string]*regexp.Regexp{
	`eval-conflict-error/complete-rules-must-not-produce-multiple-outputs`: regexp.MustCompile(`^eval_conflict_error: complete rules must not produce multiple outputs$`),
	`eval-conflict-error/object-keys-must-be-unique`:                       regexp.MustCompile(`^object insert conflict$|^eval_conflict_error: object keys must be unique$`),
	`rego-unsafe-var-error/var-name-is-unsafe`:                             regexp.MustCompile(`^rego_unsafe_var_error: var .* is unsafe$`),
	`rego-recursion-error/rule-name-is-recursive`:                          regexp.MustCompile(`^rego_recursion_error: rule .* is recursive:`),
	`rego-parse-error/var-cannot-be-used-for-rule-name`:                    regexp.MustCompile(`^rego_parse_error: var cannot be used for rule name$`),
	`rego-type-error/conflicting-rules-name-found`:                         regexp.MustCompile(`^rego_type_error: conflicting rules .* found$`),
	`rego-type-error/match-error`:                                          regexp.MustCompile(`^rego_type_error: match error`),
	`rego-type-error/arity-mismatch`:                                       regexp.MustCompile(`^rego_type_error: .*: arity mismatch`),
	`rego-type-error/function-has-arity-got-argument`:                      regexp.MustCompile(`^rego_type_error: function .* has arity [0-9]+, got [0-9]+ arguments?$`),
	`rego-compile-error/assigned-var-name-unused`:                          regexp.MustCompile(`^rego_compile_error: assigned var .* unused$`),
	`rego-parse-error/unexpected-assign-token`:                             regexp.MustCompile(`^rego_parse_error: unexpected assign token:`),
	`rego-parse-error/unexpected-identifier-token`:                         regexp.MustCompile(`^rego_parse_error: unexpected identifier token:`),
	`rego-parse-error/unexpected-left-curly-token`:                         regexp.MustCompile(`^rego_parse_error: unexpected { token:`),
	`rego-parse-error/unexpected-right-curly-token`:                        regexp.MustCompile(`^rego_parse_error: unexpected } token`),
	`rego-parse-error/unexpected-name-keyword`:                             regexp.MustCompile(`^rego_parse_error: unexpected .* keyword:`),
	`rego-parse-error/unexpected-string-token`:                             regexp.MustCompile(`^rego_parse_error: unexpected string token:`),
	`rego-type-error/multiple-default-rules-name-found`:                    regexp.MustCompile(`^rego_type_error: multiple default rules .* found`),
}

// GetForError will return the key of any matched, documented, error
// for a given Go error, or empty string if none was found.
func GetForError(e error) string {
	value := reflect.Indirect(reflect.ValueOf(unwrapToFP(e)))
	switch value.Kind() {
	case reflect.Slice:
		for _, value := range value.Seq2() {
			if value = reflect.Indirect(value); value.Kind() == reflect.Struct {
				if key := matchMessage(value); key != "" {
					return key
				}
			}
		}
	case reflect.Struct:
		return matchMessage(value)
	}

	return ""
}

func matchMessage(value reflect.Value) string {
	var msg string

	for field, value := range value.Fields() {
		switch jsonTag := field.Tag.Get("json"); jsonTag {
		case "code":
			msg = value.String() + ": " + msg
		case "message":
			msg += value.String()
		}
	}

	if msg = strings.TrimSuffix(msg, ": "); msg != "" {
		for path, r := range patterns {
			if r.MatchString(msg) {
				return path
			}
		}
	}

	return ""
}

func unwrapToFP(e error) error {
	if w := errors.Unwrap(e); w != nil {
		return unwrapToFP(w)
	}

	return e
}
