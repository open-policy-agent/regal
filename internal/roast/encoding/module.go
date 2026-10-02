package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
	"github.com/open-policy-agent/regal/internal/util"
)

func notDocumentOrRuleScope(a *ast.Annotations) bool {
	return a.Scope != "document" && a.Scope != "rule"
}

func ModuleMarshalToFn(enc *jsontext.Encoder, mod *ast.Module) (err error) {
	enc.WriteToken(jsontext.BeginObject)

	if mod.Package != nil {
		enc.WriteToken(jsontext.String("package"))
		_ = writePackage(enc, mod.Package, util.Filter(mod.Annotations, notDocumentOrRuleScope))
	}

	if len(mod.Imports) > 0 {
		write.ArrayFieldFn(enc, "imports", mod.Imports, ImportMarshalToFn)
	}

	if len(mod.Rules) > 0 {
		write.ArrayFieldFn(enc, "rules", mod.Rules, RuleMarshalToFn)
	}

	if len(mod.Comments) > 0 {
		write.ArrayFieldFn(enc, "comments", mod.Comments, CommentMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}
