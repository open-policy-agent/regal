package encoding

import (
	jsoniter "github.com/json-iterator/go"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

var (
	astModuleCodec              = &moduleCodec{}
	astPackageCodec             = &packageCodec{}
	astImportCodec              = &importCodec{}
	astAnnotationsCodec         = &annotationsCodec{}
	astRuleCodec                = &ruleCodec{}
	astHeadCodec                = &headCodec{}
	astBodyCodec                = &bodyCodec{}
	astExprCodec                = &exprCodec{}
	astRefCodec                 = &refCodec{}
	astTermCodec                = &termCodec{}
	astSomeDeclCodec            = &someDeclCodec{}
	astEveryCodec               = &everyCodec{}
	astWithCodec                = &withCodec{}
	astNotCodec                 = &notCodec{}
	astLogicalAndCodec          = &logicalAndCodec{}
	astLogicalOrCodec           = &logicalOrCodec{}
	astCommentCodec             = &commentCodec{}
	astLocationCodec            = &locationCodec{}
	astArrayCodec               = &arrayCodec{}
	astArrayComprehensionCodec  = &arrayComprehensionCodec{}
	astObjectComprehensionCodec = &objectComprehensionCodec{}
	astSetComprehensionCodec    = &setComprehensionCodec{}
	astTemplateStringCodec      = &templateStringCodec{}
	astNumberCodec              = &numberCodec{}
	// special cases as these are not public — see implementation for details.
	astSetCodec    = &setCodec{}
	astObjectCodec = &objectCodec{}
)

func init() {
	jsoniter.RegisterTypeEncoder("ast.Module", astModuleCodec)
	jsoniter.RegisterTypeEncoder("ast.Package", astPackageCodec)
	jsoniter.RegisterTypeEncoder("ast.Import", astImportCodec)
	jsoniter.RegisterTypeEncoder("ast.Annotations", astAnnotationsCodec)
	jsoniter.RegisterTypeEncoder("ast.Rule", astRuleCodec)
	jsoniter.RegisterTypeEncoder("ast.Head", astHeadCodec)
	jsoniter.RegisterTypeEncoder("ast.Body", astBodyCodec)
	jsoniter.RegisterTypeEncoder("ast.Expr", astExprCodec)
	jsoniter.RegisterTypeEncoder("ast.Ref", astRefCodec)
	jsoniter.RegisterTypeEncoder("ast.Term", astTermCodec)
	jsoniter.RegisterTypeEncoder("ast.SomeDecl", astSomeDeclCodec)
	jsoniter.RegisterTypeEncoder("ast.Every", astEveryCodec)
	jsoniter.RegisterTypeEncoder("ast.With", astWithCodec)
	jsoniter.RegisterTypeEncoder("ast.Not", astNotCodec)
	jsoniter.RegisterTypeEncoder("ast.LogicalAnd", astLogicalAndCodec)
	jsoniter.RegisterTypeEncoder("ast.LogicalOr", astLogicalOrCodec)
	jsoniter.RegisterTypeEncoder("ast.Comment", astCommentCodec)

	jsoniter.RegisterTypeEncoder("ast.Location", astLocationCodec)
	jsoniter.RegisterTypeEncoder("location.Location", astLocationCodec)

	jsoniter.RegisterTypeEncoder("ast.Array", astArrayCodec)
	jsoniter.RegisterTypeEncoder("ast.ArrayComprehension", astArrayComprehensionCodec)
	jsoniter.RegisterTypeEncoder("ast.ObjectComprehension", astObjectComprehensionCodec)
	jsoniter.RegisterTypeEncoder("ast.SetComprehension", astSetComprehensionCodec)
	jsoniter.RegisterTypeEncoder("ast.TemplateString", astTemplateStringCodec)
	jsoniter.RegisterTypeEncoder("ast.Number", astNumberCodec)

	// special cases as these are not public — see implementation for details
	jsoniter.RegisterTypeEncoder("ast.set", astSetCodec)
	jsoniter.RegisterTypeEncoder("ast.object", astObjectCodec)
}
