package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func RuleMarshalToFn(enc *jsontext.Encoder, rule *ast.Rule) error {
	write.ObjectStart(enc, rule.Location)

	if len(rule.Annotations) > 0 {
		write.ArrayFieldFn(enc, "annotations", rule.Annotations, AnnotationsMarshalToFn)
	}

	write.WhenTrue(enc, rule.Default, "default")

	if rule.Head != nil {
		write.FieldFn(enc, "head", rule.Head, HeadMarshalToFn)
	}

	if !rast.IsBodyGenerated(rule) {
		write.FieldFn(enc, "body", rule.Body, BodyMarshalToFn)
	}

	if rule.Else != nil {
		write.FieldFn(enc, "else", rule.Else, RuleMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}

func HeadMarshalToFn(enc *jsontext.Encoder, head *ast.Head) error {
	write.ObjectStart(enc, head.Location)

	if head.Reference != nil {
		write.ArrayFieldFn(enc, "ref", head.Reference, TermMarshalToFn)
	}

	if len(head.Args) > 0 {
		write.ArrayFieldFn(enc, "args", head.Args, TermMarshalToFn)
	}

	write.WhenTrue(enc, head.Assign, "assign")

	if head.Key != nil {
		write.FieldFn(enc, "key", head.Key, TermMarshalToFn)
	}

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil && head.Location != nil {
			if head.Value.Location.Row == head.Location.Row && head.Value.Location.Col == head.Location.Col {
				head.Value.Location = nil
			}
		}

		write.FieldFn(enc, "value", head.Value, TermMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}

func BodyMarshalToFn(enc *jsontext.Encoder, body ast.Body) error {
	return write.ArrayToFn(enc, body, ExprMarshalToFn)
}
