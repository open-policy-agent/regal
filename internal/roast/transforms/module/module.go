package module

import (
	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

// ToValue converts an AST module to RoAST value representation.
// This is much more efficient than using a JSON encode/decode round trip.
func ToValue(mod *ast.Module) (ast.Value, error) {
	value := ast.NewObjectWithCapacity(1 +
		min(1, len(mod.Imports)) +
		min(1, len(mod.Rules)) +
		min(1, len(mod.Comments)),
	)

	if mod.Package != nil {
		rast.Insert(value, "package", ast.NewTerm(packageToValue(mod.Package, mod.Annotations)))
	}

	if len(mod.Imports) > 0 {
		imports := make([]*ast.Term, len(mod.Imports))

		for i, imp := range mod.Imports {
			impObj := objectWithLocationAndCap(imp.Location, 1+min(1, len(imp.Alias)))
			rast.Insert(impObj, "path", termToObjectLoc(imp.Path, true))

			if imp.Alias != "" {
				rast.Insert(impObj, "alias", ast.InternedTerm(string(imp.Alias)))
			}

			imports[i] = ast.NewTerm(impObj)
		}

		rast.Insert(value, "imports", ast.ArrayTerm(imports...))
	}

	if len(mod.Rules) > 0 {
		rast.Insert(value, "rules", ast.ArrayTerm(outil.Map(mod.Rules, ruleToObject)...))
	}

	if len(mod.Comments) > 0 {
		comments := make([]*ast.Term, len(mod.Comments))
		for i, comment := range mod.Comments {
			comments[i] = ast.InternedTerm(outil.ByteSliceToString(rast.AppendLocation(nil, comment.Location)))
		}

		rast.Insert(value, "comments", ast.ArrayTerm(comments...))
	}

	return value, nil
}

func packageToValue(pkg *ast.Package, annotations []*ast.Annotations) ast.Value {
	value := objectWithLocationAndCap(pkg.Location, 1+min(1, len(pkg.Path)+min(1, len(annotations))))

	if pkg.Path != nil {
		rast.Insert(value, "path", pathArray(pkg.Path))
	}

	if len(annotations) > 0 {
		pkgan := make([]*ast.Term, 0, len(annotations))

		for _, a := range annotations {
			if a.Scope != "document" && a.Scope != "rule" {
				pkgan = append(pkgan, ast.NewTerm(annotationsToObject(a)))
			}
		}

		if len(pkgan) > 0 {
			rast.Insert(value, "annotations", ast.ArrayTerm(pkgan...))
		}
	}

	return value
}

func pathArray(terms []*ast.Term) *ast.Term {
	if len(terms) == 0 {
		return ast.InternedEmptyArray
	}

	r := make([]*ast.Term, len(terms))
	for i := range terms {
		r[i] = termToObjectLoc(terms[i], i != 0) // Skip location for the first term (data)
	}

	return ast.ArrayTerm(r...)
}

func locationItem(location *ast.Location) [2]*ast.Term {
	return rast.Item("location", ast.InternedTerm(outil.ByteSliceToString(rast.AppendLocation(nil, location))))
}

func termToObjectLoc(term *ast.Term, includeLocation bool) *ast.Term {
	if term == nil {
		return ast.InternedEmptyObject
	}

	var value *ast.Term

	if term.Value != nil {
		if term.Location != nil && includeLocation {
			return ast.ObjectTerm(
				rast.Item("type", ast.InternedTerm(ast.ValueName(term.Value))),
				rast.Item("value", termValueTerm(term.Value)),
				locationItem(term.Location),
			)
		}

		return ast.ObjectTerm(
			rast.Item("type", ast.InternedTerm(ast.ValueName(term.Value))),
			rast.Item("value", termValueTerm(term.Value)),
		)
	}

	return value
}

func termToObject(term *ast.Term) *ast.Term {
	return termToObjectLoc(term, true)
}

func termValueTerm(val ast.Value) *ast.Term {
	switch v := val.(type) {
	case ast.Var:
		return ast.InternedTerm(string(v))
	case ast.Null:
		return ast.InternedNullTerm
	case ast.Boolean:
		return ast.InternedTerm(bool(v))
	case ast.String:
		return ast.InternedTerm(string(v))
	case ast.Number:
		if i, ok := v.Int(); ok {
			return ast.InternedTerm(i)
		}
	case ast.Ref:
		return ast.ArrayTerm(outil.Map(v, termToObject)...)
	case ast.Call:
		return ast.ArrayTerm(outil.Map(v, termToObject)...)
	case *ast.Array:
		if v.Len() == 0 {
			return ast.InternedEmptyArray
		}

		terms := make([]*ast.Term, 0, v.Len())
		for i := range v.Len() {
			terms = append(terms, termToObject(v.Elem(i)))
		}

		return ast.ArrayTerm(terms...)
	case ast.Object:
		if v.Len() == 0 {
			return ast.InternedEmptyArray
		}

		items := make([]*ast.Term, 0, v.Len())
		v.Foreach(func(k, v *ast.Term) {
			items = append(items, ast.ArrayTerm(termToObject(k), termToObject(v)))
		})

		return ast.ArrayTerm(items...)
	case ast.Set:
		if v.Len() == 0 {
			return ast.InternedEmptyArray
		}

		items := outil.Map(v.Slice(), termToObject)

		return ast.ArrayTerm(items...)
	case *ast.ArrayComprehension:
		return ast.ObjectTerm(rast.Item("term", termToObject(v.Term)), rast.Item("body", bodyToArray(v.Body)))
	case *ast.SetComprehension:
		return ast.ObjectTerm(rast.Item("term", termToObject(v.Term)), rast.Item("body", bodyToArray(v.Body)))
	case *ast.ObjectComprehension:
		return ast.ObjectTerm(
			rast.Item("key", termToObject(v.Key)),
			rast.Item("value", termToObject(v.Value)),
			rast.Item("body", bodyToArray(v.Body)),
		)
	case *ast.TemplateString:
		parts := make([]*ast.Term, 0, len(v.Parts))
		for _, part := range v.Parts {
			switch p := part.(type) {
			case *ast.Term:
				parts = append(parts, termToObject(p))
			case *ast.Expr:
				exprObj := objectWithLocationAndCap(p.Location, 2)
				if p.Terms != nil {
					switch t := p.Terms.(type) {
					case *ast.Term:
						rast.Insert(exprObj, "terms", termToObject(t))
					case []*ast.Term:
						rast.Insert(exprObj, "terms", ast.ArrayTerm(outil.Map(t, termToObject)...))
					}
				}
				// Mark expression as part of a template string as interpolated, as some linter rules
				// apply differently or not at all in that context, like e.g. unassigned-return-value
				rast.Insert(exprObj, "interpolated", ast.InternedBooleanTrue)
				parts = append(parts, ast.NewTerm(exprObj))
			}
		}

		if v.MultiLine {
			return ast.ObjectTerm(
				rast.Item("parts", ast.ArrayTerm(parts...)),
				rast.Item("multi_line", ast.InternedBooleanTrue),
			)
		}

		return ast.ObjectTerm(rast.Item("parts", ast.ArrayTerm(parts...)))
	}

	return ast.NewTerm(val)
}

// Mostly copied from OPA's private implementation.
func annotationsToObject(a *ast.Annotations) ast.Object {
	if a == nil {
		return nil
	}

	preAlloc := 0
	if a.Entrypoint {
		preAlloc++
	}

	preAlloc += min(1, len(a.Scope)) + min(1, len(a.Title)) + min(1, len(a.Description)) +
		min(1, len(a.Organizations)) + min(1, len(a.RelatedResources)) + min(1, len(a.Authors)) +
		min(1, len(a.Schemas)) + min(1, len(a.Custom))

	obj := objectWithLocationAndCap(a.Location, preAlloc)

	if a.Scope != "" {
		rast.Insert(obj, "scope", ast.InternedTerm(a.Scope))
	}

	if a.Title != "" {
		rast.Insert(obj, "title", ast.InternedTerm(a.Title))
	}

	if a.Entrypoint {
		rast.Insert(obj, "entrypoint", ast.InternedBooleanTrue)
	}

	if a.Description != "" {
		rast.Insert(obj, "description", ast.StringTerm(a.Description))
	}

	if len(a.Organizations) > 0 {
		rast.Insert(obj, "organizations", ast.ArrayTerm(outil.Map(a.Organizations, ast.InternedTerm)...))
	}

	if len(a.RelatedResources) > 0 {
		rrs := make([]*ast.Term, 0, len(a.RelatedResources))

		for _, rr := range a.RelatedResources {
			rrObj := ast.NewObject(rast.Item("ref", ast.StringTerm(rr.Ref.String())))
			if rr.Description != "" {
				rast.Insert(rrObj, "description", ast.StringTerm(rr.Description))
			}

			rrs = append(rrs, ast.NewTerm(rrObj))
		}

		rast.Insert(obj, "related_resources", ast.ArrayTerm(rrs...))
	}

	if len(a.Authors) > 0 {
		as := make([]*ast.Term, 0, len(a.Authors))

		for _, author := range a.Authors {
			aObj := ast.NewObjectWithCapacity(min(1, len(author.Name)) + min(1, len(author.Email)))
			if author.Name != "" {
				rast.Insert(aObj, "name", ast.InternedTerm(author.Name))
			}

			if author.Email != "" {
				rast.Insert(aObj, "email", ast.InternedTerm(author.Email))
			}

			as = append(as, ast.NewTerm(aObj))
		}

		rast.Insert(obj, "authors", ast.ArrayTerm(as...))
	}

	if len(a.Schemas) > 0 {
		ss := make([]*ast.Term, 0, len(a.Schemas))

		for _, s := range a.Schemas {
			preAlloc := 0
			if s.Definition != nil {
				preAlloc++
			}

			sObj := ast.NewObjectWithCapacity(preAlloc + min(1, len(s.Path)) + min(1, len(s.Schema)))
			if len(s.Path) > 0 {
				rast.Insert(sObj, "path", ast.NewTerm(refToArray(s.Path)))
			}

			if len(s.Schema) > 0 {
				rast.Insert(sObj, "schema", ast.NewTerm(refToArray(s.Schema)))
			}

			if s.Definition != nil {
				def, err := ast.InterfaceToValue(s.Definition)
				if err != nil {
					panic(err)
				}

				rast.Insert(sObj, "definition", ast.NewTerm(def))
			}

			ss = append(ss, ast.NewTerm(sObj))
		}

		rast.Insert(obj, "schemas", ast.ArrayTerm(ss...))
	}

	if len(a.Custom) > 0 {
		c, err := ast.InterfaceToValue(a.Custom)
		if err != nil {
			panic(err)
		}

		rast.Insert(obj, "custom", ast.NewTerm(c))
	}

	return obj
}

func refToArray(ref ast.Ref) *ast.Array {
	terms := make([]*ast.Term, 0, len(ref))

	for _, term := range ref {
		if _, ok := term.Value.(ast.String); ok {
			terms = append(terms, term)
		} else {
			terms = append(terms, ast.InternedTerm(term.Value.String()))
		}
	}

	return ast.NewArray(terms...)
}

func ruleToObject(rule *ast.Rule) *ast.Term {
	generated := rast.IsBodyGenerated(rule)
	preAlloc := util.BoolToInt(rule.Default) +
		util.BoolToInt(rule.Head != nil) +
		util.BoolToInt(!generated) +
		util.BoolToInt(rule.Else != nil)

	obj := objectWithLocationAndCap(rule.Location, preAlloc+min(1, len(rule.Annotations)))

	if len(rule.Annotations) > 0 {
		annotations := make([]*ast.Term, 0, len(rule.Annotations))

		for _, a := range rule.Annotations {
			annotations = append(annotations, ast.NewTerm(annotationsToObject(a)))
		}

		if len(annotations) > 0 {
			rast.Insert(obj, "annotations", ast.ArrayTerm(annotations...))
		}
	}

	if rule.Default {
		rast.Insert(obj, "default", ast.InternedTerm(true))
	}

	if rule.Head != nil {
		rast.Insert(obj, "head", headToObject(rule.Head))
	}

	if !generated {
		rast.Insert(obj, "body", bodyToArray(rule.Body))
	}

	if rule.Else != nil {
		rast.Insert(obj, "else", ruleToObject(rule.Else))
	}

	return ast.NewTerm(obj)
}

func headToObject(head *ast.Head) *ast.Term {
	preAlloc := util.BoolToInt(head.Reference != nil) +
		util.BoolToInt(len(head.Args) > 0) +
		util.BoolToInt(head.Assign) +
		util.BoolToInt(head.Key != nil) +
		util.BoolToInt(head.Value != nil)

	obj := objectWithLocationAndCap(head.Location, preAlloc)

	if head.Reference != nil {
		obj.Insert(ast.InternedTerm("ref"), termValueTerm(head.Reference))
	}

	if len(head.Args) > 0 {
		obj.Insert(ast.InternedTerm("args"), ast.ArrayTerm(outil.Map(head.Args, termToObject)...))
	}

	if head.Assign {
		obj.Insert(ast.InternedTerm("assign"), ast.InternedTerm(true))
	}

	if head.Key != nil {
		obj.Insert(ast.InternedTerm("key"), termToObject(head.Key))
	}

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil && head.Location != nil {
			if head.Value.Location.Row == head.Location.Row && head.Value.Location.Col == head.Location.Col {
				head.Value.Location = nil
			}
		}

		obj.Insert(ast.InternedTerm("value"), termToObject(head.Value))
	}

	return ast.NewTerm(obj)
}

