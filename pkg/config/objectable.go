package config

import (
	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

type objectable interface {
	toObject() ast.Object
}

func mapToObject[T objectable](items map[string]T) ast.Object {
	obj := ast.NewObjectWithCapacity(len(items))
	for name, item := range items {
		rast.Insert(obj, name, ast.NewTerm(item.toObject()))
	}

	return obj
}

func (c Config) ToValue() ast.Value {
	return c.toObject()
}

func (c Config) toObject() ast.Object {
	obj := ast.NewObjectWithCapacity(min(1, len(c.Rules)) +
		util.BoolToInt(c.Capabilities != nil) +
		util.BoolToInt(!c.Features.IsZero()) +
		min(1, len(c.Ignore.Files)) +
		util.BoolToInt(c.Project != nil) +
		util.BoolToInt(c.CapabilitiesURL != ""),
	)

	if len(c.Rules) > 0 {
		rast.Insert(obj, "rules", ast.NewTerm(mapToObject(c.Rules)))
	}

	if c.Capabilities != nil {
		rast.Insert(obj, "capabilities", ast.NewTerm(c.Capabilities.toObject()))
	}

	if !c.Features.IsZero() {
		rast.Insert(obj, "features", ast.NewTerm(c.Features.toObject()))
	}

	if len(c.Ignore.Files) > 0 {
		rast.Insert(obj, "ignore", ast.NewTerm(c.Ignore.toObject()))
	}

	if c.Project != nil {
		rast.Insert(obj, "project", ast.NewTerm(c.Project.toObject()))
	}

	if c.CapabilitiesURL != "" {
		rast.Insert(obj, "capabilities_url", ast.InternedTerm(c.CapabilitiesURL))
	}

	return obj
}

func (rule Rule) toObject() ast.Object {
	obj := ast.NewObject(rast.Item(keyLevel, ast.InternedTerm(rule.Level)))

	if rule.Ignore != nil && len(rule.Ignore.Files) != 0 {
		obj.Insert(ast.InternedTerm(keyIgnore), ast.ObjectTerm(rast.Item("files", rast.ArrayTerm(rule.Ignore.Files...))))
	}

	for key, val := range rule.Extra {
		if value, err := transform.AnyToValue(val); err == nil {
			obj.Insert(ast.InternedTerm(key), ast.NewTerm(value))
		}
	}

	return obj
}

func (i Ignore) toObject() ast.Object {
	return ast.NewObject(rast.Item("files", rast.ArrayTerm(i.Files...)))
}

func (c Category) toObject() ast.Object {
	return mapToObject(c)
}

func (p *Project) toObject() ast.Object {
	n := util.BoolToInt(p.Roots != nil)
	if p.Roots != nil {
		n += len(*p.Roots)
	}

	obj := ast.NewObjectWithCapacity(n)

	if p.Roots != nil {
		rootsArr := make([]*ast.Term, len(*p.Roots))
		for i, root := range *p.Roots {
			rootsArr[i] = ast.NewTerm(root.toObject())
		}

		obj.Insert(ast.InternedTerm("roots"), ast.ArrayTerm(rootsArr...))
	}

	if p.RegoVersion != nil {
		obj.Insert(ast.InternedTerm("rego_version"), ast.InternedTerm(*p.RegoVersion))
	}

	return obj
}

func (f *Features) toObject() ast.Object {
	obj := ast.NewObject()

	if f.Remote != nil {
		remoteObj := ast.NewObject(rast.Item("check-version", ast.InternedTerm(f.Remote.CheckVersion)))
		obj.Insert(ast.InternedTerm("remote"), ast.NewTerm(remoteObj))
	}

	return obj
}

func (r Root) toObject() ast.Object {
	if r.RegoVersion != nil {
		return ast.NewObject(
			rast.Item("path", ast.InternedTerm(r.Path)),
			rast.Item("rego_version", ast.InternedTerm(*r.RegoVersion)),
		)
	}

	return ast.NewObject(rast.Item("path", ast.InternedTerm(r.Path)))
}

func (d Decl) toObject() ast.Object {
	return ast.NewObject(
		rast.Item("result", ast.InternedTerm(d.Result)),
		rast.Item("args", rast.ArrayTerm(d.Args...)),
	)
}

func (b *Builtin) toObject() ast.Object {
	return ast.NewObject(rast.Item("decl", ast.NewTerm(b.Decl.toObject())))
}

func (c *Capabilities) toObject() ast.Object {
	return ast.NewObject(
		rast.Item("builtins", ast.NewTerm(mapToObject(c.Builtins))),
		rast.Item("future_keywords", rast.ArrayTerm(c.FutureKeywords...)),
		rast.Item("features", rast.ArrayTerm(c.Features...)),
	)
}
