package encoding

import (
	jsoniter "github.com/json-iterator/go"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

func init() {
	jsoniter.RegisterTypeEncoder("ast.Module", &moduleCodec{})
	jsoniter.RegisterTypeEncoder("ast.Package", &packageCodec{})
	jsoniter.RegisterTypeEncoder("ast.Import", &importCodec{})
	jsoniter.RegisterTypeEncoder("ast.Annotations", &annotationsCodec{})
	jsoniter.RegisterTypeEncoder("ast.Rule", &ruleCodec{})
	jsoniter.RegisterTypeEncoder("ast.Head", &headCodec{})
	jsoniter.RegisterTypeEncoder("ast.Body", &bodyCodec{})
	jsoniter.RegisterTypeEncoder("ast.Expr", &exprCodec{})
	jsoniter.RegisterTypeEncoder("ast.Ref", &refCodec{})
	jsoniter.RegisterTypeEncoder("ast.Term", &termCodec{})
	jsoniter.RegisterTypeEncoder("ast.SomeDecl", &someDeclCodec{})
	jsoniter.RegisterTypeEncoder("ast.Every", &everyCodec{})
	jsoniter.RegisterTypeEncoder("ast.With", &withCodec{})
	jsoniter.RegisterTypeEncoder("ast.Comment", &commentCodec{})

	jsoniter.RegisterTypeEncoder("ast.Location", &locationCodec{})
	jsoniter.RegisterTypeEncoder("location.Location", &locationCodec{})

	jsoniter.RegisterTypeEncoder("ast.Array", &arrayCodec{})
	jsoniter.RegisterTypeEncoder("ast.ArrayComprehension", &arrayComprehensionCodec{})
	jsoniter.RegisterTypeEncoder("ast.ObjectComprehension", &objectComprehensionCodec{})
	jsoniter.RegisterTypeEncoder("ast.SetComprehension", &setComprehensionCodec{})
	jsoniter.RegisterTypeEncoder("ast.TemplateString", &templateStringCodec{})

	// special cases as these are not public — see implementation for details
	jsoniter.RegisterTypeEncoder("ast.set", &setCodec{})
	jsoniter.RegisterTypeEncoder("ast.object", &objectCodec{})
}
