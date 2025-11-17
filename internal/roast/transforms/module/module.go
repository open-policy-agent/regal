package module

import (
	"encoding/base64"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

// ToValue converts an AST module to RoAST value representation.
// This is much more efficient than using a JSON encode/decode round trip.
func ToValue(mod *ast.Module) (ast.Value, error) {
	value := ast.NewObject()

	if mod.Package != nil {
		value.Insert(ast.InternedTerm("package"), ast.NewTerm(packageToValue(mod.Package, mod.Annotations)))
	}

	if len(mod.Imports) > 0 {
		imports := make([]*ast.Term, len(mod.Imports))

		for i, imp := range mod.Imports {
			impObj := objectWithLocation(imp.Location)
			impObj.Insert(ast.InternedTerm("path"), termToObjectLoc(imp.Path, true))

			if imp.Alias != "" {
				impObj.Insert(ast.InternedTerm("alias"), ast.InternedTerm(string(imp.Alias)))
			}

			imports[i] = ast.NewTerm(impObj)
		}

		value.Insert(ast.InternedTerm("imports"), ast.ArrayTerm(imports...))
	}

	if len(mod.Rules) > 0 {
		value.Insert(ast.InternedTerm("rules"), ast.ArrayTerm(util.Map(mod.Rules, ruleToObject)...))
	}

	if len(mod.Comments) > 0 {
		comments := make([]*ast.Term, len(mod.Comments))

		for i, comment := range mod.Comments {
			encoded := base64.StdEncoding.EncodeToString(comment.Text)
			comments[i] = ast.ObjectTerm(item("text", ast.InternedTerm(encoded)), locationItem(comment.Location))
		}

		value.Insert(ast.InternedTerm("comments"), ast.ArrayTerm(comments...))
	}

	return value, nil
}

func packageToValue(pkg *ast.Package, annotations []*ast.Annotations) ast.Value {
	value := objectWithLocation(pkg.Location)

	if pkg.Path != nil {
		value.Insert(ast.InternedTerm("path"), pathArray(pkg.Path))
	}

	if len(annotations) > 0 {
		pkgan := make([]*ast.Term, 0, len(annotations))

		for _, a := range annotations {
			if a.Scope != "document" && a.Scope != "rule" {
				pkgan = append(pkgan, ast.NewTerm(annotationsToObject(a)))
			}
		}

		if len(pkgan) > 0 {
			value.Insert(ast.InternedTerm("annotations"), ast.ArrayTerm(pkgan...))
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
	return item("location", ast.InternedTerm(outil.ByteSliceToString(rast.AppendLocation(nil, location))))
}

func termToObjectLoc(term *ast.Term, includeLocation bool) *ast.Term {
	if term == nil {
		return ast.InternedEmptyObject
	}

	var value *ast.Term

	if term.Value != nil {
		if term.Location != nil && includeLocation {
			return ast.ObjectTerm(
				item("type", ast.InternedTerm(ast.ValueName(term.Value))),
				item("value", termValueTerm(term.Value)), // TODO: Interning
				locationItem(term.Location),
			)
		}

		return ast.ObjectTerm(
			item("type", ast.InternedTerm(ast.ValueName(term.Value))),
			item("value", termValueTerm(term.Value)), // TODO: Interning
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
		return ast.ArrayTerm(util.Map(v, termToObject)...)
	case ast.Call:
		return ast.ArrayTerm(util.Map(v, termToObject)...)
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

		items := util.Map(v.Slice(), termToObject)

		return ast.ArrayTerm(items...)
	case *ast.ArrayComprehension:
		return ast.ObjectTerm(item("term", termToObject(v.Term)), item("body", bodyToArray(v.Body)))
	case *ast.SetComprehension:
		return ast.ObjectTerm(item("term", termToObject(v.Term)), item("body", bodyToArray(v.Body)))
	case *ast.ObjectComprehension:
		return ast.ObjectTerm(
			item("key", termToObject(v.Key)),
			item("value", termToObject(v.Value)),
			item("body", bodyToArray(v.Body)),
		)
	}

	return ast.NewTerm(val)
}

// Mostly copied from OPA's private implementation.
func annotationsToObject(a *ast.Annotations) ast.Object {
	if a == nil {
		return nil
	}

	obj := objectWithLocation(a.Location)

	if a.Scope != "" {
		obj.Insert(ast.InternedTerm("scope"), ast.InternedTerm(a.Scope))
	}

	if a.Title != "" {
		obj.Insert(ast.InternedTerm("title"), ast.InternedTerm(a.Title))
	}

	if a.Entrypoint {
		obj.Insert(ast.InternedTerm("entrypoint"), ast.InternedTerm(true))
	}

	if a.Description != "" {
		obj.Insert(ast.InternedTerm("description"), ast.StringTerm(a.Description))
	}

	if len(a.Organizations) > 0 {
		orgs := util.Map(a.Organizations, ast.InternedTerm)
		obj.Insert(ast.InternedTerm("organizations"), ast.ArrayTerm(orgs...))
	}

	if len(a.RelatedResources) > 0 {
		rrs := make([]*ast.Term, 0, len(a.RelatedResources))

		for _, rr := range a.RelatedResources {
			rrObj := ast.NewObject(item("ref", ast.StringTerm(rr.Ref.String())))
			if rr.Description != "" {
				rrObj.Insert(ast.InternedTerm("description"), ast.StringTerm(rr.Description))
			}

			rrs = append(rrs, ast.NewTerm(rrObj))
		}

		obj.Insert(ast.InternedTerm("related_resources"), ast.ArrayTerm(rrs...))
	}

	if len(a.Authors) > 0 {
		as := make([]*ast.Term, 0, len(a.Authors))

		for _, author := range a.Authors {
			aObj := ast.NewObject()
			if author.Name != "" {
				aObj.Insert(ast.InternedTerm("name"), ast.InternedTerm(author.Name))
			}

			if author.Email != "" {
				aObj.Insert(ast.InternedTerm("email"), ast.InternedTerm(author.Email))
			}

			as = append(as, ast.NewTerm(aObj))
		}

		obj.Insert(ast.InternedTerm("authors"), ast.ArrayTerm(as...))
	}

	if len(a.Schemas) > 0 {
		ss := make([]*ast.Term, 0, len(a.Schemas))

		for _, s := range a.Schemas {
			sObj := ast.NewObject()
			if len(s.Path) > 0 {
				sObj.Insert(ast.InternedTerm("path"), ast.NewTerm(refToArray(s.Path)))
			}

			if len(s.Schema) > 0 {
				sObj.Insert(ast.InternedTerm("schema"), ast.NewTerm(refToArray(s.Schema)))
			}

			if s.Definition != nil {
				def, err := ast.InterfaceToValue(s.Definition)
				if err != nil {
					panic(err)
				}

				sObj.Insert(ast.InternedTerm("definition"), ast.NewTerm(def))
			}

			ss = append(ss, ast.NewTerm(sObj))
		}

		obj.Insert(ast.InternedTerm("schemas"), ast.ArrayTerm(ss...))
	}

	if len(a.Custom) > 0 {
		c, err := ast.InterfaceToValue(a.Custom)
		if err != nil {
			panic(err)
		}

		obj.Insert(ast.InternedTerm("custom"), ast.NewTerm(c))
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
	obj := objectWithLocation(rule.Location)

	if len(rule.Annotations) > 0 {
		annotations := make([]*ast.Term, 0, len(rule.Annotations))

		for _, a := range rule.Annotations {
			annotations = append(annotations, ast.NewTerm(annotationsToObject(a)))
		}

		if len(annotations) > 0 {
			obj.Insert(ast.InternedTerm("annotations"), ast.ArrayTerm(annotations...))
		}
	}

	if rule.Default {
		obj.Insert(ast.InternedTerm("default"), ast.InternedTerm(true))
	}

	if rule.Head != nil {
		obj.Insert(ast.InternedTerm("head"), headToObject(rule.Head))
	}

	if !rast.IsBodyGenerated(rule) {
		obj.Insert(ast.InternedTerm("body"), bodyToArray(rule.Body))
	}

	if rule.Else != nil {
		obj.Insert(ast.InternedTerm("else"), ruleToObject(rule.Else))
	}

	return ast.NewTerm(obj)
}

func headToObject(head *ast.Head) *ast.Term {
	obj := objectWithLocation(head.Location)

	if head.Reference != nil {
		obj.Insert(ast.InternedTerm("ref"), termValueTerm(head.Reference))
	}

	if len(head.Args) > 0 {
		obj.Insert(ast.InternedTerm("args"), ast.ArrayTerm(util.Map(head.Args, termToObject)...))
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
			item("target", termToObject(with.Target)),
			item("value", termToObject(with.Value)),
		)
	}

	return ast.ObjectTerm(
		item("target", termToObject(with.Target)),
		item("value", termToObject(with.Value)),
	)
}

func bodyToArray(body ast.Body) *ast.Term {
	exprs := make([]*ast.Term, len(body))

	for i, expr := range body {
		exprObj := objectWithLocation(expr.Location)

		if expr.Negated {
			exprObj.Insert(ast.InternedTerm("negated"), ast.InternedTerm(true))
		}

		if expr.Generated {
			exprObj.Insert(ast.InternedTerm("generated"), ast.InternedTerm(expr.Generated))
		}

		if len(expr.With) > 0 {
			exprObj.Insert(ast.InternedTerm("with"), ast.ArrayTerm(util.Map(expr.With, withToObject)...))
		}

		if expr.Terms != nil {
			switch t := expr.Terms.(type) {
			case *ast.Term:
				insert(exprObj, "terms", termToObject(t))
			case []*ast.Term:
				insert(exprObj, "terms", ast.ArrayTerm(util.Map(t, termToObject)...))
			case *ast.SomeDecl:
				terms := objectWithLocation(t.Location)
				insert(terms, "symbols", ast.ArrayTerm(util.Map(t.Symbols, termToObject)...))
				insert(exprObj, "terms", ast.NewTerm(terms))
			case *ast.Every:
				terms := objectWithLocation(t.Location)
				if t.Key == nil {
					// This is only to replicate roast encoding — we probably shouldn't do this
					insert(terms, "key", ast.InternedNullTerm)
				} else {
					insert(terms, "key", termToObject(t.Key))
				}

				insert(terms, "value", termToObject(t.Value))
				insert(terms, "domain", termToObject(t.Domain))
				insert(terms, "body", bodyToArray(t.Body))
				insert(exprObj, "terms", ast.NewTerm(terms))
			}
		}

		exprs[i] = ast.NewTerm(exprObj)
	}

	return ast.ArrayTerm(exprs...)
}

func objectWithLocation(loc *ast.Location) ast.Object {
	if loc == nil {
		return ast.NewObject()
	}

	return ast.NewObject(locationItem(loc))
}

func item(key string, value *ast.Term) [2]*ast.Term {
	if value == nil {
		return [2]*ast.Term{ast.InternedTerm(key), ast.InternedNullTerm}
	}

	return [2]*ast.Term{ast.InternedTerm(key), value}
}

func insert(obj ast.Object, key string, value *ast.Term) {
	if value == nil {
		return
	}

	obj.Insert(ast.InternedTerm(key), value)
}