func withToObject(with *ast.With) *ast.Term {
	if with.Location != nil {
		return ast.ObjectTerm(
			locationItem(with.Location),
			rast.Item("target", termToObject(with.Target)),
			rast.Item("value", termToObject(with.Value)),
		)
	}

	return ast.ObjectTerm(rast.Item("target", termToObject(with.Target)), rast.Item("value", termToObject(with.Value)))
}

func bodyToArray(body ast.Body) *ast.Term {
	exprs := make([]*ast.Term, len(body))

	for i, expr := range body {
		preAlloc := util.BoolToInt(expr.Negated) +
			util.BoolToInt(expr.Generated) +
			util.BoolToInt(len(expr.With) > 0) +
			util.BoolToInt(expr.Terms != nil)

		exprObj := objectWithLocationAndCap(expr.Location, preAlloc)

		if expr.Negated {
			rast.Insert(exprObj, "negated", ast.InternedBooleanTrue)
		}

		if expr.Generated {
			rast.Insert(exprObj, "generated", ast.InternedBooleanTrue)
		}

		if len(expr.With) > 0 {
			rast.Insert(exprObj, "with", ast.ArrayTerm(outil.Map(expr.With, withToObject)...))
		}

		if expr.Terms != nil {
			switch t := expr.Terms.(type) {
			case *ast.Term:
				rast.Insert(exprObj, "terms", termToObject(t))
			case []*ast.Term:
				rast.Insert(exprObj, "terms", ast.ArrayTerm(outil.Map(t, termToObject)...))
			case *ast.SomeDecl:
				terms := objectWithLocationAndCap(t.Location, 2)
				rast.Insert(terms, "symbols", ast.ArrayTerm(outil.Map(t.Symbols, termToObject)...))
				rast.Insert(exprObj, "terms", ast.NewTerm(terms))
			case *ast.Every:
				terms := objectWithLocationAndCap(t.Location, 4+util.BoolToInt(t.Key != nil))
				if t.Key != nil {
					rast.Insert(terms, "key", termToObject(t.Key))
				}

				rast.Insert(terms, "value", termToObject(t.Value))
				rast.Insert(terms, "domain", termToObject(t.Domain))
				rast.Insert(terms, "body", bodyToArray(t.Body))
				rast.Insert(exprObj, "terms", ast.NewTerm(terms))
			case *ast.Not:
				terms := objectWithLocationAndCap(t.Location, 3+util.BoolToInt(t.ExplicitBody))
				rast.Insert(terms, "type", ast.InternedTerm("not"))
				rast.Insert(terms, "body", bodyToArray(t.Body))
				rast.Insert(exprObj, "terms", ast.NewTerm(terms))

				if t.ExplicitBody {
					rast.Insert(terms, "explicit_body", ast.InternedTerm(true))
				}
			case *ast.LogicalAnd:
				rast.Insert(exprObj, "terms", logicalToTerm("and", t.Location, t.Lhs, t.Rhs, t.ExplicitLhs, t.ExplicitRhs))
			case *ast.LogicalOr:
				rast.Insert(exprObj, "terms", logicalToTerm("or", t.Location, t.Lhs, t.Rhs, t.ExplicitLhs, t.ExplicitRhs))
			}
		}

		exprs[i] = ast.NewTerm(exprObj)
	}

	return ast.ArrayTerm(exprs...)
}

// logicalToTerm converts an `and`/`or` expression, where explicit_lhs/explicit_rhs mark brace enclosed operands.
func logicalToTerm(op string, loc *ast.Location, lhs, rhs ast.Body, explicitLhs, explicitRhs bool) *ast.Term {
	terms := objectWithLocationAndCap(loc, 3+util.BoolToInt(explicitLhs)+util.BoolToInt(explicitRhs))

	rast.Insert(terms, "type", ast.InternedTerm(op))

	if explicitLhs {
		rast.Insert(terms, "explicit_lhs", ast.InternedBooleanTrue)
	}

	if explicitRhs {
		rast.Insert(terms, "explicit_rhs", ast.InternedBooleanTrue)
	}

	return ast.NewTerm(rast.Insert(rast.Insert(terms, "lhs", bodyToArray(lhs)), "rhs", bodyToArray(rhs)))
}

func objectWithLocationAndCap(loc *ast.Location, c int) ast.Object {
	if loc == nil {
		return ast.NewObjectWithCapacity(c)
	}

	trm := ast.InternedTerm(outil.ByteSliceToString(rast.AppendLocation(nil, loc)))

	return rast.Insert(ast.NewObjectWithCapacity(c+1), "location", trm)
}
