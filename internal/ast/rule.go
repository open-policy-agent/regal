package ast

import (
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
)

// GetRuleDetail returns a short descriptive string value for a given rule stating
// if the rule is constant, multi-value, single-value etc and the type of the rule's
// value if known.
func GetRuleDetail(rule *ast.Rule, builtins map[string]*ast.Builtin) string {
	if rule.Head.Args != nil {
		return "function" + rule.Head.Args.String()
	}

	if rule.Head.Key != nil && rule.Head.Value == nil {
		return "multi-value rule"
	}

	if rule.Head.Value == nil {
		return ""
	}

	detail := "single-value "

	if rule.Head.Key != nil {
		detail += "map "
	}

	detail += "rule"

	switch v := rule.Head.Value.Value.(type) {
	case ast.Boolean:
		if strings.HasPrefix(rule.Head.Ref()[0].String(), "test_") {
			detail += " (test)"
		} else {
			detail += " (boolean)"
		}
	case ast.Number:
		detail += " (number)"
	case ast.String:
		detail += " (string)"
	case *ast.Array, *ast.ArrayComprehension:
		detail += " (array)"
	case ast.Object, *ast.ObjectComprehension:
		detail += " (object)"
	case ast.Set, *ast.SetComprehension:
		detail += " (set)"
	case ast.Call:
		name := v[0].String()

		if builtin, ok := builtins[name]; ok {
			retType := builtin.Decl.NamedResult().String()

			detail += fmt.Sprintf(" (%s)", simplifyType(retType))
		}
	}

	return detail
}

// IsConstant returns true if the rule is a "constant" rule, i.e.
// one without conditions and scalar value in the head.
func IsConstant(rule *ast.Rule) bool {
	if rule.Head.Value == nil {
		return false
	}

	isScalar := false

	switch rule.Head.Value.Value.(type) {
	case ast.Boolean, ast.Number, ast.String, ast.Null:
		isScalar = true
	}

	return isScalar &&
		rule.Head.Args == nil &&
		rule.Body.Equal(ast.NewBody(ast.NewExpr(ast.BooleanTerm(true)))) &&
		rule.Else == nil
}

// simplifyType removes anything but the base type from the type name.
func simplifyType(name string) string {
	result := name

	if strings.Contains(result, ":") {
		result = result[strings.Index(result, ":")+1:]
	}

	// silence gocritic linter here as strings.Index can in
	// fact *not* return -1 in these cases
	if strings.Contains(result, "[") {
		result = result[:strings.Index(result, "[")] //nolint:gocritic
	}

	if strings.Contains(result, "<") {
		result = result[:strings.Index(result, "<")] //nolint:gocritic
	}

	return strings.TrimSpace(result)
}
